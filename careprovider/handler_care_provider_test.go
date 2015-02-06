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
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_handler"
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
	careProviderHandler := NewCareProviderHandler(mockedDataAPI_careProvider{DataAPI: &api.DataService{}, doctor: nil, doctorError: nil}, "api.spruce.local")
	handler := test_handler.MockHandler{
		H: careProviderHandler,
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteError(apiservice.NewValidationError("RequestID: 0, Error: Unable to parse input parameters: The following parameters are missing: provider_id, StatusCode: 400"), expectedWriter, r)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestHandlerCareProviderGETSuccess(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?provider_id=9", nil)
	test.OK(t, err)
	doctor := buildDummyDoctor("foo")
	careProviderHandler := NewCareProviderHandler(mockedDataAPI_careProvider{DataAPI: &api.DataService{}, doctor: doctor, doctorError: nil}, "api.spruce.local")
	handler := test_handler.MockHandler{
		H: careProviderHandler,
	}
	response := responses.NewCareProviderFromDoctorDBModel(doctor, "api.spruce.local")
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteJSON(expectedWriter, response)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestHandlerCareProviderGETNoRecord(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?provider_id=9", nil)
	test.OK(t, err)
	careProviderHandler := NewCareProviderHandler(mockedDataAPI_careProvider{DataAPI: &api.DataService{}, doctor: nil, doctorError: api.ErrNotFound("Foo")}, "api.spruce.local")
	handler := test_handler.MockHandler{
		H: careProviderHandler,
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteResourceNotFoundError(fmt.Sprintf("No care provider exists for ID %d", 9), expectedWriter, r)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func buildDummyDoctor(name string) *common.Doctor {
	return &common.Doctor{
		DoctorID:  encoding.NewObjectID(1),
		FirstName: name,
		LastName:  name,
	}
}
