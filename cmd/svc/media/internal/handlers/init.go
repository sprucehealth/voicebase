package handlers

import (
	"net/http"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/rs/cors"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/service"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/urlutil"
	"github.com/sprucehealth/backend/svc/auth"
	"golang.org/x/net/context"
)

const (
	authTokenCookieName = "at"
)

// InitRoutes registers the media service handlers on the provided mux
func InitRoutes(
	r *mux.Router,
	awsSession *session.Session,
	authClient auth.AuthClient,
	urlSigner *urlutil.Signer,
	dal dal.DAL,
	webDomain string,
	mediaStorageBucket string,
	mediaAPIDomain string,
	maxMemory int64,
) {
	svc := initService(awsSession, dal, mediaStorageBucket)
	corsOrigins := []string{"https://" + webDomain}
	mHandler := newAuthHandler(&mediaHandler{
		svc:            svc,
		mediaAPIDomain: mediaAPIDomain,
		maxMemory:      maxMemory,
	}, authClient)

	// Register the same handler on both paths
	r.Handle("/media", cors.New(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{httputil.Options, httputil.Post},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).ContextHandler(mHandler))
	r.Handle("/media/{id:"+media.IDRegexPattern+"}", cors.New(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{httputil.Get, httputil.Options},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).ContextHandler(mHandler))

	r.Handle("/media/{id:"+media.IDRegexPattern+"}/thumbnail", cors.New(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{httputil.Get, httputil.Options},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).ContextHandler(newAuthHandler(&thumbnailHandler{svc: svc}, authClient)))
}

func initService(awsSession *session.Session, dal dal.DAL, mediaStorageBucket string) service.Service {
	s3Store := storage.NewS3(awsSession, mediaStorageBucket, "media")
	s3CacheStore := storage.NewS3(awsSession, mediaStorageBucket, "media-cache")
	return service.New(
		dal,
		media.NewImageService(s3Store, s3CacheStore, 0, 0),
		media.NewAudioService(s3Store, s3CacheStore, 0),
		media.NewVideoService(s3Store, s3CacheStore, 0),
		media.NewBinaryService(s3Store, s3CacheStore, 0),
	)
}

type authHandler struct {
	auth auth.AuthClient
	h    httputil.ContextHandler
}

func newAuthHandler(h httputil.ContextHandler, auth auth.AuthClient) httputil.ContextHandler {
	return &authHandler{
		auth: auth,
		h:    h,
	}
}

func (a *authHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(authTokenCookieName)
	if err == http.ErrNoCookie {
		w.WriteHeader(http.StatusForbidden)
	} else if err != nil {
		golog.Warningf("Error getting cookie: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if c.Value == "" {
		golog.Warningf("Empty cookie value. Temporary log to weed out any issues with cookie handling between subdomains")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	res, err := a.auth.CheckAuthentication(ctx,
		&auth.CheckAuthenticationRequest{
			Token: c.Value,
		},
	)
	if err != nil {
		golog.Errorf("Failed to check auth token: %s", err)
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if !res.IsAuthenticated {
		golog.Warningf("User is unauthenticated. Temporary log to weed out any issues with cookie handling between subdomains")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	a.h.ServeHTTP(ctx, w, r)
}
