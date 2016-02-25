package main

import (
	"flag"
	"log"
	"net"
	"net/http"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/server"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/branch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sendgrid/sendgrid-go"
	"google.golang.org/grpc"
)

var (
	flagAWSAccessKey  = flag.String("aws_access_key", "", "AWS access `key`")
	flagAWSSecretKey  = flag.String("aws_secret_key", "", "AWS secret `key`")
	flagAWSToken      = flag.String("aws_token", "", "AWS `token`")
	flagAWSRegion     = flag.String("aws_region", "", "AWS `region`")
	flagBranchKey     = flag.String("branch_key", "", "Branch API key")
	flagDirectoryAddr = flag.String("directory_addr", "", "`host:port` of directory service")
	flagFromEmail     = flag.String("from_email", "", "Email address from which to send invites")
	flagListen        = flag.String("listen_addr", ":5001", "`host:port` for grpc server")
	flagSendGridKey   = flag.String("sendgrid_key", "", "SendGrid API `key`")
	flagEventsTopic   = flag.String("events_topic", "", "SNS topic ARN for publishing events")
	flagKMSKeyARN     = flag.String("kms_key_arn", "", "the arn of the master key that should be used to encrypt outbound and decrypt inbound data")

	// For local development
	flagDynamoDBEndpoint = flag.String("dynamodb_endpoint", "", "DynamoDB endpoint `URL` (for local development)")
)

func init() {
	// Disable the built in grpc tracing and use our own
	grpc.EnableTracing = false
}

func main() {
	boot.ParseFlags("INVITE_")
	boot.InitService()

	if *flagFromEmail == "" {
		golog.Fatalf("from_email required")
	}
	if *flagSendGridKey == "" {
		golog.Fatalf("sendgrid_key required")
	}

	awsConfig, err := awsutil.Config(*flagAWSRegion, *flagAWSAccessKey, *flagAWSSecretKey, *flagAWSToken)
	if err != nil {
		golog.Fatalf(err.Error())
	}
	awsSession := session.New(awsConfig)

	db := dynamodb.New(awsSession)

	// Start management server
	go func() {
		golog.Fatalf("%s", http.ListenAndServe(":8005", nil))
	}()

	if *flagDirectoryAddr == "" {
		golog.Fatalf("Directory service not configured")
	}
	conn, err := grpc.Dial(*flagDirectoryAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to directory service: %s", err)
	}
	defer conn.Close()
	directoryClient := directory.NewDirectoryClient(conn)

	sg := sendgrid.NewSendGridClientWithApiKey(*flagSendGridKey)
	branchCli := branch.NewClient(*flagBranchKey)

	eSNS, err := awsutil.NewEncryptedSNS(*flagKMSKeyARN, kms.New(awsSession), sns.New(awsSession))
	if err != nil {
		golog.Fatalf("Unable to initialize enrypted sns: %s", err.Error())
		return
	}

	srv := server.New(dal.New(db, environment.GetCurrent()), nil, directoryClient, eSNS, branchCli, sg, *flagFromEmail, *flagEventsTopic)
	s := grpc.NewServer()
	defer s.Stop()
	invite.RegisterInviteServer(s, srv)
	golog.Infof("Starting invite service on %s...", *flagListen)

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

	boot.WaitForTermination()
}
