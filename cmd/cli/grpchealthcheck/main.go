// grpchealthcheck is a tool to call the health check endpoitn on a gRPC
// service. It is intended to be used with Consul script checks and follows
// the expected output at https://www.consul.io/docs/agent/checks.html.
//
// It exits with 0 for success, 3 for non-serving, and 255 for other failure.
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sprucehealth/backend/boot"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	flagCACertPath := flag.String("ca_cert_path", "", "Path to CA cert")
	flagServiceAddr := flag.String("addr", "", "Address of service to health check")
	flagServiceName := flag.String("name", "", "Name of service to health check (normally not set)")
	flagTimeout := flag.Duration("t", time.Second*2, "Timeout for check call")
	flagTLS := flag.Bool("tls", false, "Use TLS to talk to service")
	flag.Parse()

	var tlsConfig *tls.Config
	if *flagTLS {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true, // Since we're always using IP ignore the server name check
		}
		if *flagCACertPath != "" {
			ca, err := boot.CAFromFile(*flagCACertPath)
			if err != nil {
				fail("Failed to load CA pool: %s", err)
			}
			tlsConfig.RootCAs = ca
		}
	}

	// Record the time before dial so we can account for any used time in the total timeout
	start := time.Now()
	cn, err := boot.DialGRPC("grpchealthcheck", *flagServiceAddr, tlsConfig, grpc.WithBlock(), grpc.WithTimeout(*flagTimeout))
	if err != nil {
		fail("Failed to connnect to service: %s\n", err)
	}
	defer cn.Close()
	timeLeft := *flagTimeout - time.Since(start)
	if timeLeft <= 0 {
		fail("Timeout after dial\n")
	}
	cli := grpc_health_v1.NewHealthClient(cn)
	ctx, cancel := context.WithTimeout(context.Background(), timeLeft)
	res, err := cli.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: *flagServiceName})
	if err != nil {
		fail("Check failed: %s\n", err)
	}
	cancel()
	fmt.Println(res.Status.String())
	if res.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		os.Exit(255)
	}
}

func fail(msg string, f ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, f...)
	os.Exit(255)
}
