package httpapi

import (
	"net/http"

	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/quality"
	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/resume"
)

type qualityEvaluationInput struct {
	Resume resume.Document `json:"resume"`
}

// evaluateResumeQuality is deliberately independent of an AI session. Wire it
// as POST /api/v1/quality/evaluate from router.go.
func (api *API) evaluateResumeQuality(writer http.ResponseWriter, request *http.Request) {
	var input qualityEvaluationInput
	if err := decodeJSON(writer, request, &input); err != nil {
		writeRequestError(writer, request, err)
		return
	}
	evaluation, err := quality.Evaluate(input.Resume)
	if err != nil {
		api.internalError(writer, request, "evaluate resume quality", err)
		return
	}
	writeData(writer, http.StatusOK, evaluation)
}
