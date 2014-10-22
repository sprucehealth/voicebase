package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/analytics"
	_ "github.com/sprucehealth/backend/third_party/github.com/lib/pq"
)

var (
	reDocID     = regexp.MustCompile(`doctor (\d+)`)
	reTimestamp = regexp.MustCompile(`^t=([^\s]+)`)
	reJBCQ      = map[string]*regexp.Regexp{
		"jbcq_claim_extend": regexp.MustCompile(`JBCQ: Claim extended for doctor (?P<doctor_id>\d+) on case (?P<case_id>\d+) with treatment plan (?P<treatment_plan_id>\d+)`),
		"jbcq_claim_revoke": regexp.MustCompile(`JBCQ: Revoking access for case (?P<case_id>\d+) from doctor (?P<doctor_id>\d+). Expiration time: (?P<expire_time>[^\.]+).`),
		"jbcq_temp_assign":  regexp.MustCompile(`JBCQ: Temporarily assigned case (?P<case_id>\d+) to doctor (?P<doctor_id>\d+)`),
		"jbcq_perm_assign":  regexp.MustCompile(`JBCQ: Permanently assigned case (?P<case_id>\d+) to doctor (?P<doctor_id>\d+)`),
	}
)

type appConfig struct {
	Verbose bool
	// Analytics database
	DBHost    string
	DBPort    int
	DBName    string
	DBUser    string
	DBPass    string
	DBSSLMode string
}

var config = &appConfig{}

type doctor struct {
	ID       int64
	Duration time.Duration
	LastTime time.Time
}

