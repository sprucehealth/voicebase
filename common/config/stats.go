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
	GraphiteAddr    string `long:"graphite.addr" description:"Graphite addr:port"`
	LibratoUsername string `long:"librato_username" description:"Librato Metrics username"`
	LibratoToken    string `long:"librato_token" description:"Librato Metrics token"`
	StatHatKey      string `long:"stathat_key" description:"StatHat EZKey"`
	CloudWatch      bool   `long:"cloudwatch" description:"Enable CloudWatch stats gathering"`
}

var (
	statsExportIncludes    []*regexp.Regexp = nil
	statsExportExcludes    []*regexp.Regexp = nil
	statsCloudWatchExports []*regexp.Regexp = nil
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
		statsReporter := reporter.NewGraphiteReporter(statsRegistry, time.Minute, true, s.Stats.GraphiteAddr, s.Stats.Source)
		statsReporter.Start()
	}

	filteredRegistry := metrics.NewFilterdRegistry(statsRegistry, statsExportIncludes, statsExportExcludes)
	if s.Stats.LibratoUsername != "" && s.Stats.LibratoToken != "" {
		statsReporter := reporter.NewLibratoReporter(
			filteredRegistry, time.Minute, true, s.Stats.LibratoUsername, s.Stats.LibratoToken, s.Stats.Source)
		statsReporter.Start()
	}
	if s.Stats.StatHatKey != "" {
		statsReporter := reporter.NewStatHatReporter(filteredRegistry, time.Minute, true, s.Stats.StatHatKey, "")
		statsReporter.Start()
	}

	if s.Stats.CloudWatch && statsCloudWatchExports != nil {
		auth := func() (string, string, string) {
			auth, err := s.AWSAuth()
			if err != nil {
				log.Printf("config/stats: failed to get AWS auth: %+v", err)
				return "", "", ""
			}
			keys := auth.Keys()
			return keys.AccessKey, keys.SecretKey, keys.Token
		}
		filteredRegistry := metrics.NewFilterdRegistry(statsRegistry, statsCloudWatchExports, nil)
		statsReporter := reporter.NewCloudWatchReporter(filteredRegistry, time.Minute, true, s.AWSRegion, auth,
			fmt.Sprintf("%s-%s", s.Environment, s.AppName), nil, time.Second*10)
		statsReporter.Start()
	}
}
