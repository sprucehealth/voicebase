package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_handler"
	"github.com/sprucehealth/backend/www"
)

type mockedDataAPI_handlerLayoutVersion struct {
	api.DataAPI
	mapping api.PathwayPurposeVersionMapping
}

func (d mockedDataAPI_handlerLayoutVersion) LayoutVersionMapping() (api.PathwayPurposeVersionMapping, error) {
	return d.mapping, nil
}

func TestLayoutVersionHandlerDoctorCannotGET(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	layoutVersionHandler := NewLayoutVersionHandler(mockedDataAPI_handlerLayoutVersion{&api.DataService{}, nil})
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

func TestLayoutVersionHandlerPatientCannotGET(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	layoutVersionHandler := NewLayoutVersionHandler(mockedDataAPI_handlerLayoutVersion{&api.DataService{}, nil})
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

func TestLayoutVersionHandlerSuccessGET(t *testing.T) {
	mapping := make(map[string]map[string][]*common.Version)
	mapping["foo"] = make(map[string][]*common.Version)
	mapping["foo"]["bar"] = append(mapping["foo"]["bar"], &common.Version{})
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	layoutVersionHandler := NewLayoutVersionHandler(mockedDataAPI_handlerLayoutVersion{&api.DataService{}, mapping})
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
