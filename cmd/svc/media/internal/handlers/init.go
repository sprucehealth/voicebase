package handlers

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/rs/cors"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/service"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/storage"
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
	webDomain string,
	mediaStorageBucket string,
	mediaAPIDomain string) {
	svc := initService(awsSession, dal, mediaStorageBucket)
	corsOrigins := []string{"https://" + webDomain}
	mHandler := &mediaHandler{svc: svc}

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
	}).ContextHandler(&thumbnailHandler{svc: svc}))
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
