package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/feedback"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"

	"golang.org/x/net/context"
)

type feedbackTemplateTypesHandler struct{}

type typeItem struct {
	Type string      `json:"template_type"`
	Data interface{} `json:"data"`
}

type feedbackTemplateTypesResponse struct {
	Types []typeItem `json:"types"`
}

func newFeedbackTemplateTypesHandler() httputil.ContextHandler {
	return httputil.SupportedMethods(
		&feedbackTemplateTypesHandler{}, httputil.Get)
}

func (h *feedbackTemplateTypesHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)
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
					PotentialAnswers: []feedback.PotentialAnswer{
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
