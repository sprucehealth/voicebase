/*
Awslogidx is a command that feed AWS CloudWatch and CloudTrail logs to ElasticSearch.

CloudTrail is configured to write logs to an S3 bucket and post a notification
to an SNS topic when a new log is written. The SNS topic is setup to enqueue a
message in SQS for each notification.

To index the log we receive message from the SQS queue, pull down the log from
S3, and then parse and index the events in ElasticSearch. Only after the events
have successfully been indexed do we delete the message from the SQS queue. A
unique ID is genreated for each event based on its contents to try and avoid
duplicates do to errors during processing.

CloudWatch Logs are indexed by using the GetLogEvents api method which returns
entries after either a provided timestamp or token. The token is used when
possible, but it is only valid for 24 hours. The token and last event time are
stored in Consul for persistence.

Log entries are bucketed into indexes on ElasticSearch named by the date of the entry.
The format of the indexe names is "log-2006.01.02", and each entry is a JSON document
of the form

	{
		"msg":    "xxx",
		"group":  "xxx",
		"stream": "xxx",
		// Used by Kibana
		"@timestamp": "2006-01-02T15:04:05Z07:00",
		"@version":   "1",
	}

TODO:
  - Can set the check status (warn, fail) based on status of AWS api and ElasticSearch so
    that if either fail then this process fails the check and drops leadership.
  - Metrics
    - messages indexed
    - track calls against AWS api
    - success rate for consul
    - success rate for elasticsearch
    - success rate for aws api
    - bytes stored per group
*/
package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/samuel/go-metrics/metrics"
	"github.com/samuel/go-metrics/reporter"
	"github.com/sprucehealth/backend/consul"
	"github.com/sprucehealth/backend/libs/golog"
)

var (
	flagCleanup         = flag.Bool("cleanup", false, "Delete old indexes and exit")
	flagCloudTrail      = flag.Bool("cloudtrail", false, "Enable CloudTrail log indexing")
	flagConsul          = flag.String("consul", "127.0.0.1:8500", "Consul HTTP API host:port")
	flagElasticSearch   = flag.String("elasticsearch", "127.0.0.1:9200", "ElasticSearch host:port")
	flagLibratoUsername = flag.String("librato.username", "", "Librato Metrics username")
	flagLibratoToken    = flag.String("librato.token", "", "Librato Metrics token")
	flagLibratoSource   = flag.String("librato.source", "", "Librato source")
	flagRetainDays      = flag.Int("retaindays", 60, "Number of days of indexes to retain")
	flagServiceID       = flag.String("id", "", "Service ID for Consul. Only needed when running more than one instance on a host")
	flagVerbose         = flag.Bool("v", false, "Verbose output")
)

var (
	statEvents                 = metrics.NewCounter()
	statSuccessfulGetLogEvents = metrics.NewCounter()
	statFailedGetLogEvents     = metrics.NewCounter()
	statsRegistry              = metrics.NewRegistry().Scope("awslogidx")
)

func init() {
	statsRegistry.Add("events", statEvents)
	statsRegistry.Add("get_log_events/successful", statSuccessfulGetLogEvents)
	statsRegistry.Add("get_log_events/failed", statFailedGetLogEvents)
}

func cleanupIndexes(es *ElasticSearch, days int) {
	aliases, err := es.Aliases()
	if err != nil {
		golog.Errorf("Failed to get index aliases: %s", err.Error())
		return
	}
	var indexList []string
	for index := range aliases {
		if len(index) == 14 && strings.HasPrefix(index, "log-") {
			indexList = append(indexList, index)
		}
	}
	sort.Strings(indexList)
	if len(indexList) > days {
		for _, index := range indexList[:len(indexList)-days] {
			if err := es.DeleteIndex(index); err != nil {
				golog.Errorf("Failed to delete index %s: %s", index, err.Error())
			}
		}
	}
}

func startPeriodicCleanup(es *ElasticSearch, days int, svc *consul.Service) {
	lock := svc.NewLock("service/awslogidx/cleanup", nil, 30*time.Second)
	go func() {
		defer lock.Release()
		for {
			if !lock.Wait() {
				return
			}
			cleanupIndexes(es, days)
			time.Sleep(time.Hour * 24)
		}
	}()
}

