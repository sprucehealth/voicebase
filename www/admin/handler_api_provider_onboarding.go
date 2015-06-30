package admin

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/www"
)

var onboardTimeExpirationDef = &cfg.ValueDef{
	Name:        "DrOnboarding.URLExpiration",
	Description: "Duration for which a doctor onboarding URL is valid",
	Type:        cfg.ValueTypeDuration,
	Default:     time.Hour * 24 * 14,
}

type providerOnboardingURLAPIHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
	signer  *sig.Signer
}

func NewProviderOnboardingURLAPIHandler(r *mux.Router, dataAPI api.DataAPI, signer *sig.Signer, cfgStore cfg.Store) http.Handler {
	cfgStore.Register(onboardTimeExpirationDef)
	return httputil.SupportedMethods(&providerOnboardingURLAPIHandler{
		router:  r,
		dataAPI: dataAPI,
		signer:  signer,
	}, httputil.Get)
}

func (h *providerOnboardingURLAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "GenerateProviderOnboardingURL", nil)

	cfgSnap := cfg.Context(r)

	nonceBytes := make([]byte, 8)
	if _, err := rand.Read(nonceBytes); err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	nonce := base64.StdEncoding.EncodeToString(nonceBytes)
	expires := time.Now().Add(cfgSnap.Duration(onboardTimeExpirationDef.Name)).Unix()
	msg := []byte(fmt.Sprintf("expires=%d&nonce=%s", expires, nonce))
	sig, err := h.signer.Sign(msg)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	sigStr := base64.StdEncoding.EncodeToString(sig)

	u, err := h.router.Get("doctor-register-intro").URLPath()
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	u.Scheme = "https"
	u.Host = r.Host
	u.RawQuery = (url.Values{
		"e": []string{strconv.FormatInt(expires, 10)},
		"n": []string{nonce},
		"s": []string{sigStr},
	}).Encode()

	httputil.JSONResponse(w, http.StatusOK, u.String())
}
