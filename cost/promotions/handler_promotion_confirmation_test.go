package promotions

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

type mockDataAPIPromotionConfirmationHandler struct {
	api.DataAPI
	lookupPromoCodeParam         string
	lookupPromoCodeErr           error
	lookupPromoCode              *common.PromoCode
	referralProgramParam         int64
	referralProgramErr           error
	referralProgram              *common.ReferralProgram
	getPatientFromAccountIDParam int64
	getPatientFromAccountIDErr   error
	getPatientFromAccountID      *common.Patient
	getDoctorFromAccountIDParam  int64
	getDoctorFromAccountIDErr    error
	getDoctorFromAccountID       *common.Doctor
	promotionParam               int64
	promotionErr                 error
	promotion                    *common.Promotion
	referralProgramTemplateParam int64
	referralProgramTemplateErr   error
	referralProgramTemplate      *common.ReferralProgramTemplate
}

func (m *mockDataAPIPromotionConfirmationHandler) LookupPromoCode(code string) (*common.PromoCode, error) {
	m.lookupPromoCodeParam = code
	return m.lookupPromoCode, m.lookupPromoCodeErr
}

func (m *mockDataAPIPromotionConfirmationHandler) ReferralProgram(codeID int64, types map[string]reflect.Type) (*common.ReferralProgram, error) {
	m.referralProgramParam = codeID
	return m.referralProgram, m.referralProgramErr
}

func (m *mockDataAPIPromotionConfirmationHandler) GetPatientFromAccountID(accountID int64) (patient *common.Patient, err error) {
	m.getPatientFromAccountIDParam = accountID
	return m.getPatientFromAccountID, m.getPatientFromAccountIDErr
}

func (m *mockDataAPIPromotionConfirmationHandler) GetDoctorFromAccountID(accountID int64) (patient *common.Doctor, err error) {
	m.getDoctorFromAccountIDParam = accountID
	return m.getDoctorFromAccountID, m.getDoctorFromAccountIDErr
}

func (m *mockDataAPIPromotionConfirmationHandler) Promotion(codeID int64, types map[string]reflect.Type) (*common.Promotion, error) {
	m.promotionParam = codeID
	return m.promotion, m.promotionErr
}

func (m *mockDataAPIPromotionConfirmationHandler) ReferralProgramTemplate(id int64, types map[string]reflect.Type) (*common.ReferralProgramTemplate, error) {
	m.referralProgramTemplateParam = id
	return m.referralProgramTemplate, m.referralProgramTemplateErr
}

func TestPromotionConfirmationHandlerGETRequiresParams(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	dataAPI := &mockDataAPIPromotionConfirmationHandler{}
	handler := NewPromotionConfirmationHandler(dataAPI, &analytics.NullLogger{})
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestPromotionConfirmationHandlerGETNoPromotion(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?code=foo", nil)
	test.OK(t, err)
	dataAPI := &mockDataAPIPromotionConfirmationHandler{
		lookupPromoCodeErr: api.ErrNotFound(`promotion_code`),
	}
	handler := NewPromotionConfirmationHandler(dataAPI, &analytics.NullLogger{})
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, "foo", dataAPI.lookupPromoCodeParam)
	test.Equals(t, http.StatusNotFound, responseWriter.Code)
}

