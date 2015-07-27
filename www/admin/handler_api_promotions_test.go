package admin

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
)

type mockedDataAPI_promotionsHandler struct {
	api.DataAPI
	promotions           []*common.Promotion
	promotionsErr        error
	createPromotionErr   error
	createPromotionParam *common.Promotion
	lookupPromoCodeErr   error
	lookupPromoCode      *common.PromoCode
	lookupPromoCodeParam string
}

func (m *mockedDataAPI_promotionsHandler) Promotions(codeIDs []int64, promoTypes []string, types map[string]reflect.Type) ([]*common.Promotion, error) {
	return m.promotions, m.promotionsErr
}

func (m *mockedDataAPI_promotionsHandler) CreatePromotion(promotion *common.Promotion) (int64, error) {
	m.createPromotionParam = promotion
	return 1, m.createPromotionErr
}

func (m *mockedDataAPI_promotionsHandler) LookupPromoCode(code string) (*common.PromoCode, error) {
	m.lookupPromoCodeParam = code
	return m.lookupPromoCode, m.lookupPromoCodeErr
}

type TestTyped struct {
	Name string
}

func (t *TestTyped) TypeName() string {
	return t.Name
}

func TestPromotionHandlerGETQueriesDataLayer(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?", nil)
	test.OK(t, err)
	promotion := &common.Promotion{Data: &TestTyped{Name: "TestType"}, Created: time.Now()}
	handler := newPromotionsHandler(&mockedDataAPI_promotionsHandler{
		DataAPI:    &api.DataService{},
		promotions: []*common.Promotion{promotion},
	})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, &PromotionsGETResponse{Promotions: []*responses.Promotion{responses.TransformPromotion(promotion)}})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
}

func TestPromotionHandlerGETNoPromotions(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?", nil)
	test.OK(t, err)
	handler := newPromotionsHandler(&mockedDataAPI_promotionsHandler{
		DataAPI:       &api.DataService{},
		promotionsErr: api.ErrNotFound(`promotion`),
	})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, &PromotionsGETResponse{[]*responses.Promotion{}})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
}

func TestPromotionHandlerGETQueryErr(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?", nil)
	test.OK(t, err)
	handler := newPromotionsHandler(&mockedDataAPI_promotionsHandler{
		DataAPI:       &api.DataService{},
		promotionsErr: errors.New("Broked"),
	})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusInternalServerError, struct{}{})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
}

func TestPromotionHandlerGETBadParams(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?not_real=should_error", nil)
	test.OK(t, err)
	handler := newPromotionsHandler(&mockedDataAPI_promotionsHandler{DataAPI: &api.DataService{}})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusBadRequest, struct{}{})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
}

func TestPromotionHandlerPOSTWritesDataLayerNoExpiration(t *testing.T) {
	req, err := json.Marshal(&PromotionsPOSTRequest{
		Code:          "Foo",
		DataJSON:      "{}",
		PromotionType: "promo_percent_off",
		Group:         "new_user",
	})
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(req))
	test.OK(t, err)
	handler := newPromotionsHandler(&mockedDataAPI_promotionsHandler{DataAPI: &api.DataService{}, lookupPromoCodeErr: api.ErrNotFound(`promotion_code`)})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, &PromotionsPOSTResponse{PromoCodeID: 1})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
}

func TestPromotionHandlerPOSTWritesDataLayerExpiration(t *testing.T) {
	var n int64
	req, err := json.Marshal(&PromotionsPOSTRequest{
		Code:          "Foo",
		DataJSON:      "{}",
		PromotionType: "promo_percent_off",
		Group:         "new_user",
		Expires:       &n,
	})
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(req))
	test.OK(t, err)
	handler := newPromotionsHandler(&mockedDataAPI_promotionsHandler{DataAPI: &api.DataService{}, lookupPromoCodeErr: api.ErrNotFound(`promotion_code`)})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, &PromotionsPOSTResponse{PromoCodeID: 1})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
}

func TestPromotionHandlerPOSTBadPromoType(t *testing.T) {
	var n int64
	req, err := json.Marshal(&PromotionsPOSTRequest{
		Code:          "Foo",
		DataJSON:      "{}",
		PromotionType: "not_real",
		Group:         "new_user",
		Expires:       &n,
	})
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(req))
	test.OK(t, err)
	handler := newPromotionsHandler(&mockedDataAPI_promotionsHandler{DataAPI: &api.DataService{}, lookupPromoCodeErr: api.ErrNotFound(`promotion_code`)})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusBadRequest, struct{}{})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
}

func TestPromotionHandlerPOSTDataLayerErr(t *testing.T) {
	req, err := json.Marshal(&PromotionsPOSTRequest{
		Code:          "Foo",
		DataJSON:      "{}",
		PromotionType: "promo_percent_off",
		Group:         "new_user",
	})
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(req))
	test.OK(t, err)
	handler := newPromotionsHandler(&mockedDataAPI_promotionsHandler{
		DataAPI:            &api.DataService{},
		createPromotionErr: errors.New("Foo"),
		lookupPromoCodeErr: api.ErrNotFound(`promotion_code`),
	})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusInternalServerError, struct{}{})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
}

func TestPromotionHandlerPOSTRequiredParams(t *testing.T) {
	r, err := http.NewRequest("POST", "mock.api.request", strings.NewReader("{}"))
	test.OK(t, err)
	handler := newPromotionsHandler(&mockedDataAPI_promotionsHandler{DataAPI: &api.DataService{}})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusBadRequest, struct{}{})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
}

func TestPromotionHandlerPOSTCodeAlreadyExists(t *testing.T) {
	req, err := json.Marshal(&PromotionsPOSTRequest{
		Code:          "Foo",
		DataJSON:      "{}",
		PromotionType: "promo_percent_off",
		Group:         "new_user",
	})
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(req))
	test.OK(t, err)
	mh := &mockedDataAPI_promotionsHandler{DataAPI: &api.DataService{}}
	handler := newPromotionsHandler(mh)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusBadRequest, struct{}{})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, "Foo", mh.lookupPromoCodeParam)
}
