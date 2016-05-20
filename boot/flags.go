package boot

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

var flagVersion = flag.Bool("version", false, "Display version information and exit")

// ParseFlags looks for config variables in the environment and the command line
// flags with the command line taking precedence. The environment variable is generated
// as `envPrefix + strings.ToUpper(strings.Replace(flagName, ".", "_", -1))`
func ParseFlags(envPrefix string) {
	// Check environment first so that command line flags take priority
	flag.VisitAll(func(f *flag.Flag) {
		key := envPrefix + strings.ToUpper(strings.Replace(f.Name, ".", "_", -1))
		if s := os.Getenv(key); s != "" {
			if err := f.Value.Set(s); err != nil {
				log.Fatalf("Environment variable %s ('%s') not valid for flag %s: %s", key, s, f.Name, err)
			}
		}
	})
	flag.Parse()
	if *flagVersion {
		for k, v := range VersionInfo {
			fmt.Printf("%s: %s\n", k, v)
		}
		os.Exit(0)
	}
}

func RequiredFlags(flags ...*string) {

}
