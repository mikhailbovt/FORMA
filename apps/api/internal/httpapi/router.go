package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/mikhailbovt/FORMA/apps/api/internal/ai"
	"github.com/mikhailbovt/FORMA/apps/api/internal/importer"
	"github.com/mikhailbovt/FORMA/apps/api/internal/resume"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type healthChecker interface {
	Ping(context.Context) error
}

type aiService interface {
	Review(context.Context, ai.Session, ai.ReviewRequest) (ai.ReviewResult, error)
	Rewrite(context.Context, ai.Session, ai.RewriteRequest) (ai.RewriteResult, error)
}

type Dependencies struct {
	Logger       *slog.Logger
	DB           healthChecker
	Resumes      resume.Store
	Sessions     *ai.SessionStore
	AI           aiService
	CORSOrigin   string
	CookieSecure bool
	MaxBodyBytes int64
}

type API struct {
	logger       *slog.Logger
	db           healthChecker
	resumes      resume.Store
	sessions     *ai.SessionStore
	ai           aiService
	cookieSecure bool
}

func New(dependencies Dependencies) http.Handler {
	api := &API{
		logger:       dependencies.Logger,
		db:           dependencies.DB,
		resumes:      dependencies.Resumes,
		sessions:     dependencies.Sessions,
		ai:           dependencies.AI,
		cookieSecure: dependencies.CookieSecure,
	}
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(cors(dependencies.CORSOrigin))
	router.Use(securityHeaders)
	router.Use(requestLogger(dependencies.Logger))
	router.Use(recoverer(dependencies.Logger))

	router.Route("/api/v1", func(router chi.Router) {
		router.With(bodyLimit(importer.MaxUploadBytes)).Post("/imports/preview", api.previewImport)
		router.Group(func(router chi.Router) {
			router.Use(bodyLimit(dependencies.MaxBodyBytes))
			router.Get("/health", api.health)
			router.Post("/quality/evaluate", api.evaluateResumeQuality)
			router.Route("/resumes", func(router chi.Router) {
				router.Get("/", api.listResumes)
				router.Post("/", api.createResume)
				router.Route("/{resumeID}", func(router chi.Router) {
					router.Get("/", api.getResume)
					router.Put("/", api.updateResume)
					router.Delete("/", api.deleteResume)
					router.Post("/duplicate", api.duplicateResume)
				})
			})
			router.Route("/ai", func(router chi.Router) {
				router.Get("/providers", api.listProviders)
				router.Get("/session", api.getAISession)
				router.Put("/session", api.putAISession)
				router.Delete("/session", api.deleteAISession)
				router.Post("/review", api.reviewResume)
				router.Post("/rewrite", api.rewriteText)
			})
		})
	})

	router.NotFound(func(writer http.ResponseWriter, request *http.Request) {
		writeError(writer, request, http.StatusNotFound, "not_found", "Route not found", nil)
	})
	router.MethodNotAllowed(func(writer http.ResponseWriter, request *http.Request) {
		writeError(writer, request, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
	})
	return router
}

func (api *API) health(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), time.Second)
	defer cancel()
	if api.db == nil || api.db.Ping(ctx) != nil {
		writeJSON(writer, http.StatusServiceUnavailable, map[string]any{
			"status": "degraded", "database": "unavailable", "time": time.Now().UTC(),
		})
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"status": "ok", "database": "ok", "time": time.Now().UTC(),
	})
}
