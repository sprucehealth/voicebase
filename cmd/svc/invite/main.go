package main

import (
	"flag"
	"log"
	"net"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/sendgrid/sendgrid-go"
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
	"google.golang.org/grpc"
)

var (
	flagBranchKey     = flag.String("branch_key", "", "Branch API key")
	flagDirectoryAddr = flag.String("directory_addr", "", "`host:port` of directory service")
	flagExcommsAddr   = flag.String("excomms_addr", "", "`host:port` of excomms service")
	flagFromEmail     = flag.String("from_email", "", "Email address from which to send invites")
	flagServiceNumber = flag.String("service_phone_number", "", "TODO: This should be managed by the excomms service")
	flagListen        = flag.String("listen_addr", ":5001", "`host:port` for grpc server")
	flagSendGridKey   = flag.String("sendgrid_key", "", "SendGrid API `key`")
	flagEventsTopic   = flag.String("events_topic", "", "SNS topic ARN for publishing events")
	flagKMSKeyARN     = flag.String("kms_key_arn", "", "the arn of the master key that should be used to encrypt outbound and decrypt inbound data")
	flagWebInviteURL  = flag.String("web_invite_url", "", "URL for the webapp invite page")

	// REST API
	flagHTTPListenAddr  = flag.String("http_listen_addr", ":8082", "host:port to listen on for http requests")
	flagInviteAPIDomain = flag.String("invite_api_domain", "", "Invite API `domain`")
	flagBehindProxy     = flag.Bool("behind_proxy", false, "Flag to indicate when the service is behind a proxy")

	// For local development
	flagDynamoDBEndpoint = flag.String("dynamodb_endpoint", "", "DynamoDB endpoint `URL` (for local development)")
)

func main() {
	svc := boot.NewService("invite")

	if *flagFromEmail == "" {
		golog.Fatalf("from_email required")
	}
	if *flagSendGridKey == "" {
		golog.Fatalf("sendgrid_key required")
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
	conn, err := grpc.Dial(*flagDirectoryAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to directory service: %s", err)
	}
	defer conn.Close()
	directoryClient := directory.NewDirectoryClient(conn)

	var exCommsClient excomms.ExCommsClient
	if *flagExcommsAddr == "stub" {
		exCommsClient = stub.NewStubExcommsClient()
	} else {
		conn, err = grpc.Dial(*flagExcommsAddr, grpc.WithInsecure())
		if err != nil {
			golog.Fatalf("Unable to connect to excomms service: %s", err)
		}
		exCommsClient = excomms.NewExCommsClient(conn)
	}

	sg := sendgrid.NewSendGridClientWithApiKey(*flagSendGridKey)
	branchCli := branch.NewClient(*flagBranchKey)

	eSNS, err := awsutil.NewEncryptedSNS(*flagKMSKeyARN, kms.New(awsSession), sns.New(awsSession))
	if err != nil {
		golog.Fatalf("Unable to initialize enrypted sns: %s", err.Error())
		return
	}

	dl := dal.New(db, environment.GetCurrent())
	srv := server.New(dl, nil, directoryClient, exCommsClient, eSNS, branchCli, sg, *flagFromEmail, *flagServiceNumber, *flagEventsTopic, *flagWebInviteURL)
	invite.InitMetrics(srv, svc.MetricsRegistry.Scope("server"))
	s := grpc.NewServer()
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
			log.Fatal(err)
		}
	}()

	r := mux.NewRouter()
	handlers.InitRoutes(r, dl)
	h := httputil.LoggingHandler(r, "media", *flagBehindProxy, nil)

	golog.Infof("Invite HTTP Listening on %s...", *flagHTTPListenAddr)
	httpSrv := &http.Server{
		Addr:           *flagHTTPListenAddr,
		Handler:        httputil.FromContextHandler(shttputil.CompressResponse(h, httputil.CompressResponse)),
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		httpSrv.ListenAndServe()
	}()

	boot.WaitForTermination()
}
