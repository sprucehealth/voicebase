package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/svc/regimens"
	"github.com/sprucehealth/schema"
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
	}, httputil.Get, httputil.Post)
}

func (h *regimensHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		rd, err := h.parseGETRequest(ctx, r)
		if err != nil {
			apiservice.WriteBadRequestError(ctx, err, w, r)
			return
		}
		h.serveGET(ctx, w, r, rd)
	case httputil.Post:
		rd, err := h.parsePOSTRequest(ctx, r)
		if err != nil {
			apiservice.WriteBadRequestError(ctx, err, w, r)
			return
		}
		h.servePOST(ctx, w, r, rd)
	}
}

func (h *regimensHandler) parseGETRequest(ctx context.Context, r *http.Request) (*responses.RegimensGETRequest, error) {
	rd := &responses.RegimensGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	return rd, nil
}

func (h *regimensHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *responses.RegimensGETRequest) {
	tags := strings.Fields(rd.Query)
	for i, t := range tags {
		tags[i] = strings.ToLower(t)
	}

	// If there are no tags return an empty result
	if len(tags) == 0 {
		httputil.JSONResponse(w, http.StatusOK, &responses.RegimensGETResponse{})
		return
	}

	// Arbitrarily limit this till we understand the implications of tag filtering
	if len(tags) > 5 {
		apiservice.WriteBadRequestError(ctx, fmt.Errorf("A maximum number of 5 tags can be used in a single query. %d provided", len(tags)), w, r)
		return
	}

	regimens, err := h.svc.TagQuery(tags)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &responses.RegimensGETResponse{Regimens: regimens})
}

func (h *regimensHandler) parsePOSTRequest(ctx context.Context, r *http.Request) (*responses.RegimenPOSTRequest, error) {
	rd := &responses.RegimenPOSTRequest{}
	// An empty body for a POST here is acceptable
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil && err != io.EOF {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	return rd, nil
}

func (h *regimensHandler) servePOST(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *responses.RegimenPOSTRequest) {
	var resourceID, authToken string
	var regimen *regimens.Regimen
	if rd.Regimen == nil || rd.Regimen.ID == "" {
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

		// Write an empty regimen to the store to bootstrap it if one wasn't provided
		url := regimenURL(h.webDomain, resourceID)
		if rd.Regimen == nil {
			regimen = &regimens.Regimen{ID: resourceID, URL: url}
		} else {
			regimen = rd.Regimen
			regimen.ID = resourceID
			regimen.URL = url
		}
	} else if rd.Regimen.ID != "" {
		// If they provided a regimen ID, make sure they can access it and it isn't published
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
	}

	if regimen == nil || regimen.ID == "" {
		golog.Errorf("The regimen preparing to be written is null or lacks an identifier - %v", regimen)
		apiservice.WriteError(ctx, errors.New("The regimen preparing to be written is null or lacks an identifier"), w, r)
		return
	}

	// We can't associate a regimen with more than 24 tags
	if len(regimen.Tags) > 24 {
		apiservice.WriteBadRequestError(ctx, errors.New("A regimen can only be associated with a meximum of 24 tags"), w, r)
		return
	}

	if err := h.svc.PutRegimen(regimen.ID, regimen, rd.Publish); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &responses.RegimenPOSTResponse{
		ID:        resourceID,
		URL:       regimenURL(h.webDomain, resourceID),
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

	// If this is a mutating request or a GET on an unpublished record check auth
	// If there is no token in the header check the params
	authToken := r.Header.Get("token")
	if authToken == "" && r.Method == httputil.Get {
		rd, err := h.parseGETRequest(ctx, r)
		if err != nil {
			apiservice.WriteBadRequestError(ctx, err, w, r)
			return
		}
		authToken = rd.AuthToken
	}
	if r.Method == httputil.Put || (r.Method == httputil.Get && !published) {
		access, err := h.svc.CanAccessResource(id, authToken)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		} else if !access {
			apiservice.WriteAccessNotAllowedError(ctx, w, r)
			return
		}
	}

	switch r.Method {
	case httputil.Get:
		h.serveGET(ctx, w, r, regimen)
	case httputil.Put:
		// Do not allow published regimens to be mutated
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

func (h *regimenHandler) parseGETRequest(ctx context.Context, r *http.Request) (*responses.RegimenGETRequest, error) {
	rd := &responses.RegimenGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	return rd, nil
}

func (h *regimenHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request, regimen *regimens.Regimen) {
	httputil.JSONResponse(w, http.StatusOK, regimen)
}

func (h *regimenHandler) parsePUTRequest(ctx context.Context, r *http.Request) (*responses.RegimenPUTRequest, error) {
	rd := &responses.RegimenPUTRequest{}
	if err := json.NewDecoder(r.Body).Decode(rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.Regimen == nil {
		return nil, fmt.Errorf("regimen required")
	}
	return rd, nil
}

func (h *regimenHandler) servePUT(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *responses.RegimenPUTRequest, resourceID string) {
	authToken := r.Header.Get("token")
	for i, t := range rd.Regimen.Tags {
		rd.Regimen.Tags[i] = strings.ToLower(t)
	}
	rd.Regimen.ID = resourceID
	rd.Regimen.URL = regimenURL(h.webDomain, resourceID)

	// We can't associate a regimen with more than 24 tags
	if len(rd.Regimen.Tags) > 24 {
		apiservice.WriteBadRequestError(ctx, errors.New("A regimen can only be associated with a meximum of 24 tags"), w, r)
		return
	}

	if err := h.svc.PutRegimen(resourceID, rd.Regimen, rd.Publish); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &responses.RegimenPOSTResponse{
		ID:        resourceID,
		URL:       regimenURL(h.webDomain, resourceID),
		AuthToken: authToken,
	})
}

func regimenURL(webDomain, resourceID string) string {
	return strings.TrimRight(webDomain, "/") + "/" + resourceID
}
