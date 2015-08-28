package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

type mockedDataAPI_handlerLayoutTemplate struct {
	api.DataAPI
	template []byte
}

func (d mockedDataAPI_handlerLayoutTemplate) LayoutTemplate(pathway, sku, purpose string, version *encoding.Version) ([]byte, error) {
	return d.template, nil
}

func TestLayoutTemplateHandlerSuccessGET(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?pathway_tag=1&purpose=INTAKE&major=1&minor=0&patch=0&sku=test", nil)
	test.OK(t, err)
	template := []byte(`{"Template":"output"}`)
	resp := make(map[string]string)
	resp["Template"] = "output"
	handler := newLayoutTemplateHandler(mockedDataAPI_handlerLayoutTemplate{template: template})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, resp)
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}
