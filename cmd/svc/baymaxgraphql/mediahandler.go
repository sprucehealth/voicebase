package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	mediastore "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/media"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/svc/auth"
	"golang.org/x/net/context"
)

type mediaHandler struct {
	auth       auth.AuthClient
	media      *media.Service
	mediaStore *mediastore.Store
}

func NewMediaHandler(auth auth.AuthClient, media *media.Service, mediaStore *mediastore.Store) httputil.ContextHandler {
	return &mediaHandler{
		auth:       auth,
		media:      media,
		mediaStore: mediaStore,
	}
}

type errorMsg struct {
	Message string `json:"message"`
}

func (m *mediaHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var acc *account
	if c, err := r.Cookie(authTokenCookieName); err == nil && c.Value != "" {
		res, err := m.auth.CheckAuthentication(ctx,
			&auth.CheckAuthenticationRequest{Token: c.Value},
		)
		if err != nil {
			golog.Errorf("Failed to check auth token: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else if !res.IsAuthenticated {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		acc = &account{
			ID: res.Account.ID,
		}
	} else {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	ctx = ctxWithAccount(ctx, acc)

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

	var err error
	var expireTime uint64
	if expireTimeStr := r.FormValue("expires"); expireTimeStr != "" {
		expireTime, err = strconv.ParseUint(expireTimeStr, 10, 64)
		if err != nil {
			httputil.JSONResponse(w, http.StatusBadRequest, errorMsg{
				Message: fmt.Sprintf("Unable to parse expires %s: %s", expireTimeStr, err),
			})
			return
		}
	}
	var crop bool
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
	accountID, err := strconv.ParseUint(acc.ID[len("account:"):], 10, 64)
	if err != nil {
		golog.Errorf("Unable to parse accountID %s: %s", acc.ID, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !m.mediaStore.ValidateSignature(mediaID, mimetype, accountID, expireTime, width, height, crop, signature) {
		httputil.JSONResponse(w, http.StatusForbidden, errorMsg{
			Message: "Signature does not match",
		})
		return
	}
	// esnure that request is not expired
	if int64(expireTime) < time.Now().UTC().Unix() {
		httputil.JSONResponse(w, http.StatusForbidden, errorMsg{
			Message: "Expired request",
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
