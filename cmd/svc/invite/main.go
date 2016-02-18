package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/server"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/branch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"google.golang.org/grpc"
)

var (
	flagAWSAccessKey  = flag.String("aws_access_key", "", "AWS access `key`")
	flagAWSSecretKey  = flag.String("aws_secret_key", "", "AWS secret `key`")
	flagAWSToken      = flag.String("aws_token", "", "AWS `token`")
	flagAWSRegion     = flag.String("aws_region", "", "AWS `region`")
	flagBranchKey     = flag.String("branch_key", "", "Branch API key")
	flagDebug         = flag.Bool("debug", false, "Enable debug logging")
	flagDirectoryAddr = flag.String("directory_addr", "", "`host:port` of directory service")
	flagEnv           = flag.String("env", "", "`Environment` (local, dev, staging, prod)")
	flagFromEmail     = flag.String("from_email", "", "Email address from which to send invites")
	flagListen        = flag.String("listen_addr", ":5001", "`host:port` for grpc server")
	flagSendGridKey   = flag.String("sendgrid_key", "", "SendGrid API `key`")

	// For local development
	flagDynamoDBEndpoint = flag.String("dynamodb_endpoint", "", "DynamoDB endpoint `URL` (for local development)")
)

func init() {
	// Disable the built in grpc tracing and use our own
	grpc.EnableTracing = false
}

func createAWSSession() (*session.Session, error) {
	var creds *credentials.Credentials
	if *flagAWSAccessKey != "" && *flagAWSSecretKey != "" {
		creds = credentials.NewStaticCredentials(*flagAWSAccessKey, *flagAWSSecretKey, *flagAWSToken)
	} else {
		creds = credentials.NewEnvCredentials()
		if v, err := creds.Get(); err != nil || v.AccessKeyID == "" || v.SecretAccessKey == "" {
			creds = ec2rolecreds.NewCredentials(session.New(), func(p *ec2rolecreds.EC2RoleProvider) {
				p.ExpiryWindow = time.Minute * 5
			})
		}
	}
	if *flagAWSRegion == "" {
		az, err := awsutil.GetMetadata(awsutil.MetadataAvailabilityZone)
		if err != nil {
			return nil, err
		}
		// Remove the last letter of the az to get the region (e.g. us-east-1a -> us-east-1)
		*flagAWSRegion = az[:len(az)-1]
	}

	awsConfig := &aws.Config{
		Credentials: creds,
		Region:      flagAWSRegion,
	}
	return session.New(awsConfig), nil
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

	awsSession, err := createAWSSession()
	if err != nil {
		golog.Fatalf(err.Error())
	}

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

	srv := server.New(dal.New(db, *flagEnv), nil, directoryClient, branchCli, sg, *flagFromEmail)
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
