package promotions

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_handler"
)

type mockDataAPIPatientPromotionsHandler struct {
	api.DataAPI
	lookupPromoCodeParam                        string
	lookupPromoCodeErr                          error
	lookupPromoCode                             *common.PromoCode
	referralProgramParam                        int64
	referralProgramErr                          error
	referralProgram                             *common.ReferralProgram
	activeReferralProgramForAccountParam        int64
	activeReferralProgramForAccountErr          error
	activeReferralProgramForAccount             *common.ReferralProgram
	getPatientFromAccountIDParam                int64
	getPatientFromAccountIDErr                  error
	getPatientFromAccountID                     *common.Patient
	promotionParam                              int64
	promotionErr                                error
	promotion                                   *common.Promotion
	patientLocationParam                        int64
	patientLocationErr                          error
	patientLocationState                        string
	promotionGroupParam                         string
	promotionGroupErr                           error
	promotionGroup                              *common.PromotionGroup
	promotionCountInGroupForAccountAccountParam int64
	promotionCountInGroupForAccountGroupParam   string
	promotionCountInGroupForAccountErr          error
	promotionCountInGroupForAccount             int
	pendingPromotionsForAccountParam            int64
	pendingPromotionsForAccountErr              error
	pendingPromotionsForAccount                 []*common.AccountPromotion
	deleteAccountPromotionAccountParam          int64
	deleteAccountPromotionCodeIDParam           int64
	deleteAccountPromotionErr                   error
}

func (m *mockDataAPIPatientPromotionsHandler) LookupPromoCode(code string) (*common.PromoCode, error) {
	m.lookupPromoCodeParam = code
	return m.lookupPromoCode, m.lookupPromoCodeErr
}

func (m *mockDataAPIPatientPromotionsHandler) ActiveReferralProgramForAccount(accountID int64, types map[string]reflect.Type) (*common.ReferralProgram, error) {
	m.activeReferralProgramForAccountParam = accountID
	return m.activeReferralProgramForAccount, m.activeReferralProgramForAccountErr
}

func (m *mockDataAPIPatientPromotionsHandler) ReferralProgram(codeID int64, types map[string]reflect.Type) (*common.ReferralProgram, error) {
	m.referralProgramParam = codeID
	return m.referralProgram, m.referralProgramErr
}

func (m *mockDataAPIPatientPromotionsHandler) GetPatientFromAccountID(accountID int64) (patient *common.Patient, err error) {
	m.getPatientFromAccountIDParam = accountID
	return m.getPatientFromAccountID, m.getPatientFromAccountIDErr
}

func (m *mockDataAPIPatientPromotionsHandler) Promotion(codeID int64, types map[string]reflect.Type) (*common.Promotion, error) {
	m.promotionParam = codeID
	return m.promotion, m.promotionErr
}

func (m *mockDataAPIPatientPromotionsHandler) PatientLocation(patientID int64) (zipcode string, state string, err error) {
	m.patientLocationParam = patientID
	return "", m.patientLocationState, m.patientLocationErr
}

func (m *mockDataAPIPatientPromotionsHandler) PromotionGroup(groupName string) (*common.PromotionGroup, error) {
	m.promotionGroupParam = groupName
	return m.promotionGroup, m.promotionGroupErr
}

func (m *mockDataAPIPatientPromotionsHandler) PromotionCountInGroupForAccount(accountID int64, group string) (int, error) {
	m.promotionCountInGroupForAccountAccountParam = accountID
	m.promotionCountInGroupForAccountGroupParam = group
	return m.promotionCountInGroupForAccount, m.promotionCountInGroupForAccountErr
}

func (m *mockDataAPIPatientPromotionsHandler) PendingPromotionsForAccount(accountID int64, types map[string]reflect.Type) ([]*common.AccountPromotion, error) {
	m.pendingPromotionsForAccountParam = accountID
	return m.pendingPromotionsForAccount, m.pendingPromotionsForAccountErr
}

func (m *mockDataAPIPatientPromotionsHandler) DeleteAccountPromotion(accountID, codeID int64) (int64, error) {
	m.deleteAccountPromotionAccountParam = accountID
	m.deleteAccountPromotionCodeIDParam = codeID
	return 0, m.deleteAccountPromotionErr
}

type mockAuthAPIPatientPromotionsHandler struct {
	api.AuthAPI
	accountForEmailParam string
	accountForEmailErr   error
	accountForEmail      *common.Account
}

func (m *mockAuthAPIPatientPromotionsHandler) AccountForEmail(email string) (*common.Account, error) {
	m.accountForEmailParam = email
	return m.accountForEmail, m.accountForEmailErr
}

