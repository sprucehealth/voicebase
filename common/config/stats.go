package config

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/reporter"
)

type Stats struct {
	Source          string `long:"stats_source" description:"Source for stats (e.g. hostname)"` // Stats Reporters
	LibratoUsername string `long:"librato_username" description:"Librato Metrics username"`
	LibratoToken    string `long:"librato_token" description:"Librato Metrics token"`
}

var (
	statsExportIncludes []*regexp.Regexp = nil
	statsExportExcludes []*regexp.Regexp = nil
)

func (s *BaseConfig) StartReporters(statsRegistry metrics.Registry) {
	if s == nil {
		return
	}

	if s.Stats.Source == "" {
		hostname, err := os.Hostname()
		if err == nil {
			s.Stats.Source = fmt.Sprintf("%s-%s-%s", s.Environment, s.AppName, hostname)
		} else {
			s.Stats.Source = "unknown"
			log.Printf("Unable to get local hostname. Using 'unknown' for stats source.")
		}
	}

	statsRegistry.Add("runtime", metrics.RuntimeMetrics)

	filteredRegistry := metrics.NewFilterdRegistry(statsRegistry, statsExportIncludes, statsExportExcludes)
	if s.Stats.LibratoUsername != "" && s.Stats.LibratoToken != "" {
		statsReporter := reporter.NewLibratoReporter(
			filteredRegistry, time.Minute, true, s.Stats.LibratoUsername, s.Stats.LibratoToken, s.Stats.Source)
		statsReporter.Start()
	}
}
