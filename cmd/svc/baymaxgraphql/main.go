package main

import (
	"flag"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/rs/cors"
	"github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/boot"
	baymaxgraphqlsettings "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/settings"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/stub"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/shttputil"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/directory/cache"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	flagListenAddr          = flag.String("listen_addr", "127.0.0.1:8080", "host:port to listen on")
	flagResourcePath        = flag.String("resource_path", "", "Path to resources (defaults to use GOPATH)")
	flagAPIDomain           = flag.String("api_domain", "", "API `domain`")
	flagMediaAPIDomain      = flag.String("media_api_domain", "", "Media API `domain`")
	flagWebDomain           = flag.String("web_domain", "", "Web `domain`")
	flagStorageBucket       = flag.String("storage_bucket", "", "storage bucket for media")
	flagEmailDomain         = flag.String("email_domain", "", "domain to use for email address provisioning")
	flagServiceNumber       = flag.String("service_phone_number", "", "TODO: This should be managed by the excomms service")
	flagSpruceOrgID         = flag.String("spruce_org_id", "", "`ID` for the Spruce support organization")
	flagStaticURLPrefix     = flag.String("static_url_prefix", "", "URL prefix of static assets")
	flagSegmentIOKey        = flag.String("segmentio_key", "", "Segment IO API `key`")
	flagBehindProxy         = flag.Bool("behind_proxy", false, "Flag to indicate when the service is behind a proxy")
	flagLayoutStoreS3Prefix = flag.String("s3_prefix_saml", "", "S3 Prefix for layouts")

	// Services
	flagAuthAddr      = flag.String("auth_addr", "", "host:port of auth service")
	flagDirectoryAddr = flag.String("directory_addr", "", "host:port of directory service")
	flagExCommsAddr   = flag.String("excomms_addr", "", "host:port of excomms service")
	flagInviteAddr    = flag.String("invite_addr", "", "host:port of invites service")
	flagSettingsAddr  = flag.String("settings_addr", "", "host:port of settings service")
	flagThreadingAddr = flag.String("threading_addr", "", "host:port of threading service")
	flagLayoutAddr    = flag.String("layout_addr", "", "host:port of layout service")
	flagCareAddr      = flag.String("care_addr", "", "host:port of care service")
	flagMediaAddr     = flag.String("media_addr", "", "host:port of media service")

	// Messages
	flagSQSDeviceRegistrationURL   = flag.String("sqs_device_registration_url", "", "the sqs url for device registration messages")
	flagSQSDeviceDeregistrationURL = flag.String("sqs_device_deregistration_url", "", "the sqs url for device deregistration messages")
	flagSQSNotificationURL         = flag.String("sqs_notification_url", "", "the sqs url for notification queueing")
	flagSupportMessageTopicARN     = flag.String("sns_support_message_arn", "", "sns topic on which to post org created events for sending support message")

	// Encryption
	flagKMSKeyARN = flag.String("kms_key_arn", "", "the arn of the master key that should be used to encrypt outbound and decrypt inbound data")
)

