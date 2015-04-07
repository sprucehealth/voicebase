package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_handler"
)

type mockedDataAPI_handlerLayoutVersion struct {
	api.DataAPI
	items []*api.LayoutVersionInfo
}

func (d mockedDataAPI_handlerLayoutVersion) LayoutVersions() ([]*api.LayoutVersionInfo, error) {
	return d.items, nil
}

func TestLayoutVersionHandlerSuccessGET(t *testing.T) {
	items := []*api.LayoutVersionInfo{
		{
			PathwayTag: "foo",
			SKUType:    "bar",
		},
		{
			PathwayTag: "foo1",
			SKUType:    "bar1",
		},
	}

	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	layoutVersionHandler := NewLayoutVersionHandler(mockedDataAPI_handlerLayoutVersion{&api.DataService{}, items})
	handler := test_handler.MockHandler{
		H: layoutVersionHandler,
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, map[string]interface{}{
		"items": items,
	})
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}
