package treatment_plan

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_handler"
)

type mockedDataAPI_handlerRXGuide struct {
	api.DataAPI
}

func (m mockedDataAPI_handlerRXGuide) QueryDrugDetails(query *api.DrugDetailsQuery) (*common.DrugDetails, error) {
	return &common.DrugDetails{
		ID:                0,
		Name:              "Name",
		NDC:               "NDC",
		GenericName:       "Generic Name",
		Route:             "Route",
		Form:              "Form",
		ImageURL:          "ImageURL",
		OtherNames:        "Other Names",
		Description:       "Desctiption",
		Tips:              []string{"Tip1", "Tip2"},
		Warnings:          []string{"Warn1"},
		CommonSideEffects: []string{"Side effect 1"},
	}, nil
}

func TestHandlerRXGuideRequiresParams(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	rxGuideHandler := NewRXGuideHandler(mockedDataAPI_handlerRXGuide{DataAPI: &api.DataService{}})
	handler := test_handler.MockHandler{
		H: rxGuideHandler,
	}
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestHandlerRXGuideSuccess(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?generic_name=generic_name&route=route&dosage=dosage", nil)
	test.OK(t, err)
	dataAPI := mockedDataAPI_handlerRXGuide{DataAPI: &api.DataService{}}
	rxGuideHandler := NewRXGuideHandler(dataAPI)
	handler := test_handler.MockHandler{
		H: rxGuideHandler,
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	treatmentGuideResponse(dataAPI, "generic_name", "route", "", "dosage", "", nil, nil, expectedWriter, r)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}