func TestPatientPromotionsHandlerGETPendingPromotionsErr(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	dataAPI := &mockDataAPIPatientPromotionsHandler{
		DataAPI: &api.DataService{},
		pendingPromotionsForAccountErr: errors.New("Foo"),
	}
	handler := test_handler.MockHandler{
		H: NewPatientPromotionsHandler(dataAPI, &mockAuthAPIPatientPromotionsHandler{}, &analytics.NullLogger{}),
	}

	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestPatientPromotionsHandlerGETNoPromotions(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	dataAPI := &mockDataAPIPatientPromotionsHandler{
		DataAPI:                     &api.DataService{},
		pendingPromotionsForAccount: []*common.AccountPromotion{},
	}
	handler := test_handler.MockHandler{
		H: NewPatientPromotionsHandler(dataAPI, &mockAuthAPIPatientPromotionsHandler{}, &analytics.NullLogger{}),
	}

	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, &PatientPromotionGETResponse{
		ActivePromotions:  []*responses.ClientPromotion{},
		ExpiredPromotions: []*responses.ClientPromotion{},
	})
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusOK, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
}

func TestPatientPromotionsHandlerGETActiveAndExpiredPromosNonPatientVisible(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	activeZeroValueAttributionPromo := createAccountPromotion("ActivePromo1", "attribution", nil, 0)
	expiredNewUserPromo := createAccountPromotion("ExpiredPromo1", "new_user", ptr.Time(time.Unix(time.Now().Unix()-1, 0)), 1)
	activeCreditPromo := createAccountPromotion("ActivePromo2", "credit", nil, 1)
	expiredCreditPromo := createAccountPromotion("ExpiredPromo2", "credit", ptr.Time(time.Unix(time.Now().Unix()-1, 0)), 1)
	dataAPI := &mockDataAPIPatientPromotionsHandler{
		DataAPI: &api.DataService{},
		pendingPromotionsForAccount: []*common.AccountPromotion{
			activeZeroValueAttributionPromo,
			expiredNewUserPromo,
			activeCreditPromo,
			expiredCreditPromo,
		},
	}
	handler := test_handler.MockHandler{
		H: NewPatientPromotionsHandler(dataAPI, &mockAuthAPIPatientPromotionsHandler{}, &analytics.NullLogger{}),
	}

	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, &PatientPromotionGETResponse{
		ActivePromotions: []*responses.ClientPromotion{
			&responses.ClientPromotion{
				Code:                 activeCreditPromo.Code,
				Description:          "successMsg",
				DescriptionHasTokens: false,
				ExpirationDate:       0,
			},
		},
		ExpiredPromotions: []*responses.ClientPromotion{
			&responses.ClientPromotion{
				Code:                 expiredNewUserPromo.Code,
				Description:          "successMsg - Expires <expiration_date>",
				DescriptionHasTokens: true,
				ExpirationDate:       expiredNewUserPromo.Expires.Unix(),
			},
			&responses.ClientPromotion{
				Code:                 expiredCreditPromo.Code,
				Description:          "successMsg - Expires <expiration_date>",
				DescriptionHasTokens: true,
				ExpirationDate:       expiredCreditPromo.Expires.Unix(),
			},
		},
	})
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusOK, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
}

func TestPatientPromotionsHandlerPOSTPromoCodeRequired(t *testing.T) {
	rb, err := json.Marshal(&PatientPromotionPOSTRequest{})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(rb))
	test.OK(t, err)
	dataAPI := &mockDataAPIPatientPromotionsHandler{
		DataAPI: &api.DataService{},
	}
	handler := test_handler.MockHandler{
		H: NewPatientPromotionsHandler(dataAPI, &mockAuthAPIPatientPromotionsHandler{}, &analytics.NullLogger{}),
	}

	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestPatientPromotionsHandlerPOSTLookupPromoCodeErr(t *testing.T) {
	rb, err := json.Marshal(&PatientPromotionPOSTRequest{PromoCode: "Foo"})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(rb))
	test.OK(t, err)
	dataAPI := &mockDataAPIPatientPromotionsHandler{
		DataAPI:            &api.DataService{},
		lookupPromoCodeErr: errors.New("Foo"),
	}
	handler := test_handler.MockHandler{
		H: NewPatientPromotionsHandler(dataAPI, &mockAuthAPIPatientPromotionsHandler{}, &analytics.NullLogger{}),
	}

	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, "Foo", dataAPI.lookupPromoCodeParam)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestPatientPromotionsHandlerPOSTLookupPromoCodeNotFound(t *testing.T) {
	rb, err := json.Marshal(&PatientPromotionPOSTRequest{PromoCode: "Foo"})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(rb))
	test.OK(t, err)
	dataAPI := &mockDataAPIPatientPromotionsHandler{
		DataAPI:            &api.DataService{},
		lookupPromoCodeErr: api.ErrNotFound(`promotion_code`),
	}
	handler := test_handler.MockHandler{
		H: NewPatientPromotionsHandler(dataAPI, &mockAuthAPIPatientPromotionsHandler{}, &analytics.NullLogger{}),
	}

	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusNotFound, responseWriter.Code)
}

