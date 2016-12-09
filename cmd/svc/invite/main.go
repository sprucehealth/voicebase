package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/stub"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/handlers"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/server"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/branch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/shttputil"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/settings"
)

var (
	flagBranchKey                 = flag.String("branch_key", "", "Branch API key")
	flagFromEmail                 = flag.String("from_email", "", "Email address from which to send invites")
	flagServiceNumber             = flag.String("service_phone_number", "", "TODO: This should be managed by the excomms service")
	flagListen                    = flag.String("listen_addr", ":5001", "`host:port` for grpc server")
	flagEventsTopic               = flag.String("events_topic", "", "SNS topic `ARN` for publishing events")
	flagKMSKeyARN                 = flag.String("kms_key_arn", "", "the `ARN` of the master key that should be used to encrypt outbound and decrypt inbound data")
	flagWebInviteURL              = flag.String("web_invite_url", "", "`URL` for the webapp invite page")
	flagColleagueInviteTemplateID = flag.String("colleague_invite_template_id", "", "`ID` of the colleague invite email template")
	flagPatientInviteTemplateID   = flag.String("patient_invite_template_id", "", "`ID` of the patient invite email template")

	// REST API
	flagHTTPListenAddr  = flag.String("http_listen_addr", ":8082", "host:port to listen on for http requests")
	flagInviteAPIDomain = flag.String("invite_api_domain", "", "Invite API `domain`")
	flagBehindHTTPProxy = flag.Bool("behind_proxy", false, "Flag to indicate when the service is behind a proxy")

	// For local development
	flagDynamoDBEndpoint = flag.String("dynamodb_endpoint", "", "DynamoDB endpoint `URL` (for local development)")

	// Services
	flagDirectoryAddr = flag.String("directory_addr", "_directory._tcp.service", "`host:port` of directory service")
	flagExcommsAddr   = flag.String("excomms_addr", "_excomms._tcp.service", "`host:port` of excomms service")
	flagSettingsAddr  = flag.String("settings_addr", "_settings._tcp.service", "host:port of settings service")
)

func main() {
	svc := boot.NewService("invite", nil)

	if *flagFromEmail == "" {
		golog.Fatalf("from_email required")
	}
	if *flagServiceNumber == "" {
		golog.Fatalf("service_phone_number required")
	}

	awsSession, err := svc.AWSSession()
	if err != nil {
		golog.Fatalf(err.Error())
	}
	var db *dynamodb.DynamoDB
	if *flagDynamoDBEndpoint != "" {
		dynamoConfig := &aws.Config{
			Region:     ptr.String("us-east-1"),
			DisableSSL: ptr.Bool(true),
			Endpoint:   flagDynamoDBEndpoint,
		}
		db = dynamodb.New(session.New(dynamoConfig))
	} else {
		db = dynamodb.New(awsSession)
	}

	if *flagDirectoryAddr == "" {
		golog.Fatalf("Directory service not configured")
	}
	conn, err := svc.DialGRPC(*flagDirectoryAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to directory service: %s", err)
	}
	defer conn.Close()
	directoryClient := directory.NewDirectoryClient(conn)

	var exCommsClient excomms.ExCommsClient
	if *flagExcommsAddr == "stub" {
		exCommsClient = stub.NewStubExcommsClient()
	} else {
		conn, err = svc.DialGRPC(*flagExcommsAddr)
		if err != nil {
			golog.Fatalf("Unable to connect to excomms service: %s", err)
		}
		exCommsClient = excomms.NewExCommsClient(conn)
	}

	conn, err = svc.DialGRPC(*flagSettingsAddr)
	if err != nil {
		golog.Fatalf("Unable to connect to settings service: %s", err)
	}
	settingsClient := settings.NewContextCacheClient(settings.NewSettingsClient(conn))

	branchCli := branch.NewClient(*flagBranchKey)

	eSNS, err := awsutil.NewEncryptedSNS(*flagKMSKeyARN, kms.New(awsSession), sns.New(awsSession))
	if err != nil {
		golog.Fatalf("Unable to initialize enrypted sns: %s", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = settings.RegisterConfigs(
		ctx,
		settingsClient,
		[]*settings.Config{
			invite.TwoFactorVerificationForSecureConversationConfig,
			invite.OrganizationCodeConfig,
			invite.PatientInviteChannelPreferenceConfig,
		})
	if err != nil {
		golog.Fatalf("Unable to register configs with the settings service: %s", err.Error())
	}
	cancel()

	dl := dal.New(db, environment.GetCurrent())
	srv := server.New(
		dl,
		nil,
		directoryClient,
		exCommsClient,
		settingsClient,
		eSNS,
		branchCli,
		*flagFromEmail,
		*flagServiceNumber,
		*flagEventsTopic,
		*flagWebInviteURL,
		*flagColleagueInviteTemplateID,
		*flagPatientInviteTemplateID)
	invite.InitMetrics(srv, svc.MetricsRegistry.Scope("server"))
	s := svc.GRPCServer()
	defer s.Stop()
	invite.RegisterInviteServer(s, srv)
	golog.Infof("Invite RPC listening on %s...", *flagListen)

	ln, err := net.Listen("tcp", *flagListen)
	if err != nil {
		golog.Fatalf("failed to listen on %s: %v", *flagListen, err)
	}
	defer ln.Close()
	go func() {
		if err := s.Serve(ln); err != nil {
			golog.Errorf(err.Error())
		}
	}()

	router := mux.NewRouter()
	handlers.InitRoutes(router, dl)
	h := httputil.LoggingHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Redirect HTTP to HTTPS
		if *flagBehindHTTPProxy {
			if r.Header.Get("X-Forwarded-Proto") == "http" {
				u := r.URL
				u.Host = r.Host
				u.Scheme = "https"
				http.Redirect(w, r, u.String(), http.StatusPermanentRedirect)
				return
			}
		}
		router.ServeHTTP(w, r)
	}), "media", *flagBehindHTTPProxy, nil)

	golog.Infof("Invite HTTP Listening on %s...", *flagHTTPListenAddr)
	httpSrv := &http.Server{
		Addr:           *flagHTTPListenAddr,
		Handler:        shttputil.CompressResponse(h, httputil.CompressResponse),
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		httpSrv.ListenAndServe()
	}()

	boot.WaitForTermination()
	svc.Shutdown()
}
