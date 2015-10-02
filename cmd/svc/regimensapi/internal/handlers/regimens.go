package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/svc/regimens"
	"golang.org/x/net/context"
)

type regimensHandler struct {
	svc       regimens.Service
	webDomain string
}

// NewRegimens returns a new regimens search and manipulation handler.
func NewRegimens(svc regimens.Service, webDomain string) httputil.ContextHandler {
	return httputil.SupportedMethods(&regimensHandler{
		svc:       svc,
		webDomain: webDomain,
	}, httputil.Post)
}

func (h *regimensHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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

func (h *regimensHandler) parsePOSTRequest(ctx context.Context, r *http.Request) (*regimenPOSTRequest, error) {
	rd := &regimenPOSTRequest{}
	// An empty body for a POST here is acceptable
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil && err != io.EOF {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	return rd, nil
}

func (h *regimensHandler) servePOST(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *regimenPOSTRequest) {
	var resourceID, authToken string
	if rd.Regimen == nil {
		iResourceID, err := idgen.NewID()
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
		resourceID = "r" + strconv.FormatInt(int64(iResourceID), 10)

		authToken, err = h.svc.AuthorizeResource(resourceID)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		// Write an empty regimen to the store to bootstrap it
		if err := h.svc.PutRegimen(resourceID, &regimens.Regimen{}, false); err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	} else {
		resourceID = rd.Regimen.ID
		authToken = r.Header.Get("token")
		if authToken == "" {
			apiservice.WriteAccessNotAllowedError(ctx, w, r)
			return
		}
		access, err := h.svc.CanAccessResource(rd.Regimen.ID, authToken)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		} else if !access {
			apiservice.WriteAccessNotAllowedError(ctx, w, r)
			return
		}

		_, published, err := h.svc.Regimen(resourceID)
		if api.IsErrNotFound(err) {
			apiservice.WriteResourceNotFoundError(ctx, err.Error(), w, r)
			return
		} else if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		} else if published {
			apiservice.WriteAccessNotAllowedError(ctx, w, r)
			return
		}

		if err := h.svc.PutRegimen(rd.Regimen.ID, rd.Regimen, rd.Publish); err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	}

	httputil.JSONResponse(w, http.StatusOK, &regimenPOSTResponse{
		ID:        resourceID,
		URL:       h.webDomain + "/" + resourceID,
		AuthToken: authToken,
	})
}

type regimenHandler struct {
	svc       regimens.Service
	webDomain string
}

// NewRegimen returns a new regimen search and manipulation handler.
func NewRegimen(svc regimens.Service, webDomain string) httputil.ContextHandler {
	return httputil.SupportedMethods(&regimenHandler{
		svc:       svc,
		webDomain: webDomain,
	}, httputil.Get, httputil.Put)
}

func (h *regimenHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	id, ok := mux.Vars(ctx)["id"]
	if !ok {
		apiservice.WriteResourceNotFoundError(ctx, "an id must be provided", w, r)
		return
	}
	regimen, published, err := h.svc.Regimen(id)
	if api.IsErrNotFound(err) {
		apiservice.WriteResourceNotFoundError(ctx, err.Error(), w, r)
		return
	} else if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	switch r.Method {
	case httputil.Get:
		h.serveGET(ctx, w, r, regimen)
	case httputil.Put:
		if published {
			apiservice.WriteAccessNotAllowedError(ctx, w, r)
			return
		}
		rd, err := h.parsePUTRequest(ctx, r)
		if err != nil {
			apiservice.WriteBadRequestError(ctx, err, w, r)
			return
		}
		h.servePUT(ctx, w, r, rd, id)
	}
}

func (h *regimenHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request, regimen *regimens.Regimen) {
	httputil.JSONResponse(w, http.StatusOK, regimen)
}

func (h *regimenHandler) parsePUTRequest(ctx context.Context, r *http.Request) (*regimenPUTRequest, error) {
	rd := &regimenPUTRequest{}
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.Regimen == nil {
		return nil, fmt.Errorf("regimen required")
	}
	return rd, nil
}

func (h *regimenHandler) servePUT(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *regimenPUTRequest, resourceID string) {
	authToken := r.Header.Get("token")
	if authToken == "" {
		apiservice.WriteAccessNotAllowedError(ctx, w, r)
		return
	}
	access, err := h.svc.CanAccessResource(resourceID, authToken)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	} else if !access {
		apiservice.WriteAccessNotAllowedError(ctx, w, r)
		return
	}

	if err := h.svc.PutRegimen(resourceID, rd.Regimen, rd.Publish); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &regimenPOSTResponse{
		ID:        resourceID,
		URL:       h.webDomain + "/" + resourceID,
		AuthToken: authToken,
	})
}
