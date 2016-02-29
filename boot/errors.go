package boot

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/ratelimit"
)

func SNSLogHandler(snsCli snsiface.SNSAPI, topic, name string, subHandler golog.Handler, rateLimiter ratelimit.KeyedRateLimiter, metricsRegistry metrics.Registry) golog.Handler {
	jsonFmt := golog.JSONFormatter()
	longFmt := golog.LongFormFormatter()
	statTotal := metrics.NewCounter()
	statFailed := metrics.NewCounter()
	statRateLimited := metrics.NewCounter()
	metricsRegistry.Add("total", statTotal)
	metricsRegistry.Add("failed", statFailed)
	metricsRegistry.Add("ratelimited", statRateLimited)
	return golog.HandlerFunc(func(e *golog.Entry) (err error) {
		if subHandler != nil {
			defer func() {
				err = subHandler.Log(e)
			}()
		}
		if e.Lvl != golog.ERR && e.Lvl != golog.CRIT {
			return nil
		}

		statTotal.Inc(1)

		if rateLimiter != nil {
			key := e.Src
			if key == "" {
				key = e.Msg
			}
			ok, err := rateLimiter.Check(key, 1)
			if err != nil || !ok {
				statRateLimited.Inc(1)
				return nil
			}
		}

		// The Entry shouldn't be used after this function returns so we
		// need to do the formatting before starting the goroutine.
		jsonMsg := string(jsonFmt.Format(e))
		longFmt := string(longFmt.Format(e))
		short := fmt.Sprintf("%s %s %s", e.Lvl.String(), e.Src, e.Msg)
		conc.Go(func() {
			msg, err := json.Marshal(&struct {
				Default string `json:"default"`
				Email   string `json:"email"`
				SMS     string `json:"sms"`
			}{
				Default: jsonMsg,
				Email:   longFmt,
				SMS:     short,
			})
			if err == nil {
				subject := name + " :: " + sanitizeSNSSubject(short)
				if len(subject) > 100 {
					subject = subject[:100]
				}
				_, err = snsCli.Publish(&sns.PublishInput{
					Message:          ptr.String(string(msg)),
					MessageStructure: ptr.String("json"),
					Subject:          &subject,
					TopicArn:         &topic,
				})
			}
			if err != nil && subHandler != nil {
				statFailed.Inc(1)
				// Pass errors publishing to the underlying error handler
				subHandler.Log(&golog.Entry{
					Time: time.Now(),
					Lvl:  golog.ERR,
					Msg:  fmt.Sprintf("Failed to publish to error SNS: %s", err),
					Src:  golog.Caller(0),
				})
			}
		})
		return nil
	})
}

// sanitizeSNSSubject cleans up a string to make it valid for an SNS event subject
func sanitizeSNSSubject(s string) string {
	buf := make([]byte, 0, len(s))
	for _, r := range s {
		if r < 32 || r >= 127 {
			buf = append(buf, ' ')
		} else {
			buf = append(buf, byte(r))
		}
	}
	return string(buf)
}
