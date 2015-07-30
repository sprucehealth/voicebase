package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
)

type mockedDataAPI_referralProgramTemplateHandler struct {
	api.DataAPI
	referralProgramTemplatesParam      common.ReferralProgramStatusList
	referralProgramTemplates           []*common.ReferralProgramTemplate
	referralProgramTemplatesErr        error
	createReferralProgramTemplateParam *common.ReferralProgramTemplate
	createReferralProgramTemplateErr   error
	promotion                          *common.Promotion
	promotionErr                       error
	promotionParam                     int64
}

func (m *mockedDataAPI_referralProgramTemplateHandler) ReferralProgramTemplates(statuses common.ReferralProgramStatusList, types map[string]reflect.Type) ([]*common.ReferralProgramTemplate, error) {
	m.referralProgramTemplatesParam = statuses
	return m.referralProgramTemplates, m.referralProgramTemplatesErr
}

func (m *mockedDataAPI_referralProgramTemplateHandler) Promotion(codeID int64, types map[string]reflect.Type) (*common.Promotion, error) {
	m.promotionParam = codeID
	return m.promotion, m.promotionErr
}

func (m *mockedDataAPI_referralProgramTemplateHandler) CreateReferralProgramTemplate(template *common.ReferralProgramTemplate) (int64, error) {
	m.createReferralProgramTemplateParam = template
	return 1, m.createReferralProgramTemplateErr
}

func TestReferralProgramTemplateHandlerGETQueriesDataLayer(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?", nil)
	test.OK(t, err)
	template := &common.ReferralProgramTemplate{Created: time.Now()}
	mh := &mockedDataAPI_referralProgramTemplateHandler{
		referralProgramTemplates: []*common.ReferralProgramTemplate{template},
	}
	handler := newReferralProgramTemplateHandler(mh)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, &ReferralProgramTemplateGETResponse{ReferralProgramTemplates: []*responses.ReferralProgramTemplate{responses.TransformReferralProgramTemplate(template)}})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	var exp common.ReferralProgramStatusList
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
	test.Equals(t, exp, mh.referralProgramTemplatesParam)
}

func TestReferralProgramTemplateHandlerGETQueriesDataLayerParams(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?statuses=Active", nil)
	test.OK(t, err)
	template := &common.ReferralProgramTemplate{Created: time.Now()}
	mh := &mockedDataAPI_referralProgramTemplateHandler{
		referralProgramTemplates: []*common.ReferralProgramTemplate{template},
	}
	handler := newReferralProgramTemplateHandler(mh)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, &ReferralProgramTemplateGETResponse{ReferralProgramTemplates: []*responses.ReferralProgramTemplate{responses.TransformReferralProgramTemplate(template)}})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	var exp common.ReferralProgramStatusList = []string{"Active"}
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
	test.Equals(t, exp, mh.referralProgramTemplatesParam)
}

func TestReferralProgramTemplateHandlerPOSTQueriesDataLayer(t *testing.T) {
	req, err := json.Marshal(&ReferralProgramTemplatePOSTRequest{
		PromotionCodeID: 1,
		Title:           "title",
		Description:     "desc",
		ShareText:       &promotions.ShareTextParams{},
		Group:           "group",
		HomeCard:        &promotions.HomeCardConfig{},
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request?", bytes.NewReader(req))
	test.OK(t, err)
	mh := &mockedDataAPI_referralProgramTemplateHandler{
		promotion: &common.Promotion{
			Data:    promotions.NewMoneyOffVisitPromotion(1000, "group", "displayMsg", "shortMsg", "successMsg", "", 0, 0, true),
			Created: time.Now(),
		},
	}
	handler := newReferralProgramTemplateHandler(mh)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, &ReferralProgramTemplatePOSTResponse{ID: 1})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
	test.Equals(t, int64(1), mh.promotionParam)
}

func TestReferralProgramTemplateHandlerPOSTParamsRequired(t *testing.T) {
	req, err := json.Marshal(&ReferralProgramTemplatePOSTRequest{
		PromotionCodeID: 1,
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request?", bytes.NewReader(req))
	test.OK(t, err)
	mh := &mockedDataAPI_referralProgramTemplateHandler{
		promotion: &common.Promotion{Data: &TestTyped{Name: "TestType"}, Created: time.Now()},
	}
	handler := newReferralProgramTemplateHandler(mh)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusBadRequest, &ReferralProgramTemplatePOSTResponse{ID: 1})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
}
