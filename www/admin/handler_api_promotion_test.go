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

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_handler"
)

type mockedDataAPI_promotionHandler struct {
	api.DataAPI
	promotions           []*common.Promotion
	promotionsErr        error
	createPromotionErr   error
	createPromotionParam *common.Promotion
	lookupPromoCodeErr   error
	lookupPromoCode      *common.PromoCode
	lookupPromoCodeParam string
}

func (m *mockedDataAPI_promotionHandler) Promotions(codeIDs []int64, promoTypes []string, types map[string]reflect.Type) ([]*common.Promotion, error) {
	return m.promotions, m.promotionsErr
}

func (m *mockedDataAPI_promotionHandler) CreatePromotion(promotion *common.Promotion) (int64, error) {
	m.createPromotionParam = promotion
	return 1, m.createPromotionErr
}

func (m *mockedDataAPI_promotionHandler) LookupPromoCode(code string) (*common.PromoCode, error) {
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
	promoHandler := NewPromotionHandler(&mockedDataAPI_promotionHandler{
		DataAPI:    &api.DataService{},
		promotions: []*common.Promotion{promotion},
	})
	handler := test_handler.MockHandler{
		H: promoHandler,
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, &PromotionGETResponse{Promotions: []*responses.Promotion{responses.TransformPromotion(promotion)}})
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
}

func TestPromotionHandlerGETNoPromotions(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?", nil)
	test.OK(t, err)
	promoHandler := NewPromotionHandler(&mockedDataAPI_promotionHandler{
		DataAPI:       &api.DataService{},
		promotionsErr: api.ErrNotFound(`promotion`),
	})
	handler := test_handler.MockHandler{
		H: promoHandler,
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, &PromotionGETResponse{[]*responses.Promotion{}})
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
}

func TestPromotionHandlerGETQueryErr(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?", nil)
	test.OK(t, err)
	promoHandler := NewPromotionHandler(&mockedDataAPI_promotionHandler{
		DataAPI:       &api.DataService{},
		promotionsErr: errors.New("Broked"),
	})
	handler := test_handler.MockHandler{
		H: promoHandler,
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusInternalServerError, struct{}{})
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
}

func TestPromotionHandlerGETBadParams(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?not_real=should_error", nil)
	test.OK(t, err)
	promoHandler := NewPromotionHandler(&mockedDataAPI_promotionHandler{DataAPI: &api.DataService{}})
	handler := test_handler.MockHandler{
		H: promoHandler,
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusBadRequest, struct{}{})
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
}

func TestPromotionHandlerPOSTWritesDataLayerNoExpiration(t *testing.T) {
	req, err := json.Marshal(&PromotionPOSTRequest{
		Code:          "Foo",
		DataJSON:      "{}",
		PromotionType: "promo_percent_off",
		Group:         "new_user",
	})
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(req))
	test.OK(t, err)
	promoHandler := NewPromotionHandler(&mockedDataAPI_promotionHandler{DataAPI: &api.DataService{}, lookupPromoCodeErr: api.ErrNotFound(`promotion_code`)})
	handler := test_handler.MockHandler{
		H: promoHandler,
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, &PromotionPOSTResponse{PromoCodeID: 1})
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
}

func TestPromotionHandlerPOSTWritesDataLayerExpiration(t *testing.T) {
	var n int64
	req, err := json.Marshal(&PromotionPOSTRequest{
		Code:          "Foo",
		DataJSON:      "{}",
		PromotionType: "promo_percent_off",
		Group:         "new_user",
		Expires:       &n,
	})
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(req))
	test.OK(t, err)
	promoHandler := NewPromotionHandler(&mockedDataAPI_promotionHandler{DataAPI: &api.DataService{}, lookupPromoCodeErr: api.ErrNotFound(`promotion_code`)})
	handler := test_handler.MockHandler{
		H: promoHandler,
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, &PromotionPOSTResponse{PromoCodeID: 1})
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
}

func TestPromotionHandlerPOSTBadPromoType(t *testing.T) {
	var n int64
	req, err := json.Marshal(&PromotionPOSTRequest{
		Code:          "Foo",
		DataJSON:      "{}",
		PromotionType: "not_real",
		Group:         "new_user",
		Expires:       &n,
	})
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(req))
	test.OK(t, err)
	promoHandler := NewPromotionHandler(&mockedDataAPI_promotionHandler{DataAPI: &api.DataService{}, lookupPromoCodeErr: api.ErrNotFound(`promotion_code`)})
	handler := test_handler.MockHandler{
		H: promoHandler,
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusBadRequest, struct{}{})
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
}

func TestPromotionHandlerPOSTDataLayerErr(t *testing.T) {
	req, err := json.Marshal(&PromotionPOSTRequest{
		Code:          "Foo",
		DataJSON:      "{}",
		PromotionType: "promo_percent_off",
		Group:         "new_user",
	})
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(req))
	test.OK(t, err)
	promoHandler := NewPromotionHandler(&mockedDataAPI_promotionHandler{
		DataAPI:            &api.DataService{},
		createPromotionErr: errors.New("Foo"),
		lookupPromoCodeErr: api.ErrNotFound(`promotion_code`),
	})
	handler := test_handler.MockHandler{
		H: promoHandler,
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusInternalServerError, struct{}{})
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
}

func TestPromotionHandlerPOSTRequiredParams(t *testing.T) {
	r, err := http.NewRequest("POST", "mock.api.request", strings.NewReader("{}"))
	test.OK(t, err)
	promoHandler := NewPromotionHandler(&mockedDataAPI_promotionHandler{DataAPI: &api.DataService{}})
	handler := test_handler.MockHandler{
		H: promoHandler,
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusBadRequest, struct{}{})
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
}

func TestPromotionHandlerPOSTCodeAlreadyExists(t *testing.T) {
	req, err := json.Marshal(&PromotionPOSTRequest{
		Code:          "Foo",
		DataJSON:      "{}",
		PromotionType: "promo_percent_off",
		Group:         "new_user",
	})
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(req))
	test.OK(t, err)
	mh := &mockedDataAPI_promotionHandler{DataAPI: &api.DataService{}}
	promoHandler := NewPromotionHandler(mh)
	handler := test_handler.MockHandler{
		H: promoHandler,
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusBadRequest, struct{}{})
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Code, responseWriter.Code)
	test.Equals(t, "Foo", mh.lookupPromoCodeParam)
}
