package handlers

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/rs/cors"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/urlutil"
	"github.com/sprucehealth/backend/svc/auth"
)

// InitRoutes registers the media service handlers on the provided mux
func InitRoutes(
	r *mux.Router,
	awsSession *session.Session,
	authClient auth.AuthClient,
	urlSigner *urlutil.Signer,
	dal dal.DAL,
	webDomain string) {
	corsOrigins := []string{"https://" + webDomain}
	r.Handle("/media/{id:"+media.IDRegexPattern+"}", httputil.ToContextHandler(cors.New(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{httputil.Get, httputil.Options, httputil.Post},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).Handler(httputil.FromContextHandler(&mediaHandler{}))))
	r.Handle("/media/{id:"+media.IDRegexPattern+"}/thumbnail", httputil.ToContextHandler(cors.New(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{httputil.Get, httputil.Options},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).Handler(httputil.FromContextHandler(&thumbnailHandler{}))))
}
