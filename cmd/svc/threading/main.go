package main

import (
	"flag"
	"net"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/server"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
)

var (
	flagAWSAccessKey       = flag.String("aws_access_key", "", "AWS access key")
	flagAWSSecretKey       = flag.String("aws_secret_key", "", "AWS secret key")
	flagAWSToken           = flag.String("aws_token", "", "AWS token")
	flagAWSRegion          = flag.String("aws_region", "", "AWS region")
	flagDBName             = flag.String("db_name", "threading", "Database name")
	flagDBHost             = flag.String("db_host", "127.0.0.1", "Database host")
	flagDBPort             = flag.Int("db_port", 3306, "Database port")
	flagDBUser             = flag.String("db_user", "", "Database username")
	flagDBPass             = flag.String("db_pass", "", "Database password")
	flagDBCACert           = flag.String("db_ca_cert", "", "Path to database CA certificate")
	flagDBTLS              = flag.String("db_tls", "false", "Enable TLS for database connection (one of 'true', 'false', 'skip-verify'). Ignored if CA cert provided.")
	flagListen             = flag.String("listen_addr", ":5001", "Address on which to listen")
	flagSNSTopicARN        = flag.String("sns_topic_arn", "", "SNS topic ARN to publish new messages to")
	flagSQSNotificationURL = flag.String("sqs_notification_url", "", "the sqs url for notification messages")
	flagDirectoryAddr      = flag.String("directory_addr", "", "host:port of directory service")
	flagWebDomain          = flag.String("web_domain", "", "the domain of the web app")
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
	boot.ParseFlags("THREADING_")
	boot.InitService()

	db, err := dbutil.ConnectMySQL(&dbutil.DBConfig{
		Host:          *flagDBHost,
		Port:          *flagDBPort,
		Name:          *flagDBName,
		User:          *flagDBUser,
		Password:      *flagDBPass,
		EnableTLS:     *flagDBTLS == "true" || *flagDBTLS == "skip-verify",
		SkipVerifyTLS: *flagDBTLS == "skip-verify",
		CACert:        *flagDBCACert,
	})
	if err != nil {
		golog.Fatalf(err.Error())
	}

	awsSession, err := createAWSSession()
	if err != nil {
		golog.Fatalf(err.Error())
	}

	sns := sns.New(awsSession)

	// Start management server
	go func() {
		golog.Fatalf("%s", http.ListenAndServe(":8005", nil))
	}()

	var notificationClient notification.Client
	if *flagSQSNotificationURL != "" {
		notificationClient = notification.NewClient(sqs.New(awsSession), &notification.ClientConfig{
			SQSNotificationURL: *flagSQSNotificationURL,
		})
	}

	if *flagWebDomain == "" {
		golog.Fatalf("Web domain not configured")
	}
	if *flagDirectoryAddr == "" {
		golog.Fatalf("Directory service not configured")
	}
	conn, err := grpc.Dial(*flagDirectoryAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to directory service: %s", err)
	}
	directoryClient := directory.NewDirectoryClient(conn)

	s := grpc.NewServer()
	threading.RegisterThreadsServer(s, server.NewThreadsServer(clock.New(), dal.New(db), sns, *flagSNSTopicARN, *flagWebDomain, notificationClient, directoryClient))
	golog.Infof("Starting Threads service on %s...", *flagListen)

	ln, err := net.Listen("tcp", *flagListen)
	if err != nil {
		golog.Fatalf("failed to listen on %s: %v", *flagListen, err)
	}
	go func() {
		s.Serve(ln)
	}()

	boot.WaitForTermination()

	// cert, err := tls.X509KeyPair(localTLSCert, localTLSKey)
	// if err != nil {
	// 	golog.Fatalf("Failed to generate key pair: %v", err)
	// }
	// tlsConfig := &tls.Config{
	// 	MinVersion:   tls.VersionTLS12,
	// 	Certificates: []tls.Certificate{cert},
	// }
	// creds := credentials.NewTLS(tlsConfig)
	// grpcServer := grpc.NewServer(grpc.Creds(creds))
	// threading.RegisterThreadsServer(grpcServer, server.NewThreadsServer(dal.New(db)))
	// golog.Infof("Listening on %s...", *flagListen)
	// grpcServer.Serve(lis)
}

