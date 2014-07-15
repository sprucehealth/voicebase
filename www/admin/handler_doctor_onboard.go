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
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

const (
	onboardExpires = 60 * 60 * 24 * 14 // seconds
)

type doctorOnboardHandler struct {
	router  *mux.Router
	dataAPI api.DataAPI
	signer  *common.Signer
}

func NewDoctorOnboardHandler(router *mux.Router, dataAPI api.DataAPI, signer *common.Signer) http.Handler {
	return httputil.SupportedMethods(&doctorOnboardHandler{
		router:  router,
		dataAPI: dataAPI,
		signer:  signer,
	}, []string{"GET"})
}

func (h *doctorOnboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	nonceBytes := make([]byte, 8)
	if _, err := rand.Read(nonceBytes); err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	nonce := base64.StdEncoding.EncodeToString(nonceBytes)
	expires := time.Now().UTC().Unix() + onboardExpires
	msg := []byte(fmt.Sprintf("expires=%d&nonce=%s", expires, nonce))
	sig, err := h.signer.Sign(msg)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}
	sigStr := base64.StdEncoding.EncodeToString(sig)

	u, err := h.router.Get("doctor-register").URLPath()
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

	w.Write([]byte(u.String()))
}
