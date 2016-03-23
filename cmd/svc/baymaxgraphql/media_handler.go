package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	imedia "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/media"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/schema"
	"golang.org/x/net/context"
)

type mediaHandler struct {
	auth        auth.AuthClient
	media       *media.Service
	mediaSigner *imedia.Signer
}

// NewMediaHandler returns an initialized instance of mediaHandler
func NewMediaHandler(auth auth.AuthClient, media *media.Service, mediaSigner *imedia.Signer) httputil.ContextHandler {
	return &mediaHandler{
		auth:        auth,
		media:       media,
		mediaSigner: mediaSigner,
	}
}

type errorMsg struct {
	Message string `json:"message"`
}

func (m *mediaHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		m.serveGET(ctx, w, r)
	case httputil.Post:
		m.servePOST(ctx, w, r)
	}
}

func (m *mediaHandler) checkAuth(ctx context.Context, r *http.Request) (*models.Account, int) {
	if c, err := r.Cookie(authTokenCookieName); err == nil && c.Value != "" {
		res, err := m.auth.CheckAuthentication(ctx,
			&auth.CheckAuthenticationRequest{
				Token: c.Value,
			},
		)
		if err != nil {
			golog.Errorf("Failed to check auth token: %s", err)
			return nil, http.StatusInternalServerError
		} else if !res.IsAuthenticated {
			return nil, http.StatusForbidden
		}
		return &models.Account{
			ID: res.Account.ID,
		}, 0
	}
	return nil, http.StatusForbidden
}

// mediaPOSTRequest represents the information associated with media posts
type mediaPOSTRequest struct {
	// TODO: For now just ask the client to send this information, but don't do anything with it
	OrganizationID string `schema:"organization_id"`
	ThreadID       string `schema:"thread_id"`
}

func parseMediaPOSTRequest(r *http.Request) (*mediaPOSTRequest, error) {
	rd := &mediaPOSTRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, err
	}

	if rd.OrganizationID == "" {
		return nil, errors.New("organization_id required")
	}

	return rd, nil
}

func (m *mediaHandler) servePOST(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, errCode := m.checkAuth(ctx, r)
	if errCode != 0 {
		w.WriteHeader(errCode)
		return
	}

	// TODO: Don't do anything for now with the information coming from the client. We just want to require it
	_, err := parseMediaPOSTRequest(r)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	file, _, err := r.FormFile("media")
	if err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Missing or invalid media in parameters: "+err.Error())
		return
	}
	defer file.Close()

	mediaID, err := media.NewID()
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	meta, err := m.media.PutReader(mediaID, file)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	res := &imedia.POSTResponse{
		MediaID: mediaID,
		URL:     meta.URL,
	}
	httputil.JSONResponse(w, http.StatusOK, res)
}

func (m *mediaHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	acc, errCode := m.checkAuth(ctx, r)
	if errCode != 0 {
		w.WriteHeader(errCode)
		return
	}

	ctx = gqlctx.WithAccount(ctx, acc)

	// get media related params
	mimetype := r.FormValue("mimetype")
	mediaID := r.FormValue("id")
	signature := r.FormValue("sig")

	if mimetype == "" {
		httputil.JSONResponse(w, http.StatusBadRequest, errorMsg{
			Message: "mimetype is required",
		})
		return
	}
	if mediaID == "" {
		httputil.JSONResponse(w, http.StatusBadRequest, errorMsg{
			Message: "id is required",
		})
		return
	}
	if signature == "" {
		httputil.JSONResponse(w, http.StatusBadRequest, errorMsg{
			Message: "signature is required",
		})
		return
	}

	var crop bool
	var err error
	if cropStr := r.FormValue("crop"); cropStr != "" {
		crop, err = strconv.ParseBool(cropStr)
		if err != nil {
			httputil.JSONResponse(w, http.StatusBadRequest, errorMsg{
				Message: fmt.Sprintf("Unable to parse crop %s: %s", cropStr, err),
			})
			return
		}
	}
	var width int
	if widthStr := r.FormValue("width"); widthStr != "" {
		width, err = strconv.Atoi(widthStr)
		if err != nil {
			httputil.JSONResponse(w, http.StatusBadRequest, errorMsg{
				Message: fmt.Sprintf("Unable to parse width %s: %s", widthStr, err),
			})
			return
		}
	}
	var height int
	if heightStr := r.FormValue("height"); heightStr != "" {
		height, err = strconv.Atoi(heightStr)
		if err != nil {
			httputil.JSONResponse(w, http.StatusBadRequest, errorMsg{
				Message: fmt.Sprintf("Unable to parse crop %s: %s", heightStr, err),
			})
			return
		}
	}
	// verify signature
	if !m.mediaSigner.ValidateSignature(mediaID, mimetype, acc.ID, width, height, crop, signature) {
		httputil.JSONResponse(w, http.StatusForbidden, errorMsg{
			Message: "Signature does not match",
		})
		return
	}

	// server media
	rc, meta, err := m.media.GetReader(mediaID, &media.Size{
		Width:        width,
		Height:       height,
		AllowScaleUp: false,
		Crop:         crop,
	})
	if err != nil {
		golog.Errorf("Unable to get media %s: %s", mediaID, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	copyWithHeaders(w, rc, meta.Size, meta.MimeType)
}

func copyWithHeaders(w http.ResponseWriter, r io.Reader, contentLen int, mimeType string) {
	w.Header().Set("Content-Type", mimeType)
	if contentLen > 0 {
		w.Header().Set("Content-Length", strconv.Itoa(contentLen))
	}
	// Note: We are currently not attaching a Last-Modified header on responses
	httputil.FarFutureCacheHeaders(w.Header(), time.Now())
	io.Copy(w, r)
}
