package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/www"
)

type mockedDataAPI_stpHandler struct {
	api.DataAPI
	stp        []byte
	pathwayTag string
	pathwayErr error
	t          *testing.T
}

func (m mockedDataAPI_stpHandler) PathwaySTP(pathwayTag string) ([]byte, error) {
	if m.pathwayErr == nil {
		return m.stp, nil
	}
	return nil, m.pathwayErr
}

func (m mockedDataAPI_stpHandler) CreatePathwaySTP(pathwayTag string, content []byte) error {
	test.Equals(m.t, m.pathwayTag, pathwayTag)
	test.Equals(m.t, m.stp, content)
	return nil
}

func TestSTPHandlerGETRequiresParams(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?", nil)
	test.OK(t, err)
	handler := newSampleTreatmentPlanHandler(mockedDataAPI_stpHandler{DataAPI: &api.DataService{}})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	www.APIBadRequestError(expectedWriter, r, fmt.Errorf("Unable to parse input parameters: The following parameters are missing: pathway_tag").Error())
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestSTPHandlerGETSuccess(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?pathway_tag=foo", nil)
	test.OK(t, err)
	stp := []byte(`{"yo":"datums"}`)
	handler := newSampleTreatmentPlanHandler(mockedDataAPI_stpHandler{DataAPI: &api.DataService{}, stp: stp})

	var response interface{}
	err = json.Unmarshal(stp, &response)
	test.OK(t, err)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, response)
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestSTPHandlerGETSuccessNoRecord(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?pathway_tag=foo", nil)
	test.OK(t, err)
	handler := newSampleTreatmentPlanHandler(mockedDataAPI_stpHandler{DataAPI: &api.DataService{}, pathwayErr: api.ErrNotFound("Not found")})

	var response interface{}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, response)
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestSTPHandlerPUTRequiresPathwayTagParam(t *testing.T) {
	r, err := http.NewRequest("PUT", "mock.api.request?", strings.NewReader("{}"))
	test.OK(t, err)
	handler := newSampleTreatmentPlanHandler(mockedDataAPI_stpHandler{DataAPI: &api.DataService{}})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	www.APIBadRequestError(expectedWriter, r, fmt.Errorf("Incomplete request body - pathway_tag required").Error())
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestSTPHandlerPUTRequiresSampleTreatmentPlanParam(t *testing.T) {
	r, err := http.NewRequest("PUT", "mock.api.request?", strings.NewReader(`{"pathway_tag":"foo"}`))
	test.OK(t, err)
	handler := newSampleTreatmentPlanHandler(mockedDataAPI_stpHandler{DataAPI: &api.DataService{}})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	www.APIBadRequestError(expectedWriter, r, fmt.Errorf("Incomplete request body - sample_treatment_plan required").Error())
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestSTPHandlerPUTSuccess(t *testing.T) {
	r, err := http.NewRequest("PUT", "mock.api.request?", strings.NewReader(`{"pathway_tag":"foo","sample_treatment_plan":{"yo":"datums"}}`))
	test.OK(t, err)
	handler := newSampleTreatmentPlanHandler(mockedDataAPI_stpHandler{DataAPI: &api.DataService{}, pathwayTag: "foo", stp: []byte(`{"yo":"datums"}`), t: t})
	var response interface{}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, response)
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}
