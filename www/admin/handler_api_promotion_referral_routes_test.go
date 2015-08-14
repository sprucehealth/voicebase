package admin

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type mockedDataAPI_promotionReferralRoutesHandler struct {
	api.DataAPI
	promotionReferralRoutes            []*common.PromotionReferralRoute
	promotionReferralRoutesErr         error
	insertPromotionReferralRoutesParam *common.PromotionReferralRoute
	insertPromotionReferralRoutesErr   error
}

func (m *mockedDataAPI_promotionReferralRoutesHandler) PromotionReferralRoutes(lifecycles []string) ([]*common.PromotionReferralRoute, error) {
	return m.promotionReferralRoutes, m.promotionReferralRoutesErr
}

func (m *mockedDataAPI_promotionReferralRoutesHandler) InsertPromotionReferralRoute(route *common.PromotionReferralRoute) (int64, error) {
	m.insertPromotionReferralRoutesParam = route
	return 1, m.insertPromotionReferralRoutesErr
}

func TestPromotionReferralRoutesHandlerGETQueriesDataLayer(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?lifecycles=ACTIVE", nil)
	test.OK(t, err)
	route := &common.PromotionReferralRoute{
		ID:              1,
		PromotionCodeID: 1,
		Created:         time.Now(),
		Priority:        1,
		Lifecycle:       "ACTIVE",
	}
	handler := newPromotionReferralRoutesHandler(&mockedDataAPI_promotionReferralRoutesHandler{
		promotionReferralRoutes: []*common.PromotionReferralRoute{route},
	})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, &promotionReferralRoutesGETResponse{PromotionReferralRoutes: []*responses.PromotionReferralRoute{responses.TransformPromotionReferralRoute(route)}})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
}

func TestPromotionReferralRoutesHandlerGETRequiredLifecycle(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	handler := newPromotionReferralRoutesHandler(&mockedDataAPI_promotionReferralRoutesHandler{})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusBadRequest, struct{}{})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
}

func TestPromotionReferralRoutesHandlerGETNoRecords(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?lifecycles=ACTIVE", nil)
	test.OK(t, err)
	handler := newPromotionReferralRoutesHandler(&mockedDataAPI_promotionReferralRoutesHandler{
		promotionReferralRoutesErr: api.ErrNotFound(`promotion_referral_route`),
	})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, &promotionReferralRoutesGETResponse{PromotionReferralRoutes: []*responses.PromotionReferralRoute{}})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
}

func TestPromotionReferralRoutesHandlerGETQueryErr(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?lifecycles=ACTIVE", nil)
	test.OK(t, err)
	handler := newPromotionReferralRoutesHandler(&mockedDataAPI_promotionReferralRoutesHandler{
		promotionReferralRoutesErr: errors.New("Foo"),
	})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusInternalServerError, struct{}{})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
}

func TestPromotionReferralRoutesHandlerPOSTQueriesDataLayer(t *testing.T) {
	req, err := json.Marshal(&promotionReferralRoutesPOSTRequest{
		PromotionCodeID: 1,
		Priority:        1,
		Lifecycle:       "ACTIVE",
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(req))
	test.OK(t, err)
	mh := &mockedDataAPI_promotionReferralRoutesHandler{}
	handler := newPromotionReferralRoutesHandler(mh)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, struct{}{})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
	test.Equals(t, int64(1), mh.insertPromotionReferralRoutesParam.PromotionCodeID)
	test.Equals(t, 1, mh.insertPromotionReferralRoutesParam.Priority)
	test.Equals(t, common.PRRLifecycle("ACTIVE"), mh.insertPromotionReferralRoutesParam.Lifecycle)
}

func TestPromotionReferralRoutesHandlerPOSTRequiredParams(t *testing.T) {
	req, err := json.Marshal(&promotionReferralRoutesPOSTRequest{})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(req))
	test.OK(t, err)
	handler := newPromotionReferralRoutesHandler(&mockedDataAPI_promotionReferralRoutesHandler{})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	www.APIBadRequestError(expectedWriter, r, errors.New("promotion_code_id, priority, lifecycle required").Error())
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
}

func TestPromotionReferralRoutesHandlerPOSTDataLayerErr(t *testing.T) {
	req, err := json.Marshal(&promotionReferralRoutesPOSTRequest{
		PromotionCodeID: 1,
		Priority:        1,
		Lifecycle:       "ACTIVE",
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(req))
	test.OK(t, err)
	mh := &mockedDataAPI_promotionReferralRoutesHandler{
		insertPromotionReferralRoutesErr: errors.New("Foo"),
	}
	handler := newPromotionReferralRoutesHandler(mh)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	www.APIInternalError(expectedWriter, r, errors.New("Foo"))
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
}
