package main

/*
TODO:
  - Can set the check status (warn, fail) based on status of AWS api and ElasticSearch so
    that if either fail then this process fails the check and drops leadership.
*/

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/third_party/github.com/armon/consul-api"
)

const (
	consulLeaderKey     = "service/awslogidx/leader"
	consulCheckIDPrefix = "awslogidx-"
	consulCheckName     = "Liveness check for awslogidx process"
	consulCheckTTL      = "60s"
	// consulLockDelay is the time after a lock is release before it can be acquired
	consulLockDelay = time.Second * 30
)

var (
	flagCleanup       = flag.Bool("cleanup", false, "Delete old indexes and exit")
	flagCloudTrail    = flag.Bool("cloudtrail", false, "Enable CloudTrail log indexing")
	flagConsul        = flag.String("consul", "127.0.0.1:8500", "Consul HTTP API host:port")
	flagElasticSearch = flag.String("elasticsearch", "127.0.0.1:9200", "ElasticSearch host:port")
	flagRetainDays    = flag.Int("retaindays", 60, "Number of days of indexes to retain")
)

var leader int32

func isLeader() bool {
	return atomic.LoadInt32(&leader) != 0
}

func setLeader(b bool) {
	if b {
		atomic.StoreInt32(&leader, 1)
	} else {
		atomic.StoreInt32(&leader, 0)
	}
}

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

func startPeriodicCleanup(es *ElasticSearch, days int) {
	go func() {
		for {
			for !isLeader() {
				time.Sleep(time.Minute * 10)
			}
			cleanupIndexes(es, days)
			time.Sleep(time.Hour * 24)
		}
	}()
}

// powerStruggle loops trying to become leader. It never stops except
// when the process quits.
func startPowerStruggle(consul *consulapi.Client, sessionID string, stopCh chan bool) {
	go func() {
		for {
			select {
			case <-stopCh:
			default:
			}
			if leader, _, err := consul.KV().Acquire(&consulapi.KVPair{
				Key:     consulLeaderKey,
				Value:   []byte(sessionID),
				Session: sessionID,
			}, nil); err != nil {
				golog.Errorf("Error acquiring lock: %s", err.Error())
			} else {
				setLeader(leader)
				if leader {
					golog.Infof("Became leader")
				} else {
					golog.Infof("Not leader")
				}

				var lastIndex uint64
			leaderCheck:
				for {
					kv, meta, err := consul.KV().Get(consulLeaderKey, &consulapi.QueryOptions{
						WaitIndex: lastIndex,
					})
					if err != nil {
						// Assume we're not the leader for now since it's safer.
						setLeader(false)
						golog.Errorf("Failed to get leader key (dropping leadership): %s", err.Error())
						time.Sleep(time.Second * 10)
						lastIndex = 0
						continue
					}
					lastIndex = meta.LastIndex
					switch kv.Session {
					case "":
						golog.Infof("No leader. Attempting to take power after %s", time.Duration(consulLockDelay).String())
						setLeader(false)
						break leaderCheck
					case sessionID:
						if !isLeader() {
							// This should only happen if there was previously an error
							// talking to consul.
							setLeader(true)
							golog.Warningf("Remembering own leadership")
						}
						continue
					}
					if isLeader() {
						setLeader(false)
						golog.Warningf("Lost leadership to %s", kv.Session)
					} else {
						golog.Infof("Current leader is %s", kv.Session)
					}
				}
			}
			// After the lock is released there's a period of time before which
			// it can be acquired. This allows for a process that involuntarily
			// lost the lock to notice they lost the lock and complete processing
			// before another process becomes leader.
			time.Sleep(consulLockDelay)
		}
	}()
}

func main() {
	flag.Parse()

	if *flagCloudTrail {
		if err := setupAWS(); err != nil {
			golog.Fatalf(err.Error())
		}
	}

	es := &ElasticSearch{
		Endpoint: "http://" + *flagElasticSearch,
	}

	consul, err := consulapi.NewClient(&consulapi.Config{
		Address:    *flagConsul,
		HttpClient: http.DefaultClient,
	})
	if err != nil {
		golog.Fatalf(err.Error())
	}

	check, err := startConsulCheck(consul, consulCheckIDPrefix+strconv.Itoa(os.Getpid()))
	if err != nil {
		golog.Fatalf(err.Error())
	}
	defer func() {
		if err := check.stop(); err != nil {
			golog.Fatalf(err.Error())
		}
	}()

	sessionID, _, err := consul.Session().Create(&consulapi.SessionEntry{
		LockDelay: consulLockDelay,
		Checks: []string{
			"serfHealth", // Default health check for consul process liveliness
			check.id,
		},
	}, nil)
	if err != nil {
		golog.Fatalf("Failed to create consul session: %s", err.Error())
	}
	defer func() {
		if _, err := consul.Session().Destroy(sessionID, nil); err != nil {
			golog.Errorf("Failed to destroy consul session: %s", err.Error())
		}
	}()

	stopCh := make(chan bool, 1)
	startPowerStruggle(consul, sessionID, stopCh)

	if *flagCleanup {
		cleanupIndexes(es, *flagRetainDays)
		return
	}
	if *flagRetainDays > 0 {
		startPeriodicCleanup(es, *flagRetainDays)
	}

	if *flagCloudTrail {
		if err := startCloudTrailIndexer(es); err != nil {
			golog.Fatalf(err.Error())
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill)
	select {
	case sig := <-sigCh:
		golog.Infof("Quitting due to signal %s", sig.String())
		break
	}
	stopCh <- true
}
