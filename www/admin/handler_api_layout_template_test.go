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

type mockedDataAPI_handlerLayoutTemplate struct {
	api.DataAPI
	template []byte
}

func (d mockedDataAPI_handlerLayoutTemplate) LayoutTemplate(pathway, purpose string, version *common.Version) ([]byte, error) {
	return d.template, nil
}

func TestLayoutTemplateHandlerSuccessGET(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?pathway_tag=1&purpose=INTAKE&major=1&minor=0&patch=0", nil)
	test.OK(t, err)
	template := []byte(`{"Template":"output"}`)
	resp := make(map[string]string)
	resp["Template"] = "output"
	layoutTemplateHandler := NewLayoutTemplateHandler(mockedDataAPI_handlerLayoutTemplate{&api.DataService{}, template})
	handler := test_handler.MockHandler{
		H: layoutTemplateHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.ADMIN_ROLE
		},
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	www.JSONResponse(expectedWriter, r, http.StatusOK, resp)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestLayoutTemplateHandlerFailGETRequiresTag(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?purpose=INTAKE&major=1&minor=0&patch=0", nil)
	test.OK(t, err)
	template := []byte(`{"Template":"output"}`)
	resp := make(map[string]string)
	resp["Template"] = "output"
	layoutTemplateHandler := NewLayoutTemplateHandler(mockedDataAPI_handlerLayoutTemplate{&api.DataService{}, template})
	handler := test_handler.MockHandler{
		H: layoutTemplateHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.ADMIN_ROLE
		},
	}
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestLayoutTemplateHandlerFailGETRequiresPurpose(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?pathway_tag=1&major=1&minor=0&patch=0", nil)
	test.OK(t, err)
	template := []byte(`{"Template":"output"}`)
	resp := make(map[string]string)
	resp["Template"] = "output"
	layoutTemplateHandler := NewLayoutTemplateHandler(mockedDataAPI_handlerLayoutTemplate{&api.DataService{}, template})
	handler := test_handler.MockHandler{
		H: layoutTemplateHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.ADMIN_ROLE
		},
	}
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestLayoutTemplateHandlerFailGETRequiresMajor(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?pathway_tag=1&purpose=INTAKE&minor=0&patch=0", nil)
	test.OK(t, err)
	template := []byte(`{"Template":"output"}`)
	resp := make(map[string]string)
	resp["Template"] = "output"
	layoutTemplateHandler := NewLayoutTemplateHandler(mockedDataAPI_handlerLayoutTemplate{&api.DataService{}, template})
	handler := test_handler.MockHandler{
		H: layoutTemplateHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.ADMIN_ROLE
		},
	}
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestLayoutTemplateHandlerFailGETRequiresMinor(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?pathway_tag=1&purpose=INTAKE&major=1&patch=0", nil)
	test.OK(t, err)
	template := []byte(`{"Template":"output"}`)
	resp := make(map[string]string)
	resp["Template"] = "output"
	layoutTemplateHandler := NewLayoutTemplateHandler(mockedDataAPI_handlerLayoutTemplate{&api.DataService{}, template})
	handler := test_handler.MockHandler{
		H: layoutTemplateHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.ADMIN_ROLE
		},
	}
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestLayoutTemplateHandlerFailGETRequiresPatch(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?pathway_tag=1&purpose=INTAKE&major=1&minor=0", nil)
	test.OK(t, err)
	template := []byte(`{"Template":"output"}`)
	resp := make(map[string]string)
	resp["Template"] = "output"
	layoutTemplateHandler := NewLayoutTemplateHandler(mockedDataAPI_handlerLayoutTemplate{&api.DataService{}, template})
	handler := test_handler.MockHandler{
		H: layoutTemplateHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.ADMIN_ROLE
		},
	}
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}