func TestPatientPromotionsHandlerPOSTPromotionErr(t *testing.T) {
	rb, err := json.Marshal(&PatientPromotionPOSTRequest{PromoCode: "Foo"})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(rb))
	test.OK(t, err)
	dataAPI := &mockDataAPIPatientPromotionsHandler{
		DataAPI:         &api.DataService{},
		lookupPromoCode: &common.PromoCode{ID: 1, Code: "Foo", IsReferral: false},
		promotionErr:    errors.New("foo"),
	}
	handler := test_handler.MockHandler{
		H: NewPatientPromotionsHandler(dataAPI, &mockAuthAPIPatientPromotionsHandler{}, &analytics.NullLogger{}),
	}

	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, int64(1), dataAPI.promotionParam)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestPatientPromotionsHandlerPOSTPromotionExpired(t *testing.T) {
	rb, err := json.Marshal(&PatientPromotionPOSTRequest{PromoCode: "Foo"})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(rb))
	test.OK(t, err)
	dataAPI := &mockDataAPIPatientPromotionsHandler{
		DataAPI:         &api.DataService{},
		lookupPromoCode: &common.PromoCode{ID: 1, Code: "Foo", IsReferral: false},
		promotion:       createPromotion("imageURL", "test_group", ptr.Time(time.Unix(time.Now().Unix()-1, 0)), 1),
	}
	handler := test_handler.MockHandler{
		H: NewPatientPromotionsHandler(dataAPI, &mockAuthAPIPatientPromotionsHandler{}, &analytics.NullLogger{}),
	}

	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusNotFound, responseWriter.Code)
}

func TestPatientPromotionsHandlerPOSTActiveReferralProgramErr(t *testing.T) {
	rb, err := json.Marshal(&PatientPromotionPOSTRequest{PromoCode: "Foo"})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(rb))
	test.OK(t, err)
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 12345
	dataAPI := &mockDataAPIPatientPromotionsHandler{
		DataAPI:                            &api.DataService{},
		lookupPromoCode:                    &common.PromoCode{ID: 1, Code: "Foo", IsReferral: true},
		activeReferralProgramForAccountErr: errors.New("foo"),
	}
	handler := test_handler.MockHandler{
		H: NewPatientPromotionsHandler(dataAPI, &mockAuthAPIPatientPromotionsHandler{}, &analytics.NullLogger{}),
	}

	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, int64(12345), dataAPI.activeReferralProgramForAccountParam)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestPatientPromotionsHandlerPOSTClaimOwnReferralCode(t *testing.T) {
	rb, err := json.Marshal(&PatientPromotionPOSTRequest{PromoCode: "Foo"})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(rb))
	test.OK(t, err)
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 12345
	rp := createReferralProgram(ctxt.AccountID, "imageURL")
	rp.CodeID = 12345
	dataAPI := &mockDataAPIPatientPromotionsHandler{
		DataAPI:                         &api.DataService{},
		lookupPromoCode:                 &common.PromoCode{ID: rp.CodeID, Code: "Foo", IsReferral: true},
		activeReferralProgramForAccount: rp,
	}
	handler := test_handler.MockHandler{
		H: NewPatientPromotionsHandler(dataAPI, &mockAuthAPIPatientPromotionsHandler{}, &analytics.NullLogger{}),
	}

	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusNotFound, responseWriter.Code)
}

