package config

import (
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

func (s *Stats) StartReporters(statsRegistry metrics.Registry) {
	if s == nil {
		return
	}

	if s.Source == "" {
		name, err := os.Hostname()
		if err == nil {
			s.Source = name
		} else {
			s.Source = "unknown"
		}
	}

	statsRegistry.Add("runtime", metrics.RuntimeMetrics)
	if s.GraphiteAddr != "" {
		statsReporter := reporter.NewGraphiteReporter(
			statsRegistry, time.Minute, s.GraphiteAddr, s.Source,
			map[string]float64{"median": 0.5, "p75": 0.75, "p90": 0.9, "p99": 0.99, "p999": 0.999})
		statsReporter.Start()
	}

	filteredRegistry := metrics.NewFilterdRegistry(statsRegistry, statsExportIncludes, statsExportExcludes)
	if s.LibratoUsername != "" && s.LibratoToken != "" {
		statsReporter := reporter.NewLibratoReporter(
			filteredRegistry, time.Minute, s.LibratoUsername, s.LibratoToken, s.Source,
			map[string]float64{"median": 0.5, "p90": 0.9, "p99": 0.99})
		statsReporter.Start()
	}
	if s.StatHatKey != "" {
		statsReporter := reporter.NewStatHatReporter(
			filteredRegistry, time.Minute, s.StatHatKey, "",
			map[string]float64{"median": 0.5, "p90": 0.9, "p99": 0.99})
		statsReporter.Start()
	}
}
