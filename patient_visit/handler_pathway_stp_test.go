package patient_visit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/test"
)

type mockDataAPI_PathwaySTPHandler struct {
	api.DataAPI
	data []byte
}

func (m *mockDataAPI_PathwaySTPHandler) PathwaySTP(pathwayTag string) ([]byte, error) {
	return m.data, nil
}

func TestPathwaySTPHandler(t *testing.T) {
	expectedData := map[string]interface{}{
		"message": "hi",
	}
	jsonData, err := json.Marshal(expectedData)
	test.OK(t, err)

	m := &mockDataAPI_PathwaySTPHandler{
		data: jsonData,
	}

	h := NewPathwaySTPHandler(m)
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "api.spruce.local/pathwaystp", nil)
	test.OK(t, err)
	h.ServeHTTP(context.Background(), w, r)
	test.Equals(t, http.StatusOK, w.Code)

	var sampleData struct {
		SampleTreatmentPlan interface{} `json:"sample_treatment_plan"`
	}
	test.OK(t, json.Unmarshal(w.Body.Bytes(), &sampleData))
	test.Equals(t, expectedData, sampleData.SampleTreatmentPlan)
}