func TestPatientPromotionsHandlerPOSTReferralProgramErr(t *testing.T) {
	rb, err := json.Marshal(&PatientPromotionPOSTRequest{PromoCode: "Foo"})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(rb))
	test.OK(t, err)
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 12345
	dataAPI := &mockDataAPIPatientPromotionsHandler{
		DataAPI:                         &api.DataService{},
		lookupPromoCode:                 &common.PromoCode{ID: 12345, Code: "Foo", IsReferral: true},
		activeReferralProgramForAccount: createReferralProgram(ctxt.AccountID, "imageURL"),
		referralProgramErr:              errors.New("foo"),
	}
	handler := test_handler.MockHandler{
		H: NewPatientPromotionsHandler(dataAPI, &mockAuthAPIPatientPromotionsHandler{}, &analytics.NullLogger{}),
	}

	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, int64(12345), dataAPI.referralProgramParam)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestPatientPromotionsHandlerPOSTGetPatientFromAccountIDErr(t *testing.T) {
	rb, err := json.Marshal(&PatientPromotionPOSTRequest{PromoCode: "Foo"})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(rb))
	test.OK(t, err)
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 12345
	dataAPI := &mockDataAPIPatientPromotionsHandler{
		DataAPI:                    &api.DataService{},
		lookupPromoCode:            &common.PromoCode{ID: 12345, Code: "Foo", IsReferral: false},
		promotion:                  createPromotion("imageURL", "test_group", nil, 1),
		getPatientFromAccountIDErr: errors.New("foo"),
	}
	handler := test_handler.MockHandler{
		H: NewPatientPromotionsHandler(dataAPI, &mockAuthAPIPatientPromotionsHandler{}, &analytics.NullLogger{}),
	}

	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, int64(12345), dataAPI.getPatientFromAccountIDParam)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestPatientPromotionsHandlerPOSTPatientLocationErr(t *testing.T) {
	rb, err := json.Marshal(&PatientPromotionPOSTRequest{PromoCode: "Foo"})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(rb))
	test.OK(t, err)
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 12345
	dataAPI := &mockDataAPIPatientPromotionsHandler{
		DataAPI:                 &api.DataService{},
		lookupPromoCode:         &common.PromoCode{ID: 12345, Code: "Foo", IsReferral: false},
		promotion:               createPromotion("imageURL", "test_group", nil, 1),
		getPatientFromAccountID: &common.Patient{ID: encoding.NewObjectID(54321)},
		patientLocationErr:      errors.New("foo"),
	}
	handler := test_handler.MockHandler{
		H: NewPatientPromotionsHandler(dataAPI, &mockAuthAPIPatientPromotionsHandler{}, &analytics.NullLogger{}),
	}

	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, int64(54321), dataAPI.patientLocationParam)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestPatientPromotionsHandlerPOSTPromotionGroupErr(t *testing.T) {
	rb, err := json.Marshal(&PatientPromotionPOSTRequest{PromoCode: "Foo"})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(rb))
	test.OK(t, err)
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 12345
	dataAPI := &mockDataAPIPatientPromotionsHandler{
		DataAPI:                 &api.DataService{},
		lookupPromoCode:         &common.PromoCode{ID: 12345, Code: "Foo", IsReferral: false},
		promotion:               createPromotion("imageURL", "test_group", nil, 1),
		getPatientFromAccountID: &common.Patient{ID: encoding.NewObjectID(54321)},
		patientLocationState:    "CA",
		promotionGroupErr:       errors.New("foo"),
	}
	handler := test_handler.MockHandler{
		H: NewPatientPromotionsHandler(dataAPI, &mockAuthAPIPatientPromotionsHandler{}, &analytics.NullLogger{}),
	}

	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, "test_group", dataAPI.promotionGroupParam)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestPatientPromotionsHandlerPOSTPromotionCountInGroupForAccountErr(t *testing.T) {
	rb, err := json.Marshal(&PatientPromotionPOSTRequest{PromoCode: "Foo"})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(rb))
	test.OK(t, err)
	ctxt := apiservice.GetContext(r)
	ctxt.AccountID = 12345
	dataAPI := &mockDataAPIPatientPromotionsHandler{
		DataAPI:                            &api.DataService{},
		lookupPromoCode:                    &common.PromoCode{ID: 12345, Code: "Foo", IsReferral: false},
		promotion:                          createPromotion("imageURL", "test_group", nil, 1),
		getPatientFromAccountID:            &common.Patient{ID: encoding.NewObjectID(54321)},
		patientLocationState:               "CA",
		promotionGroup:                     &common.PromotionGroup{Name: "test_group", MaxAllowedPromos: 1},
		promotionCountInGroupForAccountErr: errors.New("foo"),
	}
	handler := test_handler.MockHandler{
		H: NewPatientPromotionsHandler(dataAPI, &mockAuthAPIPatientPromotionsHandler{}, &analytics.NullLogger{}),
	}

	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, int64(12345), dataAPI.promotionCountInGroupForAccountAccountParam)
	test.Equals(t, "test_group", dataAPI.promotionCountInGroupForAccountGroupParam)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func createAccountPromotion(code, group string, expires *time.Time, value int) *common.AccountPromotion {
	return &common.AccountPromotion{
		Code:    code,
		Group:   group,
		Expires: expires,
		Data: NewPercentOffVisitPromotion(value,
			"group", "displayMsg", "shortMsg", "successMsg", "imageURL",
			1, 1, true),
	}
}
