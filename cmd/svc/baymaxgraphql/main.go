package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof" // imported for implicitly registered handlers
	"os"
	"path"
	"strings"

	"github.com/rs/cors"
	"github.com/sprucehealth/backend/boot"
	mediastore "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/media"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/stub"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
)

var (
	flagListenAddr    = flag.String("listen_addr", "127.0.0.1:8080", "host:port to listen on")
	flagDebugAddr     = flag.String("debug_addr", "127.0.0.1:9090", "host:port to listen for debug interface")
	flagResourcePath  = flag.String("resource_path", "", "Path to resources (defaults to use GOPATH)")
	flagEnv           = flag.String("env", "", "Execution environment")
	flagAPIDomain     = flag.String("api_domain", "", "API `domain`")
	flagWebDomain     = flag.String("web_domain", "", "Web `domain`")
	flagStorageBucket = flag.String("storage_bucket", "", "storage bucket for media")
	flagSigKeys       = flag.String("signature_keys_csv", "", "csv signature keys")

	// Services
	flagAuthAddr                 = flag.String("auth_addr", "", "host:port of auth service")
	flagDirectoryAddr            = flag.String("directory_addr", "", "host:port of directory service")
	flagExCommsAddr              = flag.String("excomms_addr", "", "host:port of excomms service")
	flagThreadingAddr            = flag.String("threading_addr", "", "host:port of threading service")
	flagSQSDeviceRegistrationURL = flag.String("sqs_device_registration_url", "", "the sqs url for device registration messages")

	// AWS
	flagAWSAccessKey = flag.String("aws_access_key", "", "access key for aws")
	flagAWSSecretKey = flag.String("aws_secret_key", "", "secret key for aws")
	flagAWSRegion    = flag.String("aws_region", "us-east-1", "aws region")
)

func main() {
	boot.ParseFlags("BAYMAXGRAPHQL_")
	if *flagEnv == "" {
		fmt.Fprintf(os.Stderr, "Flag -env is required\n")
		os.Exit(1)
	}
	environment.SetCurrent(*flagEnv)

	if *flagDebugAddr != "" {
		go func() {
			http.ListenAndServe(*flagDebugAddr, nil)
		}()
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
	directoryClient := directory.NewDirectoryClient(conn)

	if *flagThreadingAddr == "" {
		golog.Fatalf("Threading service not configured")
	}
	conn, err = grpc.Dial(*flagThreadingAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to threading service: %s", err)
	}
	threadingClient := threading.NewThreadsClient(conn)

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

	baseConfig := &config.BaseConfig{
		AppName:      "baymaxgraphql",
		AWSRegion:    *flagAWSRegion,
		AWSSecretKey: *flagAWSSecretKey,
		AWSAccessKey: *flagAWSAccessKey,
	}

	if *flagSQSDeviceRegistrationURL == "" {
		golog.Fatalf("Notification service not configured")
	}
	awsSession := baseConfig.AWSSession()
	notificationClient := notification.NewClient(&notification.ClientConfig{
		SQSDeviceRegistrationURL: *flagSQSDeviceRegistrationURL,
		Session:                  awsSession,
	})

	r := mux.NewRouter()
	if *flagSigKeys == "" {
		golog.Fatalf("Signature keys not specified")
	}
	if *flagAPIDomain == "" {
		golog.Fatalf("API Domain not specified")
	}
	if *flagWebDomain == "" {
		golog.Fatalf("Web domain not specified")
	}
	if *flagStorageBucket == "" {
		golog.Fatalf("Storage bucket not specified")
	}

	sigKeys := strings.Split(*flagSigKeys, ",")
	sigKeysByteSlice := make([][]byte, len(sigKeys))
	for i, sk := range sigKeys {
		sigKeysByteSlice[i] = []byte(sk)
	}
	signer, err := sig.NewSigner(sigKeysByteSlice, nil)
	if err != nil {
		golog.Fatalf("Failed to create signer: %s", err.Error())
	}

	ms := mediastore.NewSigner("https://"+*flagAPIDomain+"/media", signer)

	corsOrigins := []string{"https://" + *flagWebDomain}

	gqlHandler := NewGraphQL(authClient, directoryClient, threadingClient, exCommsClient, notificationClient, ms)
	r.Handle("/graphql", httputil.ToContextHandler(cors.New(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{httputil.Get, httputil.Options, httputil.Post},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).Handler(httputil.FromContextHandler(gqlHandler))))

	mediaHandler := NewMediaHandler(authClient, media.New(storage.NewS3(awsSession, *flagStorageBucket, "media"), storage.NewS3(awsSession, *flagStorageBucket, "media-cache"), 0, 0), ms)

	r.Handle("/media", httputil.ToContextHandler(cors.New(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{httputil.Get, httputil.Options},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).Handler(httputil.FromContextHandler(mediaHandler))))

	if *flagResourcePath == "" {
		if p := os.Getenv("GOPATH"); p != "" {
			*flagResourcePath = path.Join(p, "src/github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/resources")
		}
	}
	if *flagResourcePath != "" {
		r.PathPrefix("/graphiql/").Handler(httputil.FileServer(http.Dir(*flagResourcePath)))
	}
	fmt.Printf("Listening on %s\n", *flagListenAddr)

	server := &http.Server{
		Addr:           *flagListenAddr,
		Handler:        httputil.FromContextHandler(r),
		MaxHeaderBytes: 1 << 20,
	}
	server.ListenAndServe()
}
