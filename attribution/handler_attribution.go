package attribution

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/attribution/model"
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
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.Data == nil {
		return nil, errors.New("data required")
	}
	return rd, nil
}

func (h *attributionHandler) servePOST(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *attributionPOSTRequest) {
	deviceID, err := apiservice.GetDeviceIDFromHeader(r)
	if err == apiservice.ErrNoDeviceIDHeader {
		apiservice.WriteBadRequestError(ctx, err, w, r)
		return
	} else if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	_, err = h.attributionDAL.InsertAttributionData(&model.AttributionData{
		DeviceID: ptr.String(deviceID),
		Data:     rd.Data,
	})
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	apiservice.WriteJSONSuccess(w)
}
