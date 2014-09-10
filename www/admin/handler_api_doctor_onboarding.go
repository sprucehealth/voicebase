package admin

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type doctorOnboardingURLAPIHandler struct {
	router               *mux.Router
	dataAPI              api.DataAPI
	signer               *common.Signer
	onboardingURLExpires int64
}

func NewDoctorOnboardingURLAPIHandler(r *mux.Router, dataAPI api.DataAPI, signer *common.Signer, onboardingURLExpires int64) http.Handler {
	return httputil.SupportedMethods(&doctorOnboardingURLAPIHandler{
		router:               r,
		dataAPI:              dataAPI,
		signer:               signer,
		onboardingURLExpires: onboardingURLExpires,
	}, []string{"GET"})
}

func (h *doctorOnboardingURLAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "GenerateDoctorOnboardingURL", nil)

	nonceBytes := make([]byte, 8)
	if _, err := rand.Read(nonceBytes); err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	nonce := base64.StdEncoding.EncodeToString(nonceBytes)
	expires := time.Now().UTC().Unix() + h.onboardingURLExpires
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

	www.JSONResponse(w, r, http.StatusOK, u.String())
}
