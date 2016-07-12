package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/regimensapi/internal/rxguide"
	rxtest "github.com/sprucehealth/backend/cmd/svc/regimensapi/internal/rxguide/test"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

func TestRXGuideGET(t *testing.T) {
	drugName := "Tretanoin Topical"
	rxGuide := &responses.RXGuide{GenericName: drugName}
	resp := &rxGuideHandlerGETResponse{RXGuide: rxGuide}
	svc := &rxtest.RXGuideService{Expector: &mock.Expector{T: t}}
	svc.Expect(mock.NewExpectation(svc.RXGuide, drugName))
	svc.RXGuideOutput = append(svc.RXGuideOutput, rxGuide)
	data, err := json.Marshal(resp)
	test.OK(t, err)

	h := NewRXGuide(svc)
	w := httptest.NewRecorder()
	r, err := http.NewRequest(httputil.Get, "/", nil)
	test.OK(t, err)
	ctx := mux.SetVars(context.Background(), map[string]string{"drug_name": drugName})
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)
	test.Equals(t, strings.TrimSpace(string(data)), strings.TrimSpace(w.Body.String()))
	svc.Finish()
}

func TestRXGuideGETNoGuide(t *testing.T) {
	drugName := "Tretanoin Topical"
	svc := &rxtest.RXGuideService{Expector: &mock.Expector{T: t}}
	svc.Expect(mock.NewExpectation(svc.RXGuide, drugName))
	svc.RXGuideOutput = append(svc.RXGuideOutput, nil)
	svc.RXGuideErrs = append(svc.RXGuideErrs, rxguide.ErrNoGuidesFound)

	h := NewRXGuide(svc)
	w := httptest.NewRecorder()
	r, err := http.NewRequest(httputil.Get, "/", nil)
	test.OK(t, err)
	ctx := mux.SetVars(context.Background(), map[string]string{"drug_name": drugName})
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusNotFound, w)
	svc.Finish()
}

func TestRXGuideGETInternalError(t *testing.T) {
	drugName := "Tretanoin Topical"
	svc := &rxtest.RXGuideService{Expector: &mock.Expector{T: t}}
	svc.Expect(mock.NewExpectation(svc.RXGuide, drugName))
	svc.RXGuideOutput = append(svc.RXGuideOutput, nil)
	svc.RXGuideErrs = append(svc.RXGuideErrs, errors.New("Random internal error"))

	h := NewRXGuide(svc)
	w := httptest.NewRecorder()
	r, err := http.NewRequest(httputil.Get, "/", nil)
	test.OK(t, err)
	ctx := mux.SetVars(context.Background(), map[string]string{"drug_name": drugName})
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusInternalServerError, w)
	svc.Finish()
}

func TestRXGuidePOST(t *testing.T) {
	rxGuide := &responses.RXGuide{GenericName: "Tretanoin Topical"}
	req := &responses.RXGuidePOSTRequest{RXGuide: rxGuide}
	svc := &rxtest.RXGuideService{Expector: &mock.Expector{T: t}}
	svc.Expect(mock.NewExpectation(svc.PutRXGuide, rxGuide))
	data, err := json.Marshal(req)
	test.OK(t, err)

	h := NewRXGuide(svc)
	w := httptest.NewRecorder()
	r, err := http.NewRequest(httputil.Post, "/", bytes.NewReader(data))
	test.OK(t, err)
	h.ServeHTTP(context.Background(), w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)
	svc.Finish()
}

func TestRXGuidePOSTGuideRequired(t *testing.T) {
	req := &responses.RXGuidePOSTRequest{}
	svc := &rxtest.RXGuideService{Expector: &mock.Expector{T: t}}
	data, err := json.Marshal(req)
	test.OK(t, err)

	h := NewRXGuide(svc)
	w := httptest.NewRecorder()
	r, err := http.NewRequest(httputil.Post, "/", bytes.NewReader(data))
	test.OK(t, err)
	h.ServeHTTP(context.Background(), w, r)
	test.HTTPResponseCode(t, http.StatusBadRequest, w)
	svc.Finish()
}

func TestRXGuidePOSTInternalError(t *testing.T) {
	rxGuide := &responses.RXGuide{GenericName: "Tretanoin Topical"}
	req := &responses.RXGuidePOSTRequest{RXGuide: rxGuide}
	svc := &rxtest.RXGuideService{Expector: &mock.Expector{T: t}}
	svc.Expect(mock.NewExpectation(svc.PutRXGuide, rxGuide))
	svc.PutRXGuideErrs = append(svc.PutRXGuideErrs, errors.New("Random internal error"))
	data, err := json.Marshal(req)
	test.OK(t, err)

	h := NewRXGuide(svc)
	w := httptest.NewRecorder()
	r, err := http.NewRequest(httputil.Post, "/", bytes.NewReader(data))
	test.OK(t, err)
	h.ServeHTTP(context.Background(), w, r)
	test.HTTPResponseCode(t, http.StatusInternalServerError, w)
	svc.Finish()
}
