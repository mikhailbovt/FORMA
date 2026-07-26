package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/mikhailbovt/FORMA/apps/api/internal/ai"
	"github.com/mikhailbovt/FORMA/apps/api/internal/quality"
	"github.com/mikhailbovt/FORMA/apps/api/internal/resume"
	"github.com/go-chi/chi/v5/middleware"
)

const aiSessionCookie = "forma_ai_session"

type sessionInput struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"api_key,omitempty"`
	BaseURL  string `json:"base_url,omitempty"`
}

type sessionView struct {
	Configured bool      `json:"configured"`
	Provider   string    `json:"provider,omitempty"`
	Model      string    `json:"model,omitempty"`
	HasAPIKey  bool      `json:"has_api_key"`
	BaseURL    string    `json:"base_url,omitempty"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
}

type reviewResponse struct {
	Quality quality.Evaluation `json:"quality"`
	AI      ai.ReviewResult    `json:"ai"`
}

func (api *API) listProviders(writer http.ResponseWriter, _ *http.Request) {
	writeData(writer, http.StatusOK, map[string]any{"items": ai.Providers()})
}

func (api *API) getAISession(writer http.ResponseWriter, request *http.Request) {
	_, session, ok := api.aiSession(request)
	if !ok {
		writeData(writer, http.StatusOK, sessionView{Configured: false})
		return
	}
	writeData(writer, http.StatusOK, sessionView{
		Configured: true,
		Provider:   session.Provider,
		Model:      session.Model,
		HasAPIKey:  session.APIKey != "",
		BaseURL:    session.BaseURL,
		ExpiresAt:  session.ExpiresAt,
	})
}

func (api *API) putAISession(writer http.ResponseWriter, request *http.Request) {
	var input sessionInput
	if err := decodeJSON(writer, request, &input); err != nil {
		writeRequestError(writer, request, err)
		return
	}
	oldID, oldSession, hasOldSession := api.aiSession(request)
	if input.APIKey == "" && hasOldSession && oldSession.Provider == input.Provider {
		input.APIKey = oldSession.APIKey
	}
	session, err := ai.NormalizeSession(ai.Session{
		Provider: input.Provider, Model: input.Model, APIKey: input.APIKey, BaseURL: input.BaseURL,
	})
	if err != nil {
		writeError(writer, request, http.StatusUnprocessableEntity, "validation_error", err.Error(), nil)
		return
	}
	newID, stored, err := api.sessions.Put(session)
	if err != nil {
		api.internalError(writer, request, "create AI session", err)
		return
	}
	if hasOldSession {
		api.sessions.Delete(oldID)
	}
	api.setAISessionCookie(writer, newID, stored.ExpiresAt)
	writeData(writer, http.StatusOK, sessionView{
		Configured: true,
		Provider:   stored.Provider,
		Model:      stored.Model,
		HasAPIKey:  stored.APIKey != "",
		BaseURL:    stored.BaseURL,
		ExpiresAt:  stored.ExpiresAt,
	})
}

func (api *API) deleteAISession(writer http.ResponseWriter, request *http.Request) {
	if id, _, ok := api.aiSession(request); ok {
		api.sessions.Delete(id)
	}
	http.SetCookie(writer, &http.Cookie{
		Name: aiSessionCookie, Value: "", Path: "/api/v1/ai", HttpOnly: true, Secure: api.cookieSecure,
		SameSite: http.SameSiteStrictMode, MaxAge: -1, Expires: time.Unix(1, 0).UTC(),
	})
	writer.WriteHeader(http.StatusNoContent)
}

