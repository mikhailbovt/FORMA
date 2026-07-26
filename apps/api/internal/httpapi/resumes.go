package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/resume"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

func (api *API) listResumes(writer http.ResponseWriter, request *http.Request) {
	limit, err := integerQuery(request, "limit", 50, 1, 100)
	if err != nil {
		writeRequestError(writer, request, err)
		return
	}
	offset, err := integerQuery(request, "offset", 0, 0, 100_000)
	if err != nil {
		writeRequestError(writer, request, err)
		return
	}
	items, err := api.resumes.List(request.Context(), limit, offset)
	if err != nil {
		api.internalError(writer, request, "list resumes", err)
		return
	}
	writeData(writer, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}

func (api *API) createResume(writer http.ResponseWriter, request *http.Request) {
	var input resume.Input
	if err := decodeJSON(writer, request, &input); err != nil {
		writeRequestError(writer, request, err)
		return
	}
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		writeError(writer, request, http.StatusUnprocessableEntity, "validation_error", "Resume is invalid", fields)
		return
	}
	item, err := api.resumes.Create(request.Context(), input)
	if err != nil {
		api.internalError(writer, request, "create resume", err)
		return
	}
	writer.Header().Set("Location", "/api/v1/resumes/"+item.ID.String())
	writeData(writer, http.StatusCreated, item)
}

func (api *API) getResume(writer http.ResponseWriter, request *http.Request) {
	id, ok := api.resumeID(writer, request)
	if !ok {
		return
	}
	item, err := api.resumes.Get(request.Context(), id)
	if err != nil {
		api.resumeStoreError(writer, request, "get resume", err)
		return
	}
	writeData(writer, http.StatusOK, item)
}

func (api *API) updateResume(writer http.ResponseWriter, request *http.Request) {
	id, ok := api.resumeID(writer, request)
	if !ok {
		return
	}
	var input resume.Input
	if err := decodeJSON(writer, request, &input); err != nil {
		writeRequestError(writer, request, err)
		return
	}
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		writeError(writer, request, http.StatusUnprocessableEntity, "validation_error", "Resume is invalid", fields)
		return
	}
	item, err := api.resumes.Update(request.Context(), id, input)
	if err != nil {
		api.resumeStoreError(writer, request, "update resume", err)
		return
	}
	writeData(writer, http.StatusOK, item)
}

func (api *API) deleteResume(writer http.ResponseWriter, request *http.Request) {
	id, ok := api.resumeID(writer, request)
	if !ok {
		return
	}
	if err := api.resumes.Delete(request.Context(), id); err != nil {
		api.resumeStoreError(writer, request, "delete resume", err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (api *API) duplicateResume(writer http.ResponseWriter, request *http.Request) {
	id, ok := api.resumeID(writer, request)
	if !ok {
		return
	}
	item, err := api.resumes.Duplicate(request.Context(), id)
	if err != nil {
		api.resumeStoreError(writer, request, "duplicate resume", err)
		return
	}
	writer.Header().Set("Location", "/api/v1/resumes/"+item.ID.String())
	writeData(writer, http.StatusCreated, item)
}

func (api *API) resumeID(writer http.ResponseWriter, request *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(request, "resumeID"))
	if err != nil {
		writeError(writer, request, http.StatusBadRequest, "invalid_id", "resumeID must be a UUID", nil)
		return uuid.Nil, false
	}
	return id, true
}

func (api *API) resumeStoreError(writer http.ResponseWriter, request *http.Request, operation string, err error) {
	if errors.Is(err, resume.ErrNotFound) {
		writeError(writer, request, http.StatusNotFound, "resume_not_found", "Resume not found", nil)
		return
	}
	api.internalError(writer, request, operation, err)
}

func (api *API) internalError(writer http.ResponseWriter, request *http.Request, operation string, err error) {
	api.logger.Error(operation, "request_id", middleware.GetReqID(request.Context()), "error", err)
	writeError(writer, request, http.StatusInternalServerError, "internal_error", "An internal error occurred", nil)
}

func integerQuery(request *http.Request, name string, fallback, minimum, maximum int) (int, error) {
	value := request.URL.Query().Get(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minimum || parsed > maximum {
		return 0, &requestError{Status: http.StatusBadRequest, Code: "invalid_query", Message: name + " is outside the allowed range"}
	}
	return parsed, nil
}
