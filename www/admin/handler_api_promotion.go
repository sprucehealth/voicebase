package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/www"
)

type promotionHandler struct {
	dataAPI api.DataAPI
}

// PromotionPUTRequest represents the data expected to be associated with a successful PUT request
type PromotionPUTRequest struct {
	Expires *int64 `json:"expires"`
}

// NewPromotionHandler returns an initialized instance of promotionHandler
func NewPromotionHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&promotionHandler{dataAPI: dataAPI}, httputil.Put)
}

func (h *promotionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}
	switch r.Method {
	case httputil.Put:
		req, err := h.parsePUTRequest(r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.servePUT(w, r, req, id)
	}
}

func (h *promotionHandler) parsePUTRequest(r *http.Request) (*PromotionPUTRequest, error) {
	rd := &PromotionPUTRequest{}
	if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	return rd, nil
}

func (h *promotionHandler) servePUT(w http.ResponseWriter, r *http.Request, req *PromotionPUTRequest, id int64) {
	var t *time.Time
	if req.Expires != nil {
		t = ptr.Time(time.Unix(*req.Expires, 0))
	}
	_, err := h.dataAPI.UpdatePromotion(&common.PromotionUpdate{
		CodeID:  id,
		Expires: t,
	})
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}
