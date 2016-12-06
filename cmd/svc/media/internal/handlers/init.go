package handlers

import (
	"net/http"

	"github.com/rs/cors"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/service"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/urlutil"
	"github.com/sprucehealth/backend/svc/auth"
)

const (
	authTokenCookieName = "at"
	idParamName         = "id"
)

// InitRoutes registers the media service handlers on the provided mux
func InitRoutes(
	r *mux.Router,
	svc service.Service,
	authClient auth.AuthClient,
	urlSigner *urlutil.Signer,
	webDomain string,
	mediaAPIDomain string,
	maxMemory int64,
) {
	corsOrigins := []string{"https://" + webDomain}
	if environment.IsProd() {
		corsOrigins = append(corsOrigins, "https://rc."+webDomain)
	}
	mHandler := &mediaHandler{
		svc:            svc,
		mediaAPIDomain: mediaAPIDomain,
		maxMemory:      maxMemory,
	}

	// Register the same handler on both paths
	r.Handle("/media", cors.New(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{httputil.Options, httputil.Post},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).Handler(authenticationRequired(mHandler, authClient, urlSigner, svc)))
	r.Handle("/media/{id:"+media.IDRegexPattern+"}", cors.New(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{httputil.Get, httputil.Options},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).Handler(authenticationRequired(authorizationRequired(mHandler, svc), authClient, urlSigner, svc)))
	r.Handle("/media/{id:"+media.IDRegexPattern+"}/thumbnail", cors.New(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{httputil.Get, httputil.Options},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).Handler(authenticationRequired(authorizationRequired(&thumbnailHandler{svc: svc}, svc), authClient, urlSigner, svc)))
	r.Handle(`/robots.txt`, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("User-agent: *\nDisallow: /\n"))
	}))
}
