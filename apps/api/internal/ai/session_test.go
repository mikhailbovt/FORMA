package ai

import (
	"testing"
	"time"
)

func TestSessionStoreExpiresAndDeletesKeys(t *testing.T) {
	store := NewSessionStore(15 * time.Millisecond)
	defer store.Close()
	id, _, err := store.Put(Session{Provider: "openai", Model: "model", APIKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if session, ok := store.Get(id); !ok || session.APIKey != "secret" {
		t.Fatal("stored session not found")
	}
	time.Sleep(25 * time.Millisecond)
	if _, ok := store.Get(id); ok {
		t.Fatal("expired session still available")
	}
}
