package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/feedback"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
)

type feedbackTemplateTypesHandler struct{}

type typeItem struct {
	Type string      `json:"template_type"`
	Data interface{} `json:"data"`
}

type feedbackTemplateTypesResponse struct {
	Types []typeItem `json:"types"`
}

func newFeedbackTemplateTypesHandler() http.Handler {
	return httputil.SupportedMethods(
		&feedbackTemplateTypesHandler{}, httputil.Get)
}

func (h *feedbackTemplateTypesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(r.Context())
	audit.LogAction(account.ID, "AdminAPI", "ListFeedbackTemplateTypes", nil)

	httputil.JSONResponse(w, http.StatusOK, feedbackTemplateTypesResponse{
		Types: []typeItem{
			{
				Type: feedback.FTFreetext,
				Data: feedback.FreeTextTemplate{},
			},
			{
				Type: feedback.FTMultipleChoice,
				Data: feedback.MultipleChoiceTemplate{
					PotentialAnswers: []*feedback.PotentialAnswer{
						{},
					},
				},
			},
			{
				Type: feedback.FTOpenURL,
				Data: feedback.OpenURLTemplate{},
			},
		},
	})
}
