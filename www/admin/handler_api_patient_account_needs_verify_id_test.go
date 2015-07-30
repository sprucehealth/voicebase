package admin

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/test"
)

type mockedNeedsIDMarker struct {
	markForNeedsIDVerificationPatientIDParam int64
	markForNeedsIDVerificationPromoCodeParam string
	markForNeedsIDVerificationErr            error
}

func (m *mockedNeedsIDMarker) MarkForNeedsIDVerification(patientID int64, promoCode string) error {
	m.markForNeedsIDVerificationPatientIDParam = patientID
	m.markForNeedsIDVerificationPromoCodeParam = promoCode
	return m.markForNeedsIDVerificationErr
}

func TestPatientAccountNeedsVerifyIDHandlerPOSTBadURL(t *testing.T) {
	req, err := json.Marshal(&patientAccountNeedsVerifyIDPOSTRequest{
		PromoCode: "Doesn't Matter",
	})
	r, err := http.NewRequest("POST", "/foo/bar", bytes.NewReader(req))
	test.OK(t, err)
	m := mux.NewRouter()
	handler := newPatientAccountNeedsVerifyIDHandler(&mockedNeedsIDMarker{})
	m.Handle(`/foo/{id:[0-9]+}`, handler)
	responseWriter := httptest.NewRecorder()
	m.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, http.StatusNotFound, responseWriter.Code)
}

func TestPatientAccountNeedsVerifyIDHandlerPOSTRequiresPromoCode(t *testing.T) {
	req, err := json.Marshal(&patientAccountNeedsVerifyIDPOSTRequest{})
	r, err := http.NewRequest("POST", "/foo/100", bytes.NewReader(req))
	test.OK(t, err)
	m := mux.NewRouter()
	handler := newPatientAccountNeedsVerifyIDHandler(&mockedNeedsIDMarker{})
	m.Handle(`/foo/{id:[0-9]+}`, handler)
	responseWriter := httptest.NewRecorder()
	m.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestPatientAccountNeedsVerifyIDHandlerPOSTSuccess(t *testing.T) {
	promoCode := "PROMO"
	req, err := json.Marshal(&patientAccountNeedsVerifyIDPOSTRequest{PromoCode: promoCode})
	r, err := http.NewRequest("POST", "/foo/100", bytes.NewReader(req))
	test.OK(t, err)
	m := mux.NewRouter()
	ms := &mockedNeedsIDMarker{}
	handler := newPatientAccountNeedsVerifyIDHandler(ms)
	m.Handle(`/foo/{id:[0-9]+}`, handler)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, struct{}{})
	m.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
	test.Equals(t, http.StatusOK, responseWriter.Code)
	test.Equals(t, promoCode, ms.markForNeedsIDVerificationPromoCodeParam)
	test.Equals(t, int64(100), ms.markForNeedsIDVerificationPatientIDParam)
}

func TestPatientAccountNeedsVerifyIDHandlerPOSTNotFoundResource(t *testing.T) {
	req, err := json.Marshal(&patientAccountNeedsVerifyIDPOSTRequest{PromoCode: "Promo"})
	r, err := http.NewRequest("POST", "/foo/100", bytes.NewReader(req))
	test.OK(t, err)
	m := mux.NewRouter()
	handler := newPatientAccountNeedsVerifyIDHandler(&mockedNeedsIDMarker{markForNeedsIDVerificationErr: api.ErrNotFound(`anything`)})
	m.Handle(`/foo/{id:[0-9]+}`, handler)
	responseWriter := httptest.NewRecorder()
	m.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestPatientAccountNeedsVerifyIDHandlerPOSTInternalErr(t *testing.T) {
	req, err := json.Marshal(&patientAccountNeedsVerifyIDPOSTRequest{PromoCode: "Promo"})
	r, err := http.NewRequest("POST", "/foo/100", bytes.NewReader(req))
	test.OK(t, err)
	m := mux.NewRouter()
	handler := newPatientAccountNeedsVerifyIDHandler(&mockedNeedsIDMarker{markForNeedsIDVerificationErr: errors.New("Foo")})
	m.Handle(`/foo/{id:[0-9]+}`, handler)
	responseWriter := httptest.NewRecorder()
	m.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}