var (
	localTLSCert = []byte(`-----BEGIN CERTIFICATE-----
MIIDGDCCAgKgAwIBAgIRAOvlgNu24IVI52mjWfaHiQIwCwYJKoZIhvcNAQELMBIx
EDAOBgNVBAoTB0FjbWUgQ28wIBcNNzAwMTAxMDAwMDAwWhgPMjA4NDAxMjkxNjAw
MDBaMBIxEDAOBgNVBAoTB0FjbWUgQ28wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAw
ggEKAoIBAQCiMm/EAvYlu+eRDdcBqxcGKO59vrxVkSz8QLVShajUPl4jWFo8xZHG
MsNBLmUXFulkIRQStvFzfpo9/QHWDyvUmrNMy5P/LE54x9EO/kmjJu1B8ReRqdyD
WsEej3RM9WBo+fISY+2yMMHbN/3PuZzIHVMGl45/PcXuCs7OMYOQWgn0yURYSvP/
ltwDrLxebgLV13S3fk9iJf9CjBV6beEMjAPbm6I+s4mtJff/74ci7nHkMyxOT1PS
w5HJW6fdmFpiId5tJd9k4MNmkRPnxHlKxwCjGi0JzAKA32qYgqqPb8OBg2RP+5JD
usC+2e/ohx2s/TZO+lPb+LsUyKDjmAFtAgMBAAGjazBpMA4GA1UdDwEB/wQEAwIA
pDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MDEGA1UdEQQq
MCiCDiouc3BydWNlLmxvY2FshwR/AAABhxAAAAAAAAAAAAAAAAAAAAABMAsGCSqG
SIb3DQEBCwOCAQEAST9NUS/YQKpj9oFY6QOR4tDro+UTlN5DkMVUBacX/alDj58q
bPFs6XwsPWnbA3ZQtWq0zMaOyFWcj1jH5tsc5RUVDbhcUmrhwc1MdzWYfiTMgLMp
7M59n0dt3icL6WYWeM+Gb2YB1wIe9I2MxqB7RZnMPocyaEjXA06wfWVrsYOH+0XV
UQ95EPwON7Izclw7CVQHnYXYK3uGtFjIRO7d9EC0KZjcURb+rKvzY0+74CxyyW/9
MInfGcQScPfqGGmUoBw8tA7wRJYLCbCTAIa3F83ikPsm/T/xg6kuml1IwNfyPiMY
b56u79PVFpIezbwxtGnZodROrIrSXHUs3OAlIg==
-----END CERTIFICATE-----`)
	localTLSKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAojJvxAL2JbvnkQ3XAasXBijufb68VZEs/EC1UoWo1D5eI1ha
PMWRxjLDQS5lFxbpZCEUErbxc36aPf0B1g8r1JqzTMuT/yxOeMfRDv5JoybtQfEX
kancg1rBHo90TPVgaPnyEmPtsjDB2zf9z7mcyB1TBpeOfz3F7grOzjGDkFoJ9MlE
WErz/5bcA6y8Xm4C1dd0t35PYiX/QowVem3hDIwD25uiPrOJrSX3/++HIu5x5DMs
Tk9T0sORyVun3ZhaYiHebSXfZODDZpET58R5SscAoxotCcwCgN9qmIKqj2/DgYNk
T/uSQ7rAvtnv6IcdrP02TvpT2/i7FMig45gBbQIDAQABAoIBAQCIwpRApuqbWHvh
b9T5gCQyunKVLi0ozPcsXvdEdJStGUVQ8h9sHH5Uqtq96/uq41O5bLa7LOwboQU2
/Uz+C96+Lg6+0uyf/ODRsFHTHZBDdAAbWMixtpLLYstxFC5Q8ZjwCsgUv5NdawUZ
7XUiIHRUu30VEtdA7Homw5Aqhc9T92+rlWASJdMD9WQJ+h1xQDcnqb/LsihnV6Od
01rT4DOtDfcgJsgHzseCUOiJiuQ5c/AILiVWB+atNxRbsSHV175/nllIbX/C9UOF
WuuAvXPhhRoFX4CxVEhseNQQKpqlX0FK6dibYC+aWkiclaqvd/52LX9CXYVYs34s
C6VarDkVAoGBANEr9tJhFG+H5VaSMLp55uhlmTJ6JwBne9sF0HC8MHgKIg4G8v2H
UDRQ98oi8hzgRKz3xvrd2wLCEaPAQSfx5cY7tGiR6Y/fwyX/uRakD2a8dMPhhttq
2Vt4x0QrFahZRLoMF1NiOcaNwHrzRm6YP7vm9X2CjYdWj+CBqWGIF7drAoGBAMaC
Qr+vwhr/9Wsmnmh5OK7lE5IWV8tjh+fnLjU5FflNykKOs0nhNDQFw59XATcKEti3
+FSvK9DYOSNU38li+njzHb2mnQlYjae616IcyudWW6J3LRCereUHJxjyO9szEbwK
VNER9ncg7LoJIBa0YATkI3Jc95jJUk6RBQdt/NiHAoGAOwBBsPn9P7B/ejnmUNNN
1MPDwL8//RczkoZDU2lh6ppBHN/M7sKaVwd3vaa50HdaJ8gEcoLd4htHyn7SYigT
fiUdMFnoHdMqQq+tT7ubNIl4DkCxP3cWNH0PCCV3CHOVtTzv329XiLA3WPcCKPP9
Fk2BdZO7xC8gil1In+A5gF0CgYBISsn6OwzKfmqnEhJgY70j3GMLMb3ZYS7uYn+u
fFKnTxAYuxVKE4zKYUsDrVDQ9Yc1i5IRbRXc4dG1L0Ssd7JV99vd5F6ON8Smz+GV
tTyjkQygFxy/T7pujPNNH3Jy+p87xttqpEsIyWHMwmQAQMIzJc5O6NJ2vuKNoDyf
nwuU4wKBgQC9ByjK1nGl5xbgQmWBn+smBZhWwY46lDLIxWFXpHzTbAFlVJlUp7H4
HjuhdXVa8R908jY2phFN9NjkEFXyUUuar9VClE5V5NSD8WYsAkPrrpYNw3BQVJRP
J15BERU7hluvxXOZn5wenPP0DcDqZX/34dNPE58CKtzlDP/UlpSqzQ==
-----END RSA PRIVATE KEY-----`)
)
