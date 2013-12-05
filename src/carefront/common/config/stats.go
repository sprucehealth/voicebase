package config

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/samuel/go-metrics/metrics"
	"github.com/samuel/go-metrics/reporter"
)

type Stats struct {
	// Stats Reporters
	Source          string `long:"stats_source" description:"Source for stats (e.g. hostname)"`
	GraphiteAddr    string `long:"graphite.addr" description:"Graphite addr:port"`
	LibratoUsername string `long:"librato_username" description:"Librato Metrics username"`
	LibratoToken    string `long:"librato_token" description:"Librato Metrics token"`
	StatHatKey      string `long:"stathat_key" description:"StatHat EZKey"`
}

var (
	statsExportIncludes = []*regexp.Regexp{
		regexp.MustCompile(`/connections`),
		regexp.MustCompile(`/requests`),
		regexp.MustCompile(`/runtime/(gc|heap)/`),
		regexp.MustCompile(`/runtime/(Frees|Mallocs)$`),
	}
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
	if s.Stats.GraphiteAddr != "" {
		statsReporter := reporter.NewGraphiteReporter(
			statsRegistry, time.Minute, s.Stats.GraphiteAddr, s.Stats.Source,
			map[string]float64{"median": 0.5, "p75": 0.75, "p90": 0.9, "p99": 0.99, "p999": 0.999})
		statsReporter.Start()
	}

	filteredRegistry := metrics.NewFilterdRegistry(statsRegistry, statsExportIncludes, statsExportExcludes)
	if s.Stats.LibratoUsername != "" && s.Stats.LibratoToken != "" {
		statsReporter := reporter.NewLibratoReporter(
			filteredRegistry, time.Minute, s.Stats.LibratoUsername, s.Stats.LibratoToken, s.Stats.Source,
			map[string]float64{"median": 0.5, "p90": 0.9, "p99": 0.99})
		statsReporter.Start()
	}
	if s.Stats.StatHatKey != "" {
		statsReporter := reporter.NewStatHatReporter(
			filteredRegistry, time.Minute, s.Stats.StatHatKey, "",
			map[string]float64{"median": 0.5, "p90": 0.9, "p99": 0.99})
		statsReporter.Start()
	}
}