func TestPromotionConfirmationHandlerGETCodeLookupErr(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?code=foo", nil)
	test.OK(t, err)
	dataAPI := &mockDataAPIPromotionConfirmationHandler{
		lookupPromoCodeErr: errors.New("Foo"),
	}
	handler := NewPromotionConfirmationHandler(dataAPI, &analytics.NullLogger{})
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestPromotionConfirmationHandlerGETPromotionLookupErr(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?code=foo", nil)
	test.OK(t, err)
	dataAPI := &mockDataAPIPromotionConfirmationHandler{
		lookupPromoCode: &common.PromoCode{ID: 1, Code: "foo", IsReferral: false},
		promotionErr:    errors.New("Foo"),
	}
	handler := NewPromotionConfirmationHandler(dataAPI, &analytics.NullLogger{})
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, int64(1), dataAPI.promotionParam)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestPromotionConfirmationHandlerGETReferralLookupErr(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?code=foo", nil)
	test.OK(t, err)
	dataAPI := &mockDataAPIPromotionConfirmationHandler{
		lookupPromoCode:    &common.PromoCode{ID: 1, Code: "foo", IsReferral: true},
		referralProgramErr: errors.New("Foo"),
	}
	handler := NewPromotionConfirmationHandler(dataAPI, &analytics.NullLogger{})
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, int64(1), dataAPI.referralProgramParam)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestPromotionConfirmationHandlerGETReferralGetPatientFromAccountIDErr(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?code=foo", nil)
	test.OK(t, err)
	dataAPI := &mockDataAPIPromotionConfirmationHandler{
		lookupPromoCode:            &common.PromoCode{ID: 1, Code: "foo", IsReferral: true},
		referralProgram:            createReferralProgram(2, "imageURL", ptr.Int64(12345)),
		getPatientFromAccountIDErr: errors.New("Foo"),
	}
	handler := NewPromotionConfirmationHandler(dataAPI, &analytics.NullLogger{})
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, int64(2), dataAPI.getPatientFromAccountIDParam)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestPromotionConfirmationHandlerGETReferralPatientNotFoundGetDoctorFromAccountIDErr(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?code=foo", nil)
	test.OK(t, err)
	dataAPI := &mockDataAPIPromotionConfirmationHandler{
		lookupPromoCode:            &common.PromoCode{ID: 1, Code: "foo", IsReferral: true},
		referralProgram:            createReferralProgram(2, "imageURL", ptr.Int64(12345)),
		getPatientFromAccountIDErr: api.ErrNotFound(`patient`),
		getDoctorFromAccountIDErr:  errors.New("Foo"),
	}
	handler := NewPromotionConfirmationHandler(dataAPI, &analytics.NullLogger{})
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, int64(2), dataAPI.getDoctorFromAccountIDParam)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestPromotionConfirmationHandlerGETReferralProgramTemplateErr(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?code=foo", nil)
	test.OK(t, err)
	dataAPI := &mockDataAPIPromotionConfirmationHandler{
		lookupPromoCode:            &common.PromoCode{ID: 1, Code: "foo", IsReferral: true},
		referralProgram:            createReferralProgram(2, "imageURL", ptr.Int64(12345)),
		getPatientFromAccountID:    &common.Patient{FirstName: "FirstName"},
		referralProgramTemplateErr: errors.New("Foo"),
	}
	handler := NewPromotionConfirmationHandler(dataAPI, &analytics.NullLogger{})
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, int64(12345), dataAPI.referralProgramTemplateParam)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestPromotionConfirmationHandlerGETReferralPromotionErr(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?code=foo", nil)
	test.OK(t, err)
	dataAPI := &mockDataAPIPromotionConfirmationHandler{
		lookupPromoCode:         &common.PromoCode{ID: 1, Code: "foo", IsReferral: true},
		referralProgram:         createReferralProgram(2, "imageURL", ptr.Int64(12345)),
		getPatientFromAccountID: &common.Patient{FirstName: "FirstName"},
		referralProgramTemplate: &common.ReferralProgramTemplate{PromotionCodeID: ptr.Int64(10)},
		promotionErr:            errors.New("Foo"),
	}
	handler := NewPromotionConfirmationHandler(dataAPI, &analytics.NullLogger{})
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, int64(10), dataAPI.promotionParam)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestPromotionConfirmationHandlerGETReferralImageProvided(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?code=foo", nil)
	test.OK(t, err)
	dataAPI := &mockDataAPIPromotionConfirmationHandler{
		lookupPromoCode:         &common.PromoCode{ID: 1, Code: "foo", IsReferral: true},
		referralProgram:         createReferralProgram(2, "imageURL", ptr.Int64(12345)),
		getPatientFromAccountID: &common.Patient{FirstName: "FirstName"},
		referralProgramTemplate: &common.ReferralProgramTemplate{PromotionCodeID: ptr.Int64(10)},
		promotion:               createPromotion("imageURL", "", nil, 0),
	}
	handler := NewPromotionConfirmationHandler(dataAPI, &analytics.NullLogger{})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	httputil.JSONResponse(expectedWriter, http.StatusOK, &PromotionConfirmationGETResponse{
		Title:       "Your friend FirstName has given you a free visit.",
		ImageURL:    "imageURL",
		BodyText:    "successMsg",
		ButtonTitle: "Let's Go",
	})
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
	test.Equals(t, http.StatusOK, responseWriter.Code)
}

func TestPromotionConfirmationHandlerGETReferralDoctorImageNotProvided(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?code=foo", nil)
	test.OK(t, err)
	dataAPI := &mockDataAPIPromotionConfirmationHandler{
		lookupPromoCode:            &common.PromoCode{ID: 1, Code: "foo", IsReferral: true},
		referralProgram:            createReferralProgram(2, "", ptr.Int64(12345)),
		getPatientFromAccountIDErr: api.ErrNotFound(`patient`),
		referralProgramTemplate:    &common.ReferralProgramTemplate{PromotionCodeID: ptr.Int64(10)},
		promotion:                  createPromotion("", "", nil, 0),
	}
	handler := NewPromotionConfirmationHandler(dataAPI, &analytics.NullLogger{})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	httputil.JSONResponse(expectedWriter, http.StatusOK, &PromotionConfirmationGETResponse{
		Title:       "Welcome to Spruce",
		ImageURL:    DefaultPromotionImageURL,
		BodyText:    "successMsg",
		ButtonTitle: "Let's Go",
	})
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
	test.Equals(t, http.StatusOK, responseWriter.Code)
}

func TestPromotionConfirmationHandlerGETDoctorReferralProgramNoTemplateID(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?code=foo", nil)
	test.OK(t, err)
	dataAPI := &mockDataAPIPromotionConfirmationHandler{
		lookupPromoCode:            &common.PromoCode{ID: 1, Code: "foo", IsReferral: true},
		referralProgram:            createReferralProgram(2, "", nil),
		getPatientFromAccountIDErr: api.ErrNotFound(`patient`),
		referralProgramTemplate:    &common.ReferralProgramTemplate{PromotionCodeID: ptr.Int64(10)},
		promotion:                  createPromotion("", "", nil, 0),
	}
	handler := NewPromotionConfirmationHandler(dataAPI, &analytics.NullLogger{})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	httputil.JSONResponse(expectedWriter, http.StatusOK, &PromotionConfirmationGETResponse{
		Title:       "Welcome to Spruce",
		ImageURL:    DefaultPromotionImageURL,
		BodyText:    "successMsg",
		ButtonTitle: "Let's Go",
	})
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
	test.Equals(t, http.StatusOK, responseWriter.Code)
}

func TestPromotionConfirmationHandlerGETPromotionImage(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?code=foo", nil)
	test.OK(t, err)
	dataAPI := &mockDataAPIPromotionConfirmationHandler{
		lookupPromoCode: &common.PromoCode{ID: 1, Code: "foo", IsReferral: false},
		promotion:       createPromotion("imageURL", "", nil, 0),
	}
	handler := NewPromotionConfirmationHandler(dataAPI, &analytics.NullLogger{})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	httputil.JSONResponse(expectedWriter, http.StatusOK, &PromotionConfirmationGETResponse{
		Title:       "displayMsg",
		ImageURL:    "imageURL",
		BodyText:    "promoSuccessMsg",
		ButtonTitle: "Let's Go",
	})
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
	test.Equals(t, http.StatusOK, responseWriter.Code)
}

func TestPromotionConfirmationHandlerGETPromotionNoImage(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?code=foo", nil)
	test.OK(t, err)
	dataAPI := &mockDataAPIPromotionConfirmationHandler{
		lookupPromoCode: &common.PromoCode{ID: 1, Code: "foo", IsReferral: false},
		promotion:       createPromotion("", "", nil, 0),
	}
	handler := NewPromotionConfirmationHandler(dataAPI, &analytics.NullLogger{})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	httputil.JSONResponse(expectedWriter, http.StatusOK, &PromotionConfirmationGETResponse{
		Title:       "displayMsg",
		ImageURL:    DefaultPromotionImageURL,
		BodyText:    "promoSuccessMsg",
		ButtonTitle: "Let's Go",
	})
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
	test.Equals(t, http.StatusOK, responseWriter.Code)
}

func createReferralProgram(accountID int64, imageURL string, templateID *int64) *common.ReferralProgram {
	rp, _ := NewGiveReferralProgram("title", "description", "group", nil,
		NewPercentOffVisitPromotion(0,
			"group", "displayMsg", "shortMsg", "successMsg", imageURL,
			1, 1, true), nil, "", 0, 0)
	return &common.ReferralProgram{
		AccountID:  accountID,
		Data:       rp,
		TemplateID: templateID,
	}
}

func createPromotion(imageURL, group string, expires *time.Time, value int) *common.Promotion {
	p := NewPercentOffVisitPromotion(value,
		"group", "displayMsg", "shortMsg", "promoSuccessMsg", imageURL,
		1, 1, true)
	return &common.Promotion{
		Data:    p,
		Expires: expires,
		Group:   group,
	}
}
