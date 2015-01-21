package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_handler"
	"github.com/sprucehealth/backend/www"
)

type mockedDataAPI struct {
	api.DataAPI
	mapping map[string]map[string][]string
}

func (d mockedDataAPI) LayoutVersionMapping() (map[string]map[string][]string, error) {
	return d.mapping, nil
}

func TestLayoutVersionHandlerDoctorCannotGET(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	layoutVersionHandler := NewLayoutVersionHandler(mockedDataAPI{&api.DataService{}, nil})
	handler := test_handler.MockHandler{
		H: layoutVersionHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.DOCTOR_ROLE
		},
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteAccessNotAllowedError(expectedWriter, r)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestLayoutVersionHandlerTestPatientCannotGET(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	layoutVersionHandler := NewLayoutVersionHandler(mockedDataAPI{&api.DataService{}, nil})
	handler := test_handler.MockHandler{
		H: layoutVersionHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.PATIENT_ROLE
		},
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteAccessNotAllowedError(expectedWriter, r)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestLayoutVersionHandlerTestSuccessGET(t *testing.T) {
	mapping := make(map[string]map[string][]string)
	mapping["foo"] = make(map[string][]string)
	mapping["foo"]["bar"] = append(mapping["foo"]["bar"], "baz")
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	layoutVersionHandler := NewLayoutVersionHandler(mockedDataAPI{&api.DataService{}, mapping})
	handler := test_handler.MockHandler{
		H: layoutVersionHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.ADMIN_ROLE
		},
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	www.JSONResponse(expectedWriter, r, http.StatusOK, mapping)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}