type streamInfo struct {
	GroupName     string
	StreamName    string
	LastEventTime int64
	LastIndexTime int64
	NextToken     string
}

func startCloudWatchLogIndexer(es *ElasticSearch, consul *consulapi.Client, svc *consul.Service) error {
	// For now this is using a single lock. If the volume of logs to ingest is
	// too high for a single process then his can be modified to use a lock per
	// group or per stream.
	lock := svc.NewLock("service/awslogidx/cwl", nil, 30*time.Second)

	lastRunTime := time.Time{}
	runDelay := time.Second * 60

	go func() {
		defer lock.Release()
		for {
			if !lock.Wait() {
				return
			}

			if dt := time.Since(lastRunTime); dt < runDelay {
				time.Sleep(runDelay - dt)
				continue
			}
			lastRunTime = time.Now()

			res, err := cwlClient.DescribeLogGroups(&cloudwatchlogs.DescribeLogGroupsInput{})
			if err != nil {
				golog.Errorf("Failed to get log groups: %s", err.Error())
				continue
			}
		groupLoop:
			for _, g := range res.LogGroups {
				if !lock.Locked() {
					break
				}

				res, err := cwlClient.DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
					LogGroupName: g.LogGroupName,
				})
				if err != nil {
					golog.Errorf("Failed to get log stream for group %s: %s", *g.LogGroupName, err.Error())
					continue
				}
				for _, s := range res.LogStreams {
					if !lock.Locked() {
						break
					}
					if s.LastEventTimestamp == nil || *s.LastEventTimestamp == 0 {
						continue
					}
					if indexStream(*g.LogGroupName, s, es, consul) {
						break groupLoop
					}
				}
			}
		}
	}()

	return nil
}

func indexStream(groupName string, stream *cloudwatchlogs.LogStream, es *ElasticSearch, consul *consulapi.Client) bool {
	hash := md5.Sum([]byte(fmt.Sprintf("%s|%s", groupName, *stream.LogStreamName)))
	key := "service/awslogidx/cwl/" + hex.EncodeToString(hash[:])

	log := golog.Context(
		"group", groupName,
		"stream", stream.LogStreamName,
		"key", key)

	kv, _, err := consul.KV().Get(key, nil)
	if err != nil {
		log.Errorf("Consul get failed: %s", err.Error())
		return false
	}

	var info streamInfo
	// var events *cloudwatchlogs.Events
	var modifyIndex uint64
	var res *cloudwatchlogs.GetLogEventsOutput
	if kv != nil {
		if err := json.Unmarshal(kv.Value, &info); err != nil {
			log.Errorf("Unmarshal failed: %s", err.Error())
			return false
		}
		if stream.LastEventTimestamp != nil && *stream.LastEventTimestamp <= info.LastEventTime {
			log.Debugf("No new events since %s", time.Unix(info.LastEventTime, 0).String())
			return false
		}
		modifyIndex = kv.ModifyIndex
		// The next token is only valid for 24 hours so use the timestamp after that
		if time.Now().Unix()-info.LastIndexTime > 23*60*60 {
			log.Debugf("Fetching by start time of %+v", info.LastEventTime)
			res, err = cwlClient.GetLogEvents(&cloudwatchlogs.GetLogEventsInput{
				LogGroupName:  &groupName,
				LogStreamName: stream.LogStreamName,
				StartFromHead: aws.Boolean(true),
				StartTime:     aws.Long(info.LastEventTime),
			})
		} else {
			log.Debugf("Fetching by token")
			res, err = cwlClient.GetLogEvents(&cloudwatchlogs.GetLogEventsInput{
				LogGroupName:  &groupName,
				LogStreamName: stream.LogStreamName,
				StartFromHead: aws.Boolean(true),
				NextToken:     &info.NextToken,
			})
		}
	} else {
		info = streamInfo{
			GroupName:  groupName,
			StreamName: *stream.LogStreamName,
		}
		log.Debugf("Fetching from beginning")
		res, err = cwlClient.GetLogEvents(&cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  &groupName,
			LogStreamName: stream.LogStreamName,
			StartFromHead: aws.Boolean(true),
		})
	}
	if err != nil {
		statFailedGetLogEvents.Inc(1)
		log.Errorf("GetLogEvents failed: %s", err.Error())
		return false
	}
	statSuccessfulGetLogEvents.Inc(1)

	events := res.Events

	statEvents.Inc(uint64(len(events)))

	var buf []byte
	for _, e := range events {
		if *e.Timestamp > info.LastEventTime {
			info.LastEventTime = *e.Timestamp
		}
		h := md5.New()
		t := time.Unix(*e.Timestamp, 0).UTC()
		ts := t.Format(time.RFC3339)
		h.Write([]byte(groupName))
		h.Write([]byte(*stream.LogStreamName))
		h.Write([]byte(ts))
		h.Write([]byte(*e.Message))
		buf = h.Sum(buf[:0])
		id := hex.EncodeToString(buf)
		idx := fmt.Sprintf("log-%s", t.Format("2006.01.02"))
		doc := map[string]interface{}{
			"msg":    e.Message,
			"group":  groupName,
			"stream": stream.LogStreamName,
			// Used by Kibana
			"@timestamp": ts,
			"@version":   "1",
		}
		if err := es.Index(idx, "log", id, doc, t); err != nil {
			log.Errorf("Failed to index: %s", err.Error())
			return true
		}
	}

	info.NextToken = *res.NextForwardToken
	info.LastIndexTime = time.Now().Unix()

	log.Debugf("New info %+v", info)
	b, err := json.Marshal(info)
	if err != nil {
		log.Errorf("Marshal failed: %s", err.Error())
		return false
	}
	kv = &consulapi.KVPair{
		Key:         key,
		Value:       b,
		ModifyIndex: modifyIndex,
	}
	ok, _, err := consul.KV().CAS(kv, nil)
	if err != nil {
		log.Errorf("CAS failed: %s", err.Error())
	} else if !ok {
		log.Warningf("CAS did not match")
		// TODO: get the current value and keep which ever has newer event timestamp
		if _, err := consul.KV().Put(kv, nil); err != nil {
			log.Errorf("Put failed: %s", err.Error())
		}
	}
	return false
}

