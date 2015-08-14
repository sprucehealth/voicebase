package admin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type providerSearchDataAPI struct {
	api.DataAPI
}

type providerSearchAuthAPI struct {
	api.AuthAPI
}

func (d *providerSearchDataAPI) RegisterProvider(provider *common.Doctor, role string) (int64, error) {
	return 2, nil
}

func (a *providerSearchAuthAPI) CreateAccount(email, password, role string) (int64, error) {
	if !strings.HasSuffix(email, "@sprucehealth.com") {
		return 0, fmt.Errorf("bad email %s", email)
	}
	if role == "" {
		return 0, errors.New("role not provided")
	}
	return 1, nil
}

func TestHandlerProviderSearchAPI(t *testing.T) {
	dataAPI := &providerSearchDataAPI{}
	authAPI := &providerSearchAuthAPI{}
	h := newProviderSearchAPIHandler(dataAPI, authAPI)

	body := &bytes.Buffer{}
	r, err := http.NewRequest("POST", "/", body)
	test.OK(t, err)
	account := &common.Account{Role: api.RoleAdmin, ID: 1}
	ctx := www.CtxWithAccount(context.Background(), account)

	// This error comes from the JSON decoder
	body.Reset()
	test.OK(t, json.NewEncoder(body).Encode(&createProviderRequest{}))
	w := httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusBadRequest, w.Code)
	test.Equals(t, "{\"error\":{\"type\":\"bad_request\",\"message\":\"Phone number has to be atleast 10 digits long\"}}\n", w.Body.String())

	// This error comes from .validate()
	body.Reset()
	test.OK(t, json.NewEncoder(body).Encode(&createProviderRequest{Role: "blah", CellPhone: "415-555-5555"}))
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusBadRequest, w.Code)
	test.Equals(t, "{\"error\":{\"type\":\"bad_request\",\"message\":\"role must be MA or DOCTOR\"}}\n", w.Body.String())

	body.Reset()
	test.OK(t, json.NewEncoder(body).Encode(&createProviderRequest{
		Role:      api.RoleCC,
		CellPhone: "415-555-5555",
		Email:     "test+doctor@sprucehealth.com",
		FirstName: "first",
		LastName:  "last",
		DOB:       encoding.Date{Year: 1980, Month: 1, Day: 1},
		Gender:    "female",
	}))
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)
	test.Equals(t, "{\"account_id\":\"1\",\"provider_id\":\"2\"}\n", w.Body.String())
}
