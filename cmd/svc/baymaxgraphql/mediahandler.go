package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	mediasigner "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/media"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/svc/auth"
	"golang.org/x/net/context"
)

type mediaHandler struct {
	auth        auth.AuthClient
	media       *media.Service
	mediaSigner *mediasigner.Signer
}

func NewMediaHandler(auth auth.AuthClient, media *media.Service, mediaSigner *mediasigner.Signer) httputil.ContextHandler {
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
	var acc *models.Account

	if c, err := r.Cookie(authTokenCookieName); err == nil && c.Value != "" {
		res, err := m.auth.CheckAuthentication(ctx,
			&auth.CheckAuthenticationRequest{
				Token: c.Value,
			},
		)
		if err != nil {
			golog.Errorf("Failed to check auth token: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else if !res.IsAuthenticated {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		acc = &models.Account{
			ID: res.Account.ID,
		}
	} else {
		w.WriteHeader(http.StatusForbidden)
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
