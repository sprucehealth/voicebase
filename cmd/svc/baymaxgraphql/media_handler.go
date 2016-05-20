package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	imedia "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/media"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/trace/tracectx"
	"github.com/sprucehealth/backend/svc/auth"
	"golang.org/x/net/context"
)

type mediaHandler struct {
	auth        auth.AuthClient
	media       *media.Service
	mediaSigner *media.Signer
}

// NewMediaHandler returns an initialized instance of mediaHandler
func NewMediaHandler(auth auth.AuthClient, media *media.Service, mediaSigner *media.Signer) httputil.ContextHandler {
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
	case httputil.Head:
		m.serveHEAD(ctx, w, r)
	case httputil.Get:
		m.serveGET(ctx, w, r)
	case httputil.Post:
		m.servePOST(ctx, w, r)
	}
}

func (m *mediaHandler) checkAuth(ctx context.Context, r *http.Request) (*auth.Account, int) {
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
		return res.Account, 0
	}
	return nil, http.StatusForbidden
}

func (m *mediaHandler) servePOST(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, errCode := m.checkAuth(ctx, r)
	if errCode != 0 {
		w.WriteHeader(errCode)
		return
	}

	file, _, err := r.FormFile("media")
	if err != nil {
		http.Error(w, "Missing or invalid media in parameters: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	mediaID, err := media.NewID()
	if err != nil {
		httpInternalError(ctx, w, err)
		return
	}

	if _, err = m.media.PutReader(mediaID, file); err != nil {
		httpInternalError(ctx, w, err)
		return
	}

	res := &imedia.POSTResponse{
		MediaID: mediaID,
	}
	httputil.JSONResponse(w, http.StatusOK, res)
}

func (m *mediaHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	m.serve(ctx, w, r, false)
}

func (m *mediaHandler) serveHEAD(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	m.serve(ctx, w, r, true)
}

func (m *mediaHandler) serve(ctx context.Context, w http.ResponseWriter, r *http.Request, headersOnly bool) {
	// get media related params
	mimetype := r.FormValue("mimetype")
	mediaID := r.FormValue("id")
	signature := r.FormValue("sig")

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
	expires := time.Time{}
	expiresStr := r.FormValue("expires")
	if expiresStr != "" {
		expiresUnix, err := strconv.ParseInt(expiresStr, 10, 64)
		if err != nil {
			httputil.JSONResponse(w, http.StatusBadRequest, errorMsg{
				Message: fmt.Sprintf("Unable to parse expiration epoch %s: %s", expiresStr, err),
			})
			return
		}
		expires = time.Unix(expiresUnix, 0)
	}

	// First check if this is a valid unauthenticated request
	if !m.mediaSigner.ValidateSignature(mediaID, mimetype, "", width, height, crop, expires, signature) {
		// If not a valid unauthenticated signature, check the authenticated sig version
		acc, errCode := m.checkAuth(ctx, r)
		if errCode != 0 {
			w.WriteHeader(errCode)
			return
		}
		ctx = gqlctx.WithAccount(ctx, acc)
		if !m.mediaSigner.ValidateSignature(mediaID, mimetype, acc.ID, width, height, crop, expires, signature) {
			httputil.JSONResponse(w, http.StatusForbidden, errorMsg{
				Message: "Signature does not match",
			})
			return
		}
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
	copyWith(w, rc, meta.Size, meta.MimeType, headersOnly)
}

func copyWith(w http.ResponseWriter, r io.Reader, contentLen int, mimeType string, headersOnly bool) {
	w.Header().Set("Content-Type", mimeType)
	if contentLen > 0 {
		w.Header().Set("Content-Length", strconv.Itoa(contentLen))
	}

	if !headersOnly {
		// Note: We are currently not attaching a Last-Modified header on responses
		httputil.FarFutureCacheHeaders(w.Header(), time.Now())
		io.Copy(w, r)
	}
}

func httpInternalError(ctx context.Context, w http.ResponseWriter, err error) {
	requestID := tracectx.RequestID(ctx)
	golog.Context(
		"RequestID", requestID,
	).LogDepthf(1, golog.ERR, err.Error())
	if environment.IsDev() {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
	}
}
