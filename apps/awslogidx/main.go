package main

/*
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

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/libs/aws/cloudwatchlogs"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/third_party/github.com/armon/consul-api"
)

const (
	eventCount = 0
)

var (
	flagCleanup       = flag.Bool("cleanup", false, "Delete old indexes and exit")
	flagCloudTrail    = flag.Bool("cloudtrail", false, "Enable CloudTrail log indexing")
	flagConsul        = flag.String("consul", "127.0.0.1:8500", "Consul HTTP API host:port")
	flagElasticSearch = flag.String("elasticsearch", "127.0.0.1:9200", "ElasticSearch host:port")
	flagRetainDays    = flag.Int("retaindays", 60, "Number of days of indexes to retain")
	flagVerbose       = flag.Bool("v", false, "Verbose output")
)

func cleanupIndexes(es *ElasticSearch, days int) {
	aliases, err := es.Aliases()
	if err != nil {
		golog.Errorf("Failed to get index aliases: %s", err.Error())
	} else {
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
}

func startPeriodicCleanup(es *ElasticSearch, days int, locker *ConsulLocker) {
	lock := locker.NewLock("service/awslogidx/cleanup", nil)
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
	LastEventTime time.Time
	LastIndexTime time.Time
	NextToken     string
}

func startCloudWatchLogIndexer(es *ElasticSearch, consul *consulapi.Client, locker *ConsulLocker) error {
	// For now this is using a single lock. If the volume of logs to ingest is
	// too high for a single process then his can be modified to use a lock per
	// group or per stream.
	lock := locker.NewLock("service/awslogidx/cwl", nil)

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

			groups, err := cwlClient.DescribeLogGroups("", "", 0)
			if err != nil {
				golog.Errorf("Failed to get log groups: %s", err.Error())
				continue
			}
		groupLoop:
			for _, g := range groups.LogGroups {
				if !lock.Locked() {
					break
				}

				stream, err := cwlClient.DescribeLogStreams(g.LogGroupName, "", "", 0)
				if err != nil {
					golog.Errorf("Failed to get log stream for group %s: %s", g.LogGroupName, err.Error())
					continue
				}
				for _, s := range stream.LogStreams {
					if !lock.Locked() {
						break
					}
					if s.LastEventTimestamp.IsZero() {
						continue
					}
					if indexStream(g.LogGroupName, s, es, consul) {
						break groupLoop
					}
				}
			}
		}
	}()

	return nil
}

func indexStream(groupName string, stream *cloudwatchlogs.LogStream, es *ElasticSearch, consul *consulapi.Client) bool {
	hash := md5.Sum([]byte(fmt.Sprintf("%s|%s", groupName, stream.LogStreamName)))
	key := "service/awslogidx/cwl/" + hex.EncodeToString(hash[:])

	log := golog.Context(
		"group", groupName,
		"stream", stream.LogStreamName,
		"key", key)

	kv, _, err := consul.KV().Get(key, nil)
	if err != nil {
		log.Errorf("Get failed: %s", err.Error())
		return false
	}

	var info streamInfo
	var events *cloudwatchlogs.Events
	var modifyIndex uint64
	if kv != nil {
		if err := json.Unmarshal(kv.Value, &info); err != nil {
			log.Errorf("Unmarshal failed: %s", err.Error())
			return false
		}
		if !stream.LastEventTimestamp.Time.After(info.LastEventTime) {
			log.Debugf("No new events since %s", info.LastEventTime.String())
			return false
		}
		modifyIndex = kv.ModifyIndex
		// The next token is only valid for 24 hours so use the timestamp after that
		if time.Since(info.LastIndexTime) > time.Hour*22 {
			log.Debugf("Fetching by start time of %+v", info.LastEventTime)
			events, err = cwlClient.GetLogEvents(groupName, stream.LogStreamName, false, info.LastEventTime, time.Time{}, "", eventCount)
		} else {
			log.Debugf("Fetching by token")
			events, err = cwlClient.GetLogEvents(groupName, stream.LogStreamName, false, time.Time{}, time.Time{}, info.NextToken, eventCount)
		}
	} else {
		info = streamInfo{
			GroupName:  groupName,
			StreamName: stream.LogStreamName,
		}
		log.Debugf("Fetching from beginning")
		events, err = cwlClient.GetLogEvents(groupName, stream.LogStreamName, true, time.Time{}, time.Time{}, "", eventCount)
	}

	var buf []byte
	for _, e := range events.Events {
		if e.Timestamp.After(info.LastEventTime) {
			info.LastEventTime = e.Timestamp.Time
		}
		h := md5.New()
		t := e.Timestamp.UTC()
		ts := t.Format(time.RFC3339)
		h.Write([]byte(ts))
		h.Write([]byte(e.Message))
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

	info.NextToken = events.NextForwardToken
	info.LastIndexTime = time.Now()

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

	if err := run(); err != nil {
		golog.Fatalf(err.Error())
	}
}

func run() error {
	es := &ElasticSearch{
		Endpoint: "http://" + *flagElasticSearch,
	}
	if *flagCleanup {
		cleanupIndexes(es, *flagRetainDays)
		return nil
	}

	consul, err := consulapi.NewClient(&consulapi.Config{
		Address:    *flagConsul,
		HttpClient: http.DefaultClient,
	})
	if err != nil {
		return err
	}

	locker, err := StartConsulLocker(consul, consulCheckIDPrefix+strconv.Itoa(os.Getpid()))
	if err != nil {
		return err
	}
	defer locker.Stop()

	if *flagCloudTrail {
		if err := startCloudTrailIndexer(es); err != nil {
			return err
		}
	}
	if *flagRetainDays > 0 {
		startPeriodicCleanup(es, *flagRetainDays, locker)
	}

	if err := startCloudWatchLogIndexer(es, consul, locker); err != nil {
		return err
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill)
	select {
	case sig := <-sigCh:
		golog.Infof("Quitting due to signal %s", sig.String())
		break
	}

	return nil
}
