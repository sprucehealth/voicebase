package home

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/branch"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ratelimit"
	"github.com/sprucehealth/backend/www"
)

const referralBranchSource = "website"

type textDownloadLinkAPIHandler struct {
	smsAPI       api.SMSAPI
	fromNumber   string
	branchClient branch.Client
	rateLimiter  ratelimit.KeyedRateLimiter
}

type textDownloadLinkAPIRequest struct {
	Number string `json:"number"`
	Code   string `json:"code"`
}

type textDownloadLinkAPIResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func NewTextDownloadLinkAPIHandler(smsAPI api.SMSAPI, fromNumber string, branchClient branch.Client, rateLimiter ratelimit.KeyedRateLimiter) http.Handler {
	return httputil.SupportedMethods(&textDownloadLinkAPIHandler{
		smsAPI:       smsAPI,
		fromNumber:   fromNumber,
		branchClient: branchClient,
		rateLimiter:  rateLimiter,
	}, httputil.Post)
}

func (h *textDownloadLinkAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Rate limit by remote IP address
	if ok, err := h.rateLimiter.Check("ref:"+r.RemoteAddr, 1); err != nil {
		golog.Errorf("Rate limit check failed: %s", err.Error())
	} else if !ok {
		www.APIBadRequestError(w, r, "invalid request")
		return
	}

	var req textDownloadLinkAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		www.APIBadRequestError(w, r, "could not decode request body")
		return
	}

	// Not a user error
	if req.Code == "" {
		www.APIBadRequestError(w, r, "missing code")
		return
	}

	number, err := common.ParsePhone(req.Number)
	if err != nil {
		httputil.JSONResponse(w, http.StatusOK, textDownloadLinkAPIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Rate limit against a single phone number
	if ok, err := h.rateLimiter.Check("ref:"+string(number), 1); err != nil {
		golog.Errorf("Rate limit check failed: %s", err.Error())
	} else if !ok {
		www.APIBadRequestError(w, r, "invalid request")
		return
	}

	earl, err := h.branchClient.URL(map[string]interface{}{
		"promo_code": req.Code,
		"source":     referralBranchSource,
	})
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	if err := h.smsAPI.Send(h.fromNumber, string(number), "To get the Spruce iOS app follow this link "+earl); err != nil {
		// TODO: should unpack this error
		// "The 'To' number abc is not a valid phone number"
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, textDownloadLinkAPIResponse{Success: true})
}
