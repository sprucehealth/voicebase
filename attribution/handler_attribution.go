package attribution

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/sprucehealth/backend/attribution/model"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ptr"
	"golang.org/x/net/context"
)

type attributionDAL interface {
	InsertAttributionData(attributionData *model.AttributionData) (int64, error)
}

type attributionHandler struct {
	attributionDAL attributionDAL
}

type attributionPOSTRequest struct {
	Data map[string]interface{} `json:"data"`
}

// NewAttributionHandler returns an initialized instance of attributionHandler
func NewAttributionHandler(attributionDAL attributionDAL) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			&attributionHandler{attributionDAL: attributionDAL}), httputil.Post)
}

func (h *attributionHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Post:
		rd, err := h.parsePOSTRequest(ctx, r)
		if err != nil {
			apiservice.WriteBadRequestError(ctx, err, w, r)
			return
		}
		h.servePOST(ctx, w, r, rd)
	}
}

func (h *attributionHandler) parsePOSTRequest(ctx context.Context, r *http.Request) (*attributionPOSTRequest, error) {
	rd := &attributionPOSTRequest{}
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil && err != io.EOF {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	} else if rd.Data == nil || err == io.EOF {
		return nil, errors.New("data required")
	}

	return rd, nil
}

func (h *attributionHandler) servePOST(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *attributionPOSTRequest) {
	ad := &model.AttributionData{Data: rd.Data}
	if account, ok := apiservice.CtxAccount(ctx); ok {
		ad.AccountID = ptr.Int64(account.ID)
	}

	if ad.AccountID == nil {
		deviceID, err := apiservice.GetDeviceIDFromHeader(r)
		if err == apiservice.ErrNoDeviceIDHeader {
			apiservice.WriteBadRequestError(ctx, err, w, r)
			return
		} else if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
		ad.DeviceID = ptr.String(deviceID)
	}
	_, err := h.attributionDAL.InsertAttributionData(ad)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	apiservice.WriteJSONSuccess(w)
}
