package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof" // imported for implicitly registered handlers
	"os"
	"path"

	"github.com/sprucehealth/backend/libs/mux"

	"github.com/rs/cors"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
)

var (
	flagListenAddr   = flag.String("l", "127.0.0.1:8080", "host:port to listen on")
	flagDebugAddr    = flag.String("debug.addr", "127.0.0.1:9090", "host:port to listen for debug interface")
	flagResourcePath = flag.String("respath", "", "Path to resources (defaults to use GOPATH)")
	flagEnv          = flag.String("env", "", "Execution environment")

	// Services
	flagAuthAddr      = flag.String("auth.addr", "", "host:port of auth service")
	flagDirectoryAddr = flag.String("directory.addr", "", "host:port of direcotry service")
	flagExCommsAddr   = flag.String("excomms.addr", "", "host:port of excomms service")
	flagThreadingAddr = flag.String("threading.addr", "", "host:port of threading service")
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
	conn, err = grpc.Dial(*flagExCommsAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to excomms service: %s", err)
	}
	exCommsClient := excomms.NewExCommsClient(conn)

	r := mux.NewRouter()

	gqlHandler := NewGraphQL(authClient, directoryClient, threadingClient, exCommsClient)
	r.Handle("/graphql", httputil.ToContextHandler(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{httputil.Get, httputil.Options, httputil.Post},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
	}).Handler(httputil.FromContextHandler(gqlHandler))))
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