func (api *API) reviewResume(writer http.ResponseWriter, request *http.Request) {
	_, session, ok := api.aiSession(request)
	if !ok {
		writeError(writer, request, http.StatusConflict, "ai_not_configured", "Configure an AI provider before running a review", nil)
		return
	}
	var input ai.ReviewRequest
	if err := decodeJSON(writer, request, &input); err != nil {
		writeRequestError(writer, request, err)
		return
	}
	var document resume.Document
	if err := json.Unmarshal(input.Resume, &document); err != nil {
		writeError(writer, request, http.StatusUnprocessableEntity, "validation_error", "resume must be a valid resume document", nil)
		return
	}
	base, err := quality.Evaluate(document)
	if err != nil {
		api.internalError(writer, request, "evaluate resume quality", err)
		return
	}
	sanitized, err := ai.SanitizeJSON(input.Resume)
	if err != nil {
		writeError(writer, request, http.StatusUnprocessableEntity, "validation_error", err.Error(), nil)
		return
	}
	ctx, cancel := context.WithTimeout(request.Context(), 65*time.Second)
	defer cancel()
	result, err := api.ai.Review(ctx, session, input)
	if err != nil {
		api.aiError(writer, request, "AI review", err)
		return
	}
	combined := quality.ApplySemanticWithContext(base, result.Assessments, string(sanitized), quality.SemanticContext{
		TargetRole:     input.TargetRole,
		JobDescription: input.JobDescription,
	})
	writeData(writer, http.StatusOK, reviewResponse{Quality: combined, AI: result})
}

func (api *API) rewriteText(writer http.ResponseWriter, request *http.Request) {
	_, session, ok := api.aiSession(request)
	if !ok {
		writeError(writer, request, http.StatusConflict, "ai_not_configured", "Configure an AI provider before rewriting text", nil)
		return
	}
	var input ai.RewriteRequest
	if err := decodeJSON(writer, request, &input); err != nil {
		writeRequestError(writer, request, err)
		return
	}
	ctx, cancel := context.WithTimeout(request.Context(), 65*time.Second)
	defer cancel()
	result, err := api.ai.Rewrite(ctx, session, input)
	if err != nil {
		api.aiError(writer, request, "AI rewrite", err)
		return
	}
	writeData(writer, http.StatusOK, result)
}

func (api *API) aiSession(request *http.Request) (string, ai.Session, bool) {
	cookie, err := request.Cookie(aiSessionCookie)
	if err != nil {
		return "", ai.Session{}, false
	}
	session, ok := api.sessions.Get(cookie.Value)
	return cookie.Value, session, ok
}

func (api *API) setAISessionCookie(writer http.ResponseWriter, id string, expiresAt time.Time) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 1 {
		maxAge = 1
	}
	http.SetCookie(writer, &http.Cookie{
		Name: aiSessionCookie, Value: id, Path: "/api/v1/ai", HttpOnly: true, Secure: api.cookieSecure,
		SameSite: http.SameSiteStrictMode, MaxAge: maxAge, Expires: expiresAt,
	})
}

func (api *API) aiError(writer http.ResponseWriter, request *http.Request, operation string, err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		writeError(writer, request, http.StatusGatewayTimeout, "provider_timeout", "The AI provider timed out", nil)
		return
	}
	if errors.Is(err, ai.ErrInvalidOutput) {
		api.logger.Warn(operation+" returned invalid output", "request_id", middleware.GetReqID(request.Context()), "error", err)
		writeError(writer, request, http.StatusBadGateway, "invalid_ai_output", "The AI provider returned an unsafe or invalid result", nil)
		return
	}
	// Prompt construction errors are caused by invalid or privacy-sensitive input.
	if errors.Is(err, context.Canceled) {
		return
	}
	if isInputError(err) {
		writeError(writer, request, http.StatusUnprocessableEntity, "validation_error", err.Error(), nil)
		return
	}
	api.logger.Warn(operation+" provider failure", "request_id", middleware.GetReqID(request.Context()), "error", err)
	writeError(writer, request, http.StatusBadGateway, "provider_error", "The AI provider could not complete the request", nil)
}

func isInputError(err error) bool {
	message := err.Error()
	for _, prefix := range []string{"resume ", "text ", "personal/contact", "sanitize context"} {
		if len(message) >= len(prefix) && message[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
