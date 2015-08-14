package patient_case

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

type feedbackDataAPI struct {
	api.DataAPI
	feedback []*common.PatientFeedback
}

func (d *feedbackDataAPI) PatientFeedback(feedbackFor string) ([]*common.PatientFeedback, error) {
	if feedbackFor != "case:1" {
		return nil, nil
	}
	return d.feedback, nil
}

func TestPatientFeedbackHandler(t *testing.T) {
	dataAPI := &feedbackDataAPI{
		feedback: []*common.PatientFeedback{
			{Rating: 4, Comment: "RULEZ!", Created: time.Unix(12341234, 0)},
		},
	}
	h := NewPatientFeedbackHandler(dataAPI)

	r, err := http.NewRequest("GET", "/?case_id=1", nil)
	test.OK(t, err)
	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 1, Role: api.RolePatient})
	w := httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusForbidden, w.Code)

	ctx = apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 2, Role: api.RoleCC})
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)
	test.Equals(t, "{\"feedback\":[{\"rating\":4,\"comment\":\"RULEZ!\",\"created_timestamp\":12341234}]}\n", w.Body.String())
}
