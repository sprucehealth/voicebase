package home

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
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
	dataAPI      api.DataAPI
}

type textDownloadLinkAPIRequest struct {
	Number string              `json:"number"`
	Code   string              `json:"code"`
	Params map[string][]string `json:"params"`
}

type textDownloadLinkAPIResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func newTextDownloadLinkAPIHandler(dataAPI api.DataAPI, smsAPI api.SMSAPI, fromNumber string, branchClient branch.Client, rateLimiter ratelimit.KeyedRateLimiter) httputil.ContextHandler {
	return httputil.SupportedMethods(&textDownloadLinkAPIHandler{
		smsAPI:       smsAPI,
		fromNumber:   fromNumber,
		branchClient: branchClient,
		rateLimiter:  rateLimiter,
		dataAPI:      dataAPI,
	}, httputil.Post)
}

func (h *textDownloadLinkAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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

	// Grab any parameters associated with our URL and throw them onto the branch link
	branchParams := map[string]interface{}{
		SourceKey: referralBranchSource,
	}

	if req.Code != "" {
		if _, err := h.dataAPI.LookupPromoCode(req.Code); err == nil {
			branchParams[PromoCodeKey] = req.Code
		}
	}

	for k, v := range req.Params {
		if k == PromoCodeKey || k == SourceKey {
			golog.Infof("Not attaching URL query param %s:%s to branch link as %s is a managed param.", k, v, k)
		} else {
			if len(v) == 1 {
				branchParams[k] = v[0]
			} else if len(v) > 1 {
				branchParams[k] = v
			}
		}
	}

	earl, err := h.branchClient.URL(branchParams)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	if err := h.smsAPI.Send(h.fromNumber, string(number), "To get the Spruce app follow this link "+earl); err != nil {
		// TODO: should unpack this error
		// "The 'To' number abc is not a valid phone number"
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, textDownloadLinkAPIResponse{Success: true})
}