type esResponse struct {
	Hits struct {
		Total int `json:"total"`
		Hits  []struct {
			Source struct {
				Timestamp time.Time `json:"@timestamp"`
				Msg       string    `json:"msg"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func init() {
	flag.StringVar(&config.DBHost, "db.host", "", "Database hostname")
	flag.IntVar(&config.DBPort, "db.port", 5439, "Database port")
	flag.StringVar(&config.DBName, "db.name", "", "Database name")
	flag.StringVar(&config.DBUser, "db.user", "", "Database username")
	flag.StringVar(&config.DBPass, "db.pass", "", "Database password")
	flag.StringVar(&config.DBSSLMode, "db.sslmode", "require", "disable, require, or verify-full")
	flag.BoolVar(&config.Verbose, "v", false, "Verbose output")
}

func fetchDoctorNames(db *sql.DB) (map[int64]string, error) {
	rows, err := db.Query(`SELECT id, first_name || ' ' || last_name FROM doctor`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names := make(map[int64]string)
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		names[id] = name
	}
	return names, rows.Err()
}

func jbcqEvents() (*esResponse, error) {
	body := strings.NewReader(`{
  "query": {
    "filtered": {
      "query": {
        "bool": {
          "should": [
            {
              "query_string": {
                "query": "JBCQ"
              }
            }
          ]
        }
      },
      "filter": {
        "bool": {
          "must": [
            {
              "fquery": {
                "query": {
                  "query_string": {
                    "query": "_type:(log)"
                  }
                },
                "_cache": true
              }
            },
            {
              "fquery": {
                "query": {
                  "query_string": {
                    "query": "group:(\"/var/log/restapi.log\")"
                  }
                },
                "_cache": true
              }
            }
          ],
          "must_not": [
            {
              "fquery": {
                "query": {
                  "query_string": {
                    "query": "msg:(apirequest)"
                  }
                },
                "_cache": true
              }
            },
            {
              "fquery": {
                "query": {
                  "query_string": {
                    "query": "msg:(webrequest)"
                  }
                },
                "_cache": true
              }
            },
            {
              "fquery": {
                "query": {
                  "query_string": {
                    "query": "msg:(\"msg=audit\")"
                  }
                },
                "_cache": true
              }
            }
          ]
        }
      }
    }
  },
  "size": 2000,
  "sort": [
    {
      "@timestamp": {
        "order": "desc",
        "ignore_unmapped": true
      }
    }
  ]
}`)

	startDate := time.Date(2014, time.Month(9), 28, 0, 0, 0, 0, time.Local).UTC()
	endDate := time.Now().UTC()
	var dates []string
	for d := startDate; d.Before(endDate); d = d.Add(time.Hour * 24 * 7) {
		dates = append(dates, d.Format("log-2006.01.02"))
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:9200/%s/_search?pretty", strings.Join(dates, ",")), body)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Unexpected status code %d: %s", res.StatusCode, res.Status)
	}

	var er esResponse

	if err := json.NewDecoder(res.Body).Decode(&er); err != nil {
		return nil, err
	}

	// b, err := ioutil.ReadAll(res.Body)
	// if err != nil {
	// 	return nil, err
	// }
	// fmt.Printf("%s\n", string(b)[:1000])

	return &er, nil
}

func main() {
	log.SetFlags(log.Lshortfile)
	flag.Parse()

	dbArgs := fmt.Sprintf("host=%s port=%d dbname=%s sslmode=%s", config.DBHost, config.DBPort, config.DBName, config.DBSSLMode)
	if config.DBUser != "" {
		dbArgs += " user=" + config.DBUser
	}
	if config.DBPass != "" {
		dbArgs += " password=" + config.DBPass
	}
	db, err := sql.Open("postgres", dbArgs)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	activityTime := time.Second * 60

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	// Parse JBCQ events from REST API log

	log.Printf("Fetching JBCQ events...")
	jbcq, err := jbcqEvents()
	if err != nil {
		log.Fatal(err)
	}

	jbcqEv := make([]*analytics.ServerEvent, 0, len(jbcq.Hits.Hits))
	for _, h := range jbcq.Hits.Hits {
		found := false
		for event, re := range reJBCQ {
			m := re.FindStringSubmatch(h.Source.Msg)
			if len(m) != 0 {
				found = true

				// Parse log line into ServerEvent struct

				tms := reTimestamp.FindStringSubmatch(h.Source.Msg)[1]
				tm, err := time.Parse("2006-01-02T15:04:05-0700", tms)
				if err != nil {
					log.Fatalf("Failed to parse timestamp '%s': %s", tms, err.Error())
				}
				if tm.Sub(h.Source.Timestamp) != 0 {
					log.Fatalf("Timestamps don't match: %s != %s", tm.String(), h.Source.Timestamp)
				}

				ev := &analytics.ServerEvent{
					Timestamp: analytics.Time(h.Source.Timestamp),
				}
				for i, g := range re.SubexpNames() {
					var err error
					switch g {
					default:
						log.Fatalf("Unsupported group %s value '%s'", g, m[i])
					case "":
						if i != 0 {
							log.Fatalf("Unnamed group #%d for jbcq event %s", i, event)
						}
					case "patient_id":
						ev.PatientID, err = strconv.ParseInt(m[i], 10, 64)
					case "doctor_id":
						ev.DoctorID, err = strconv.ParseInt(m[i], 10, 64)
					case "case_id":
						ev.CaseID, err = strconv.ParseInt(m[i], 10, 64)
					case "treatment_plan_id":
						ev.TreatmentPlanID, err = strconv.ParseInt(m[i], 10, 64)
					case "expire_time":
						et, err := time.Parse("2006-01-02 15:04:05 -0700 MST", m[i])
						if err != nil {
							log.Fatalf("Failed to parse group %s value '%s': %s", g, m[i], err.Error())
						}
						b, err := json.Marshal(&struct {
							ExpireTime time.Time `json:"expire_time"`
						}{
							ExpireTime: et,
						})
						if err != nil {
							log.Fatalf("Failed to encode json: %s", err.Error())
						}
						ev.ExtraJSON = string(b)
					}
					if err != nil {
						log.Fatalf("Failed to parse group %s value '%s': %s", g, m[i], err.Error())
					}
				}
				jbcqEv = append(jbcqEv, ev)
				break
			}
		}
		if !found {
			log.Fatalf("Failed to parse '%s'", h.Source.Msg)
		}
	}

	log.Printf("Insert JBCQ events into temp table...")

	inserts := make([]string, 0, len(jbcqEv))
	values := make([]interface{}, 0, len(jbcqEv)*2)
	for _, ev := range jbcqEv {
		if ev.Event == "jbcq_claim_revoke" {
			continue
		}

		inserts = append(inserts,
			fmt.Sprintf("($%d, $%d)", len(inserts)*2+1, len(inserts)*2+2),
		)
		values = append(values,
			ev.Time(),
			ev.DoctorID,
		)
	}

	if _, err := tx.Exec(`
		CREATE TEMP TABLE jbcq (
			tstamp TIMESTAMP NOT NULL,
			doctor_id INT8 NOT NULL
		)`,
	); err != nil {
		log.Fatal(err)
	}
	if _, err := tx.Exec(`INSERT INTO jbcq (tstamp, doctor_id) VALUES `+strings.Join(inserts, ","), values...); err != nil {
		log.Fatal(err)
	}

	log.Printf("Running...")

	startDate := time.Date(2014, time.Month(9), 28, 0, 0, 0, 0, time.Local).UTC()
	endDate := time.Now().UTC()

	names, err := fetchDoctorNames(db)
	if err != nil {
		log.Fatal(err)
	}
	doctorIDs := make([]int64, 0, len(names))
	for id := range names {
		doctorIDs = append(doctorIDs, id)
	}
	var dates []time.Time
	durations := make(map[int64][]int)
	nonZero := make(map[int64]bool)

	for date := startDate; date.Before(endDate); date = date.Add(time.Duration(time.Hour * 24 * 7)) {
		rows, err := tx.Query(`
			SELECT *
			FROM (
					SELECT "time" AS tstamp, doctor_id
					FROM server_event
					WHERE event IN (
						'visit_opened', 'diagnosis_modified', 'visit_marked_unsuitable',
						'treatment_plan_submitted', 'treatment_plan_activated', 'treatment_plan_started'
					)
						AND "time" >= $1 AND "time" < $1 + INTERVAL '7 day'
				UNION
					SELECT tstamp, role_id AS doctor_id
					FROM patient_case_message
					INNER JOIN person ON person.id = person_id
					INNER JOIN role_type ON role_type.id = role_type_id
					WHERE role_type_tag IN ('DOCTOR', 'MA')
						AND tstamp >= $1 AND tstamp < $1 + INTERVAL '7 day'
				UNION
					SELECT treatment.creation_date AS tstamp, treatment_plan.doctor_id
					FROM treatment
					INNER JOIN treatment_plan ON treatment_plan.id = treatment.treatment_plan_id
					WHERE treatment.creation_date >= $1 AND treatment.creation_date < $1 + INTERVAL '7 day'
				UNION
					SELECT tstamp, doctor_id
					FROM jbcq
					WHERE tstamp >= $1 AND tstamp < $1 + INTERVAL '7 day'
			)
			ORDER BY tstamp`, date)
		if err != nil {
			log.Fatal(err)
		}
		events := 0
		doctors := make(map[int64]*doctor)
		for rows.Next() {
			events++
			var tm time.Time
			var doctorID int64
			if err := rows.Scan(&tm, &doctorID); err != nil {
				log.Fatal(err)
			}

			dr := doctors[doctorID]
			if dr == nil {
				dr = &doctor{ID: doctorID}
				doctors[doctorID] = dr
			}
			startTime := tm
			endTime := tm.Add(activityTime)
			if !dr.LastTime.IsZero() && dr.LastTime.After(startTime) {
				startTime = dr.LastTime
			}
			if endTime.After(startTime) {
				dr.Duration += endTime.Sub(startTime)
				nonZero[doctorID] = true
			}
			dr.LastTime = endTime
		}
		if err := rows.Err(); err != nil {
			log.Fatal(err)
		}
		rows.Close()

		dates = append(dates, date)
		for _, id := range doctorIDs {
			sec := 0
			if dr := doctors[id]; dr != nil {
				sec = int(dr.Duration.Seconds())
			}
			durations[id] = append(durations[id], sec)
		}
	}
	fmt.Print("")
	for _, d := range dates {
		fmt.Printf("\t%s", d.Format("Mon Jan _2"))
	}
	fmt.Println()
	for _, id := range doctorIDs {
		if !nonZero[id] {
			continue
		}
		fmt.Print(names[id])
		for i := range dates {
			fmt.Printf("\t%d", durations[id][i])
		}
		fmt.Println()
	}
}
