package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/feedback"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/www"

	"golang.org/x/net/context"
)

type feedbackTemplateHandler struct {
	feedbackClient feedback.DAL
}

type feedbackTemplateGetResponse struct {
	TemplateData *feedback.FeedbackTemplateData `json:"template_data"`
}

type feedbackTemplatePutRequest struct {
	Tag          string `json:"tag"`
	Type         string `json:"type"`
	TemplateJSON string `json:"template_json"`
}

func newFeedbackTemplateHandler(feedbackClient feedback.DAL) httputil.ContextHandler {
	return httputil.SupportedMethods(&feedbackTemplateHandler{
		feedbackClient: feedbackClient,
	}, httputil.Get, httputil.Put)
}

func (f *feedbackTemplateHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		f.get(ctx, w, r)
	case httputil.Put:
		f.put(ctx, w, r)
	}
}

func (f *feedbackTemplateHandler) get(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(ctx)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "GetFeedbackTemplate", map[string]interface{}{
		"id": id,
	})

	fd, err := f.feedbackClient.FeedbackTemplate(id)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &feedbackTemplateGetResponse{
		TemplateData: fd,
	})
}

func (f *feedbackTemplateHandler) put(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "PutFeedbackTemplate", nil)

	var rd feedbackTemplatePutRequest
	if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
		www.APIBadRequestError(w, r, err.Error())
		return
	} else if rd.Tag == "" {
		www.APIBadRequestError(w, r, "tag is required")
		return
	} else if rd.Type == "" {
		www.APIBadRequestError(w, r, "type is required")
		return
	} else if rd.TemplateJSON == "" {
		www.APIBadRequestError(w, r, "template json is required")
		return
	}

	ft, err := feedback.TemplateFromJSON(rd.Type, []byte(rd.TemplateJSON))
	if err != nil {
		www.APIBadRequestError(w, r, err.Error())
		return
	}

	if err := ft.Validate(); err != nil {
		www.APIBadRequestError(w, r, err.Error())
		return
	}

	_, err = f.feedbackClient.CreateFeedbackTemplate(feedback.FeedbackTemplateData{
		Type:     rd.Type,
		Tag:      rd.Tag,
		Template: ft,
	})
	if err != nil {
		www.APIBadRequestError(w, r, err.Error())
		return
	}

	httputil.JSONResponse(w, http.StatusOK, map[string]interface{}{})
}
