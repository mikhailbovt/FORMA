package ai

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

type Session struct {
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	APIKey    string    `json:"-"`
	BaseURL   string    `json:"base_url,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
	ttl      time.Duration
	stop     chan struct{}
	done     chan struct{}
}

func NewSessionStore(ttl time.Duration) *SessionStore {
	store := &SessionStore{
		sessions: make(map[string]Session),
		ttl:      ttl,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
	go store.cleanup()
	return store
}

func (s *SessionStore) Put(session Session) (string, Session, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", Session{}, err
	}
	id := base64.RawURLEncoding.EncodeToString(bytes)
	session.ExpiresAt = time.Now().Add(s.ttl).UTC()
	s.mu.Lock()
	s.sessions[id] = session
	s.mu.Unlock()
	return id, session, nil
}

func (s *SessionStore) Get(id string) (Session, bool) {
	if id == "" {
		return Session{}, false
	}
	s.mu.RLock()
	session, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok {
		return Session{}, false
	}
	if time.Now().After(session.ExpiresAt) {
		s.Delete(id)
		return Session{}, false
	}
	return session, true
}

func (s *SessionStore) Delete(id string) {
	if id == "" {
		return
	}
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

func (s *SessionStore) Close() {
	close(s.stop)
	<-s.done
	s.mu.Lock()
	clear(s.sessions)
	s.mu.Unlock()
}

func (s *SessionStore) cleanup() {
	defer close(s.done)
	interval := s.ttl / 2
	if interval > time.Minute {
		interval = time.Minute
	}
	if interval < time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			s.mu.Lock()
			for id, session := range s.sessions {
				if now.After(session.ExpiresAt) {
					delete(s.sessions, id)
				}
			}
			s.mu.Unlock()
		case <-s.stop:
			return
		}
	}
}