func main() {
	svc := boot.NewService("baymaxgraphql")

	if *flagKMSKeyARN == "" {
		golog.Fatalf("-kms_key_arn flag is required")
	}

	if *flagSpruceOrgID == "" {
		golog.Fatalf("-spruce_org_id flag is required")
	}
	if *flagStaticURLPrefix == "" {
		golog.Fatalf("-static_url_prefix flag required")
	}
	*flagStaticURLPrefix = strings.Replace(*flagStaticURLPrefix, "{BuildNumber}", boot.BuildNumber, -1)
	if !strings.HasSuffix(*flagStaticURLPrefix, "/") {
		*flagStaticURLPrefix += "/"
	}

	if *flagServiceNumber == "" {
		golog.Fatalf("A service phone number must be provided")
	}
	pn, err := phone.ParseNumber(*flagServiceNumber)
	if err != nil {
		golog.Fatalf("Failed to parse service phone number: %s", err)
	}
	if *flagAuthAddr == "" {
		golog.Fatalf("Auth service not configured")
	}
	conn, err := grpc.Dial(*flagAuthAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to auth service: %s", err)
	}
	authClient := auth.NewAuthClient(conn)

	if *flagDirectoryAddr == "" {
		golog.Fatalf("Directory service not configured")
	}
	conn, err = grpc.Dial(*flagDirectoryAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to directory service: %s", err)
	}
	directoryClient := cache.NewCachedClient(directory.NewDirectoryClient(conn), svc.MetricsRegistry.Scope("CachedDirectoryClient"))

	if *flagThreadingAddr == "" {
		golog.Fatalf("Threading service not configured")
	}
	conn, err = grpc.Dial(*flagThreadingAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to threading service: %s", err)
	}
	threadingClient := threading.NewThreadsClient(conn)

	if *flagSettingsAddr == "" {
		golog.Fatalf("Settings service not configured")
	}
	conn, err = grpc.Dial(*flagSettingsAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to settings service: %s", err)
	}
	settingsClient := settings.NewContextCacheClient(settings.NewSettingsClient(conn))

	if *flagLayoutAddr == "" {
		golog.Fatalf("Layout service not configured")
	}
	conn, err = grpc.Dial(*flagLayoutAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to Layout service: %s", err)
	}
	layoutClient := layout.NewLayoutClient(conn)

	if *flagCareAddr == "" {
		golog.Fatalf("Care service not configured")
	}
	conn, err = grpc.Dial(*flagCareAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to care service: %s", err)
	}
	careClient := care.NewCareClient(conn)

	if *flagMediaAddr == "" {
		golog.Fatalf("Media service not configured")
	}
	conn, err = grpc.Dial(*flagMediaAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to media service: %s", err)
	}
	mediaClient := media.NewMediaClient(conn)

	// enable for non-prod
	baymaxgraphqlsettings.VisitAttachmentsConfig.GetBoolean().Default.Value = !environment.IsProd()
	baymaxgraphqlsettings.ShakeToMarkThreadsAsReadConfig.GetBoolean().Default.Value = !environment.IsProd()
	baymaxgraphqlsettings.CarePlansConfig.GetBoolean().Default.Value = !environment.IsProd()
	baymaxgraphqlsettings.FilteredTabsInInboxConfig.GetBoolean().Default.Value = !environment.IsProd()
	baymaxgraphqlsettings.VideoCallingConfig.GetBoolean().Default.Value = !environment.IsProd()

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = settings.RegisterConfigs(
		ctx,
		settingsClient,
		[]*settings.Config{
			baymaxgraphqlsettings.TeamConversationsConfig,
			baymaxgraphqlsettings.SecureThreadsConfig,
			baymaxgraphqlsettings.VisitAttachmentsConfig,
			baymaxgraphqlsettings.ShakeToMarkThreadsAsReadConfig,
			baymaxgraphqlsettings.CarePlansConfig,
			baymaxgraphqlsettings.FilteredTabsInInboxConfig,
			baymaxgraphqlsettings.VideoCallingConfig,
		})
	if err != nil {
		golog.Fatalf("Unable to register configs with the settings service: %s", err.Error())
	}

	if *flagExCommsAddr == "" {
		golog.Fatalf("ExComm service not configured")
	}
	var exCommsClient excomms.ExCommsClient
	if *flagExCommsAddr == "stub" {
		exCommsClient = stub.NewStubExcommsClient()
	} else {
		conn, err = grpc.Dial(*flagExCommsAddr, grpc.WithInsecure())
		if err != nil {
			golog.Fatalf("Unable to connect to excomms service: %s", err)
		}
		exCommsClient = excomms.NewExCommsClient(conn)
	}

	if *flagInviteAddr == "" {
		golog.Fatalf("Invite service not configured")
	}
	conn, err = grpc.Dial(*flagInviteAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to invite service: %s", err)
	}
	inviteClient := invite.NewInviteClient(conn)

	if *flagSQSDeviceRegistrationURL == "" {
		golog.Fatalf("Notification service device registration not configured")
	}
	if *flagSQSDeviceDeregistrationURL == "" {
		golog.Fatalf("Notification service device deregistration not configured")
	}
	if *flagSQSNotificationURL == "" {
		golog.Fatalf("Notification service notification queue not configured")
	}

	awsSession, err := svc.AWSSession()
	if err != nil {
		golog.Fatalf("Failed to create AWS session: %s", err)
	}

	eSQS, err := awsutil.NewEncryptedSQS(*flagKMSKeyARN, kms.New(awsSession), sqs.New(awsSession))
	if err != nil {
		golog.Fatalf("Unable to initialize Encrypted SQS: %s", err)
	}
	eSNS, err := awsutil.NewEncryptedSNS(*flagKMSKeyARN, kms.New(awsSession), sns.New(awsSession))
	if err != nil {
		golog.Fatalf("Unable to initialize enrypted sns: %s", err.Error())
		return
	}
	notificationClient := notification.NewClient(
		eSQS,
		&notification.ClientConfig{
			SQSDeviceRegistrationURL:   *flagSQSDeviceRegistrationURL,
			SQSNotificationURL:         *flagSQSNotificationURL,
			SQSDeviceDeregistrationURL: *flagSQSDeviceDeregistrationURL,
		})
	if *flagAPIDomain == "" {
		golog.Fatalf("API Domain not specified")
	}
	if *flagWebDomain == "" {
		golog.Fatalf("Web domain not specified")
	}
	if *flagStorageBucket == "" {
		golog.Fatalf("Storage bucket not specified")
	}
	if *flagEmailDomain == "" {
		golog.Fatalf("Email domain not specified")
	}
	if *flagSupportMessageTopicARN == "" {
		golog.Fatalf("SNS topic for posting requests to send support message")
	}
	if *flagLayoutStoreS3Prefix == "" {
		golog.Fatalf("S3 Prefix for SAML documents not specified")
	}

	corsOrigins := []string{"https://" + *flagWebDomain}

	var segmentClient *analytics.Client
	if *flagSegmentIOKey != "" {
		segmentClient = analytics.New(*flagSegmentIOKey)
		defer segmentClient.Close()
	}
	if *flagMediaAPIDomain == "" {
		golog.Fatalf("Media API Domain required")
	}

	r := mux.NewRouter()
	gqlHandler := NewGraphQL(
		authClient,
		directoryClient,
		threadingClient,
		exCommsClient,
		notificationClient,
		settingsClient,
		inviteClient,
		layoutClient,
		careClient,
		mediaClient,
		layout.NewStore(storage.NewS3(awsSession, *flagStorageBucket, *flagLayoutStoreS3Prefix)),
		*flagEmailDomain,
		*flagWebDomain,
		*flagMediaAPIDomain,
		pn,
		*flagSpruceOrgID,
		*flagStaticURLPrefix,
		segmentClient,
		eSNS,
		*flagSupportMessageTopicARN,
		svc.MetricsRegistry.Scope("handler"))
	r.Handle("/graphql", httputil.ToContextHandler(cors.New(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{httputil.Get, httputil.Options, httputil.Post},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).Handler(httputil.FromContextHandler(gqlHandler))))

	mediaHandler := NewMediaHandler(*flagMediaAPIDomain)
	r.Handle("/media", httputil.ToContextHandler(cors.New(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{httputil.Get, httputil.Options, httputil.Post},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).Handler(httputil.FromContextHandler(mediaHandler))))

	if *flagResourcePath == "" {
		if p := os.Getenv("GOPATH"); p != "" {
			*flagResourcePath = path.Join(p, "src/github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/resources")
		}
	}
	if !environment.IsProd() {
		if *flagResourcePath != "" {
			r.PathPrefix("/graphiql/").Handler(httputil.FileServer(http.Dir(*flagResourcePath)))
		}
		r.Handle("/schema", newSchemaHandler())
	}

	h := shttputil.CompressResponse(r, httputil.CompressResponse)
	h = httputil.LoggingHandler(h, "baymaxgraphql", *flagBehindProxy, nil)
	h = httputil.RequestIDHandler(h)

	golog.Infof("Listening on %s", *flagListenAddr)

	server := &http.Server{
		Addr:           *flagListenAddr,
		Handler:        httputil.FromContextHandler(h),
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		server.ListenAndServe()
	}()

	boot.WaitForTermination()
}
