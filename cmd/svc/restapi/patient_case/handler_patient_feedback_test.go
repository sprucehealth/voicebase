package patient_case

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/feedback"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
)

type feedbackMockClient struct {
	feedback.DAL
	f *feedback.PatientFeedback
}

func (d *feedbackMockClient) PatientFeedback(feedbackFor string) (*feedback.PatientFeedback, error) {
	if feedbackFor != "case:1" {
		return nil, nil
	}
	return d.f, nil
}

func (d *feedbackMockClient) AdditionalFeedback(feedbackID int64) (*feedback.FeedbackTemplateData, []byte, error) {
	jsonData, err := json.Marshal(&feedback.FreeTextResponse{
		Response: "hello",
	})
	if err != nil {
		return nil, nil, err
	}

	ft := &feedback.FeedbackTemplateData{
		Template: &feedback.FreeTextTemplate{
			Title: "title",
		},
	}

	return ft, jsonData, nil
}

func TestPatientFeedbackHandler(t *testing.T) {
	fClient := &feedbackMockClient{
		f: &feedback.PatientFeedback{
			Rating:  ptr.Int(4),
			Comment: ptr.String("RULEZ!"),
			Created: time.Unix(12341234, 0),
		},
	}
	h := NewPatientFeedbackHandler(fClient)

	r, err := http.NewRequest("GET", "/?case_id=1", nil)
	test.OK(t, err)
	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 1, Role: api.RolePatient})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r.WithContext(ctx))
	test.Equals(t, http.StatusForbidden, w.Code)

	ctx = apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 2, Role: api.RoleCC})
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r.WithContext(ctx))
	test.Equals(t, http.StatusOK, w.Code)
	test.Equals(t, "{\"feedback\":[{\"rating\":4,\"comment\":\"RULEZ!\",\"created_timestamp\":12341234}]}\n", w.Body.String())
}
