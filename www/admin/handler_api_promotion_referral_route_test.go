package admin

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/test"
)

type mockedDataAPI_promotionReferralRouteHandler struct {
	api.DataAPI
	updatePromotionReferralRouteParam *common.PromotionReferralRouteUpdate
	updatePromotionReferralRouteErr   error
}

func (m *mockedDataAPI_promotionReferralRouteHandler) UpdatePromotionReferralRoute(routeUpdate *common.PromotionReferralRouteUpdate) (int64, error) {
	m.updatePromotionReferralRouteParam = routeUpdate
	return 1, m.updatePromotionReferralRouteErr
}

func TestPromotionReferralRouteHandlerPUTQueriesDataLayer(t *testing.T) {
	mh := &mockedDataAPI_promotionReferralRouteHandler{
		DataAPI: &api.DataService{},
	}
	promoReferralRouteHandler := NewPromotionReferralRouteHandler(mh)
	req, err := json.Marshal(&PromotionReferralRoutePUTRequest{
		Lifecycle: "DEPRECATED",
	})
	test.OK(t, err)
	r, err := http.NewRequest("PUT", "/admin/api/promotion/referral_route/1", bytes.NewReader(req))
	test.OK(t, err)
	m := mux.NewRouter()
	m.HandleFunc(`/admin/api/promotion/referral_route/{id:[0-9]+}`, promoReferralRouteHandler.ServeHTTP)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, struct{}{})
	m.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
	test.Equals(t, int64(1), mh.updatePromotionReferralRouteParam.ID)
	test.Equals(t, common.PRRLifecycle("DEPRECATED"), mh.updatePromotionReferralRouteParam.Lifecycle)
}

func TestPromotionReferralRouteHandlerPUTIDRequired(t *testing.T) {
	mh := &mockedDataAPI_promotionReferralRouteHandler{
		DataAPI: &api.DataService{},
	}
	promoReferralRouteHandler := NewPromotionReferralRouteHandler(mh)
	req, err := json.Marshal(&PromotionReferralRoutePUTRequest{
		Lifecycle: "DEPRECATED",
	})
	test.OK(t, err)
	r, err := http.NewRequest("PUT", "/admin/api/promotion/referral_route", bytes.NewReader(req))
	test.OK(t, err)
	m := mux.NewRouter()
	m.HandleFunc(`/admin/api/promotion/referral_route`, promoReferralRouteHandler.ServeHTTP)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusNotFound, struct{}{})
	m.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
}

func TestPromotionReferralRouteHandlerPUTLifecycleRequired(t *testing.T) {
	mh := &mockedDataAPI_promotionReferralRouteHandler{
		DataAPI: &api.DataService{},
	}
	promoReferralRouteHandler := NewPromotionReferralRouteHandler(mh)
	req, err := json.Marshal(&PromotionReferralRoutePUTRequest{})
	test.OK(t, err)
	r, err := http.NewRequest("PUT", "/admin/api/promotion/referral_route/1", bytes.NewReader(req))
	test.OK(t, err)
	m := mux.NewRouter()
	m.HandleFunc(`/admin/api/promotion/referral_route/{id:[0-9]+}`, promoReferralRouteHandler.ServeHTTP)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusBadRequest, struct{}{})
	m.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
}

func TestPromotionReferralRouteHandlerPUTDataLayerErr(t *testing.T) {
	mh := &mockedDataAPI_promotionReferralRouteHandler{
		DataAPI: &api.DataService{},
		updatePromotionReferralRouteErr: errors.New("Foo"),
	}
	promoReferralRouteHandler := NewPromotionReferralRouteHandler(mh)
	req, err := json.Marshal(&PromotionReferralRoutePUTRequest{
		Lifecycle: "DEPRECATED",
	})
	test.OK(t, err)
	r, err := http.NewRequest("PUT", "/admin/api/promotion/referral_route/1", bytes.NewReader(req))
	test.OK(t, err)
	m := mux.NewRouter()
	m.HandleFunc(`/admin/api/promotion/referral_route/{id:[0-9]+}`, promoReferralRouteHandler.ServeHTTP)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusInternalServerError, struct{}{})
	m.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
}