func main() {
	flag.Parse()

	if *flagVerbose {
		golog.Default().SetLevel(golog.DEBUG)
	}

	if err := setupAWS(); err != nil {
		golog.Fatalf(err.Error())
	}

	if err := setupLibrato(); err != nil {
		golog.Fatalf(err.Error())
	}

	if err := run(); err != nil {
		golog.Fatalf(err.Error())
	}
}

func setupLibrato() error {
	if *flagLibratoUsername == "" || *flagLibratoToken == "" {
		return nil
	}

	source := *flagLibratoSource
	if source == "" {
		var err error
		source, err = os.Hostname()
		if err != nil {
			return err
		}
	}

	statsReporter := reporter.NewLibratoReporter(statsRegistry, time.Minute, true, *flagLibratoUsername, *flagLibratoToken, source)
	statsReporter.Start()
	return nil
}

func run() error {
	es := &ElasticSearch{
		Endpoint: "http://" + *flagElasticSearch,
	}
	if *flagCleanup {
		cleanupIndexes(es, *flagRetainDays)
		return nil
	}

	consulClient, err := consulapi.NewClient(&consulapi.Config{
		Address:    *flagConsul,
		HttpClient: http.DefaultClient,
	})
	if err != nil {
		return err
	}

	svc, err := consul.RegisterService(consulClient, *flagServiceID, "awslogidx", nil, 0)
	if err != nil {
		log.Fatalf("Failed to register service with Consul: %s", err.Error())
	}
	defer svc.Deregister()

	if *flagCloudTrail {
		if err := startCloudTrailIndexer(es); err != nil {
			return err
		}
	}
	if *flagRetainDays > 0 {
		startPeriodicCleanup(es, *flagRetainDays, svc)
	}

	if err := startCloudWatchLogIndexer(es, consulClient, svc); err != nil {
		return err
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill, syscall.SIGTERM)
	select {
	case sig := <-sigCh:
		golog.Infof("Quitting due to signal %s", sig.String())
		break
	}

	return nil
}
