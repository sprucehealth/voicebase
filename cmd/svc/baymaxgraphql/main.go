package main

import (
	"context"
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
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
)

var (
	flagListenAddr               = flag.String("listen_addr", "127.0.0.1:8080", "host:port to listen on")
	flagLetsEncrypt              = flag.Bool("letsencrypt", false, "Enable Let's Encrypt certificates")
	flagCertCacheURL             = flag.String("cert_cache_url", "", "URL path where to store cert cache (e.g. s3://bucket/path/)")
	flagProxyProtocol            = flag.Bool("proxy_protocol", false, "If behind a TCP proxy and proxy protocol wrapping is enabled")
	flagResourcePath             = flag.String("resource_path", "", "Path to resources (defaults to use GOPATH)")
	flagAPIDomain                = flag.String("api_domain", "", "API `domain`")
	flagMediaAPIDomain           = flag.String("media_api_domain", "", "Media API `domain`")
	flagInviteAPIDomain          = flag.String("invite_api_domain", "", "Invite API `domain`")
	flagWebDomain                = flag.String("web_domain", "", "Web `domain`")
	flagStorageBucket            = flag.String("storage_bucket", "", "storage bucket for media")
	flagEmailDomain              = flag.String("email_domain", "", "domain to use for email address provisioning")
	flagServiceNumber            = flag.String("service_phone_number", "", "TODO: This should be managed by the excomms service")
	flagSpruceOrgID              = flag.String("spruce_org_id", "", "`ID` for the Spruce support organization")
	flagStaticURLPrefix          = flag.String("static_url_prefix", "", "URL prefix of static assets")
	flagBehindProxy              = flag.Bool("behind_proxy", false, "Flag to indicate when the service is behind a proxy")
	flagLayoutStoreS3Prefix      = flag.String("s3_prefix_saml", "", "S3 Prefix for layouts")
	flagTransactionalEmailSender = flag.String("transactional_email_sender", "", "Email address for the transactional email sender")

	// Email tempaltes
	flagPasswordResetTemplateID     = flag.String("password_reset_template_id", "", "ID of password reset template")
	flagEmailVerificationTemplateID = flag.String("email_verification_template_id", "", "ID of email verification template")

	// Services
	flagAuthAddr      = flag.String("auth_addr", "_auth._tcp.service", "host:port of auth service")
	flagDirectoryAddr = flag.String("directory_addr", "_directory._tcp.service", "host:port of directory service")
	flagExCommsAddr   = flag.String("excomms_addr", "_excomms._tcp.service", "host:port of excomms service")
	flagInviteAddr    = flag.String("invite_addr", "_invite._tcp.service", "host:port of invites service")
	flagSettingsAddr  = flag.String("settings_addr", "_settings._tcp.service", "host:port of settings service")
	flagThreadingAddr = flag.String("threading_addr", "_threading._tcp.service", "host:port of threading service")
	flagLayoutAddr    = flag.String("layout_addr", "_layout._tcp.service", "host:port of layout service")
	flagCareAddr      = flag.String("care_addr", "_care._tcp.service", "host:port of care service")
	flagMediaAddr     = flag.String("media_addr", "_media._tcp.service", "host:port of media service")
	flagPaymentsAddr  = flag.String("payments_addr", "_payments._tcp.service", "host:port of payments service")

	// Messages
	flagSQSDeviceRegistrationURL   = flag.String("sqs_device_registration_url", "", "the sqs url for device registration messages")
	flagSQSDeviceDeregistrationURL = flag.String("sqs_device_deregistration_url", "", "the sqs url for device deregistration messages")
	flagSQSNotificationURL         = flag.String("sqs_notification_url", "", "the sqs url for notification queueing")
	flagSupportMessageTopicARN     = flag.String("sns_support_message_arn", "", "sns topic on which to post org created events for sending support message")

	// Encryption
	flagKMSKeyARN = flag.String("kms_key_arn", "", "the arn of the master key that should be used to encrypt outbound and decrypt inbound data")
)

