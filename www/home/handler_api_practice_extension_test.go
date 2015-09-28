package home

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"

	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/test"
)

func TestPracticeExtensionDemoRequestAPIHandler_POST(t *testing.T) {

	cfgStore, err := cfg.NewLocalStore(config.CfgDefs())
	test.OK(t, err)
	cfgStore.Update(map[string]interface{}{
		demoRequestSlackWebhookURLDef.Name: "",
	})

	h := newPracticeExtensionDemoAPIHandler(cfgStore)

	// Success

	body, err := json.Marshal(&demoPOSTRequest{
		FirstName: "jon",
		LastName:  "sibley",
		Email:     "jon@sprucehealth.com",
		Phone:     "415-915-9986",
		State:     "IL",
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	w := httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)

	// Failure

	body, err = json.Marshal(&demoPOSTRequest{})
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.HTTPResponseCode(t, http.StatusBadRequest, w)
}

func TestPracticeExtensionWhitepaperRequestAPIHandler_POST(t *testing.T) {

	cfgStore, err := cfg.NewLocalStore(config.CfgDefs())
	test.OK(t, err)
	cfgStore.Update(map[string]interface{}{
		whitepaperRequestSlackWebhookURLDef.Name: "",
	})

	h := newPracticeExtensionWhitepaperAPIHandler(cfgStore)

	// Success

	body, err := json.Marshal(&whitepaperPOSTRequest{
		FirstName: "jon",
		LastName:  "sibley",
		Email:     "jon@sprucehealth.com",
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	w := httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)

	// Failure

	body, err = json.Marshal(&whitepaperPOSTRequest{})
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", bytes.NewReader(body))
	test.OK(t, err)
	w = httptest.NewRecorder()
	h.ServeHTTP(context.Background(), w, r)
	test.HTTPResponseCode(t, http.StatusBadRequest, w)
}
