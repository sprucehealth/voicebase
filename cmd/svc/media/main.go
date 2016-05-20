package main

import (
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/handlers"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/urlutil"
	"github.com/sprucehealth/backend/shttputil"
	"github.com/sprucehealth/backend/svc/auth"
	"google.golang.org/grpc"
)

var (
	flagHTTPListenAddr     = flag.String("http_listen_addr", ":8081", "host:port to listen on for http requests")
	flagWebDomain          = flag.String("web_domain", "", "Web `domain`")
	flagMediaAPIDomain     = flag.String("media_api_domain", "", "Media API `domain`")
	flagMediaStorageBucket = flag.String("media_storage_bucket", "", "storage bucket for media")
	flagSigKeys            = flag.String("signature_keys_csv", "", "csv signature keys")
	flagBehindProxy        = flag.Bool("behind_proxy", false, "Flag to indicate when the service is behind a proxy")

	// Services
	flagAuthAddr = flag.String("auth_addr", "", "host:port of auth service")
)

func main() {
	svc := boot.NewService("media")
	awsSession, err := svc.AWSSession()
	if err != nil {
		golog.Fatalf("Failed to create AWS session: %s", err)
	}

	if *flagAuthAddr == "" {
		golog.Fatalf("Auth service not configured")
	}
	conn, err := grpc.Dial(*flagAuthAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to auth service: %s", err)
	}
	authClient := auth.NewAuthClient(conn)

	if *flagMediaAPIDomain == "" {
		golog.Fatalf("Media API Domain not specified")
	}

	if *flagMediaStorageBucket == "" {
		golog.Fatalf("Media Storage bucket not specified")
	}
	urlSigner := urlutil.NewSigner("https://"+*flagMediaAPIDomain, createSigner(*flagSigKeys), clock.New())
	r := mux.NewRouter()
	handlers.InitRoutes(r, awsSession, authClient, urlSigner, *flagWebDomain)
	h := httputil.LoggingHandler(r, shttputil.WebRequestLogger(*flagBehindProxy))

	fmt.Printf("HTTP Listening on %s\n", *flagHTTPListenAddr)
	server := &http.Server{
		Addr:           *flagHTTPListenAddr,
		Handler:        httputil.FromContextHandler(shttputil.CompressResponse(h, httputil.CompressResponse)),
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		server.ListenAndServe()
	}()

	boot.WaitForTermination()
}

func createSigner(keyCSV string) *sig.Signer {
	if keyCSV == "" {
		golog.Fatalf("Failed to create signer: non empty keys csv required")
	}

	sigKeys := strings.Split(keyCSV, ",")
	sigKeysByteSlice := make([][]byte, len(sigKeys))
	for i, sk := range sigKeys {
		sigKeysByteSlice[i] = []byte(sk)
	}

	signer, err := sig.NewSigner(sigKeysByteSlice, nil)
	if err != nil {
		golog.Fatalf("Failed to create signer: %s", err.Error())
	}
	return signer
}
