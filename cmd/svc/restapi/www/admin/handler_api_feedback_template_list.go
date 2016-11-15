package admin

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/audit"
	"github.com/sprucehealth/backend/cmd/svc/restapi/feedback"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
)

type feedbackTemplateListHandler struct {
	feedbackClient feedback.DAL
}

type feedbackTemplateListResponse struct {
	Templates []*feedback.FeedbackTemplateData `json:"templates"`
}

func newFeedbackTemplateListHandler(feedbackClient feedback.DAL) http.Handler {
	return httputil.SupportedMethods(&feedbackTemplateListHandler{
		feedbackClient: feedbackClient,
	}, httputil.Get)
}

func (f *feedbackTemplateListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(r.Context())
	audit.LogAction(account.ID, "AdminAPI", "ListActiveFeedbackTemplates", nil)

	templates, err := f.feedbackClient.ListActiveTemplates()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &feedbackTemplateListResponse{
		Templates: templates,
	})
}