func main() {
	var authClient auth.AuthClient
	svc := boot.NewService("baymaxgraphql", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if authClient == nil {
			w.WriteHeader(http.StatusOK)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		_, err := authClient.CheckAuthentication(ctx,
			&auth.CheckAuthenticationRequest{
				Token: "dummy",
			},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

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

	conn, err := boot.DialGRPC("baymaxgraphql", *flagAuthAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to auth service: %s", err)
	}
	authClient = auth.NewAuthClient(conn)

	conn, err = boot.DialGRPC("baymaxgraphql", *flagDirectoryAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to directory service: %s", err)
	}
	directoryClient := cache.NewCachedClient(directory.NewDirectoryClient(conn), svc.MetricsRegistry.Scope("CachedDirectoryClient"))

	conn, err = boot.DialGRPC("baymaxgraphql", *flagThreadingAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to threading service: %s", err)
	}
	threadingClient := threading.NewThreadsClient(conn)

	conn, err = boot.DialGRPC("baymaxgraphql", *flagSettingsAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to settings service: %s", err)
	}
	settingsClient := settings.NewContextCacheClient(settings.NewSettingsClient(conn))

	conn, err = boot.DialGRPC("baymaxgraphql", *flagLayoutAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to Layout service: %s", err)
	}
	layoutClient := layout.NewLayoutClient(conn)

	conn, err = boot.DialGRPC("baymaxgraphql", *flagCareAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to care service: %s", err)
	}
	careClient := care.NewCareClient(conn)

	conn, err = boot.DialGRPC("baymaxgraphql", *flagMediaAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to media service: %s", err)
	}
	mediaClient := media.NewMediaClient(conn)

	conn, err = boot.DialGRPC("baymaxgraphql", *flagPaymentsAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to payments service: %s", err)
	}
	paymentsClient := payments.NewPaymentsClient(conn)

	// enable for non-prod
	baymaxgraphqlsettings.VisitAttachmentsConfig.GetBoolean().Default.Value = !environment.IsProd()
	baymaxgraphqlsettings.CarePlansConfig.GetBoolean().Default.Value = !environment.IsProd()
	baymaxgraphqlsettings.VideoCallingConfig.GetBoolean().Default.Value = !environment.IsProd()
	baymaxgraphqlsettings.PaymentsConfig.GetBoolean().Default.Value = !environment.IsProd()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
			baymaxgraphqlsettings.PaymentsConfig,
			invite.OrganizationCodeConfig,
		})
	if err != nil {
		golog.Fatalf("Unable to register configs with the settings service: %s", err.Error())
	}
	cancel()

	var exCommsClient excomms.ExCommsClient
	if *flagExCommsAddr == "stub" {
		exCommsClient = stub.NewStubExcommsClient()
	} else {
		conn, err = boot.DialGRPC("baymaxgraphql", *flagExCommsAddr)
		if err != nil {
			golog.Fatalf("Unable to connect to excomms service: %s", err)
		}
		exCommsClient = excomms.NewExCommsClient(conn)
	}

	conn, err = boot.DialGRPC("baymaxgraphql", *flagInviteAddr)
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
	if environment.IsProd() {
		corsOrigins = append(corsOrigins, "https://rc."+*flagWebDomain)
	}

	if *flagMediaAPIDomain == "" {
		golog.Fatalf("Media API Domain required")
	}
	if *flagInviteAPIDomain == "" {
		golog.Fatalf("Invite API Domain required")
	}
	if *flagTransactionalEmailSender == "" {
		golog.Fatalf("Transactioanl email sender required")
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
		paymentsClient,
		layout.NewStore(storage.NewS3(awsSession, *flagStorageBucket, *flagLayoutStoreS3Prefix)),
		*flagEmailDomain,
		*flagWebDomain,
		*flagMediaAPIDomain,
		*flagInviteAPIDomain,
		pn,
		*flagSpruceOrgID,
		*flagStaticURLPrefix,
		eSNS,
		*flagSupportMessageTopicARN,
		emailTemplateIDs{
			passwordReset:     *flagPasswordResetTemplateID,
			emailVerification: *flagEmailVerificationTemplateID,
		},
		svc.MetricsRegistry.Scope("handler"),
		*flagTransactionalEmailSender,
	)
	r.Handle("/graphql", cors.New(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{httputil.Get, httputil.Options, httputil.Post},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).Handler(gqlHandler))

	mediaHandler := NewMediaHandler(*flagMediaAPIDomain)
	r.Handle("/media", cors.New(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{httputil.Get, httputil.Options, httputil.Post},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).Handler(mediaHandler))

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

	if !*flagLetsEncrypt {
		go func() {
			server := &http.Server{
				Addr:           *flagListenAddr,
				Handler:        h,
				MaxHeaderBytes: 1 << 20,
			}
			server.ListenAndServe()
		}()
	} else {
		server := &http.Server{
			Addr:      *flagListenAddr,
			TLSConfig: boot.TLSConfig(),
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r.Header.Set("X-Forwarded-Proto", "https")
				h.ServeHTTP(w, r)
			}),
			MaxHeaderBytes: 1 << 20,
		}
		certStore, err := svc.StoreFromURL(*flagCertCacheURL)
		if err != nil {
			golog.Fatalf("Failed to generate cert cache store from url '%s': %s", *flagCertCacheURL, err)
		}
		server.TLSConfig.GetCertificate = boot.LetsEncryptCertManager(certStore.(storage.DeterministicStore), []string{*flagAPIDomain})
		go func() {
			if err := boot.HTTPSListenAndServe(server, *flagProxyProtocol); err != nil {
				golog.Fatalf(err.Error())
			}
		}()
	}

	boot.WaitForTermination()
}
