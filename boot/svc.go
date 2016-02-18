package boot

import (
	"flag"
	"net/http"
	_ "net/http/pprof" // imported for side-effect of registering HTTP handlers
	"os"
	"os/signal"
	"syscall"

	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/golog"
)

var (
	flagDebug          = flag.Bool("debug", false, "Enable debug logging")
	flagEnv            = flag.String("env", "", "Execution environment")
	flagManagementAddr = flag.String("management_addr", ":9000", "host:port of management HTTP server")
)

// InitService should be called at the start of a service after flags have been parsed.
func InitService() {
	if *flagEnv == "" {
		golog.Fatalf("-env flag required")
	}
	environment.SetCurrent(*flagEnv)

	if *flagDebug {
		golog.Default().SetLevel(golog.DEBUG)
	}

	// TODO: this can be expanded in the future to support registering custom health checks (e.g. checking connection to DB)
	http.Handle("/health-check", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))

	// Start management server
	go func() {
		golog.Fatalf("%s", http.ListenAndServe(*flagManagementAddr, nil))
	}()
}

// WaitForTermination waits for an INT or TERM signal.
func WaitForTermination() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	select {
	case sig := <-ch:
		golog.Infof("Quitting due to signal %s", sig.String())
		break
	}
}
