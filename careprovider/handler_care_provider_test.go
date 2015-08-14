package careprovider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

type mockedDataAPI_careProvider struct {
	api.DataAPI
	doctor      *common.Doctor
	doctorError error
}

func (m mockedDataAPI_careProvider) Doctor(doctorID int64, long bool) (*common.Doctor, error) {
	if m.doctorError != nil {
		return nil, m.doctorError
	}
	return m.doctor, nil
}

func TestHandlerCareProviderGETRequiresProviderID(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	handler := NewCareProviderHandler(mockedDataAPI_careProvider{DataAPI: &mockedDataAPI_careProvider{}, doctor: nil, doctorError: nil}, "api.spruce.local")
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteError(context.Background(), apiservice.NewValidationError("RequestID: 0, Error: Unable to parse input parameters: The following parameters are missing: provider_id, StatusCode: 400"), expectedWriter, r)
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestHandlerCareProviderGETSuccess(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?provider_id=9", nil)
	test.OK(t, err)
	doctor := buildDummyDoctor("foo")
	handler := NewCareProviderHandler(mockedDataAPI_careProvider{DataAPI: &mockedDataAPI_careProvider{}, doctor: doctor, doctorError: nil}, "api.spruce.local")
	response := responses.NewCareProviderFromDoctorDBModel(doctor, "api.spruce.local")
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, response)
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestHandlerCareProviderGETNoRecord(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?provider_id=9", nil)
	test.OK(t, err)
	handler := NewCareProviderHandler(mockedDataAPI_careProvider{DataAPI: &mockedDataAPI_careProvider{}, doctor: nil, doctorError: api.ErrNotFound("Foo")}, "api.spruce.local")
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteResourceNotFoundError(context.Background(), fmt.Sprintf("No care provider exists for ID %d", 9), expectedWriter, r)
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func buildDummyDoctor(name string) *common.Doctor {
	return &common.Doctor{
		ID:        encoding.DeprecatedNewObjectID(1),
		FirstName: name,
		LastName:  name,
	}
}
