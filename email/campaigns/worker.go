package campaigns

import (
	"net/url"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/mandrill"
	"github.com/sprucehealth/backend/libs/sig"
)

type Worker struct {
	campaigns             []campaign
	cfgStore              cfg.Store
	dataAPI               api.DataAPI
	emailService          email.Service
	lock                  api.LockAPI
	signer                *sig.Signer
	stopCh                chan struct{}
	webDomain             string
	statCampaignSucceeded *metrics.Counter
	statCampaignFailed    *metrics.Counter
}

func NewWorker(dataAPI api.DataAPI, emailService email.Service, webDomain string, signer *sig.Signer, cfgStore cfg.Store, lock api.LockAPI, metricsRegistry metrics.Registry) *Worker {
	w := &Worker{
		campaigns: []campaign{
			newAbandonedVisitCampaign(dataAPI, cfgStore),
		},
		cfgStore:              cfgStore,
		dataAPI:               dataAPI,
		emailService:          emailService,
		lock:                  lock,
		signer:                signer,
		stopCh:                make(chan struct{}),
		webDomain:             webDomain,
		statCampaignSucceeded: metrics.NewCounter(),
		statCampaignFailed:    metrics.NewCounter(),
	}
	metricsRegistry.Add("campaign/succeeded", w.statCampaignSucceeded)
	metricsRegistry.Add("campaign/failed", w.statCampaignFailed)
	return w
}

func (w *Worker) Start() {
	go func() {
		defer w.lock.Release()
		tc := time.NewTicker(time.Hour)
		for {
			if !w.lock.Wait() {
				return
			}

			select {
			case <-w.stopCh:
				return
			case <-tc.C:
				if w.lock.Locked() {
					if err := w.Do(); err != nil {
						golog.Errorf(err.Error())
					}
				}
			}
		}
	}()
}

func (w *Worker) Stop() {
	close(w.stopCh)
}

func (w *Worker) Do() error {
	snap := w.cfgStore.Snapshot()
	for _, c := range w.campaigns {
		ci, err := c.Run(snap)
		if err != nil {
			w.statCampaignFailed.Inc(1)
			golog.Errorf("Failed to run email campaign %T: %s", c, err)
			continue
		}
		w.statCampaignSucceeded.Inc(1)
		if ci != nil && len(ci.accounts) != 0 {
			to := make([]int64, 0, len(ci.accounts))
			for id := range ci.accounts {
				to = append(to, id)
				sig, err := w.signer.Sign([]byte("optout:" + strconv.FormatInt(id, 10)))
				if err != nil {
					golog.Errorf("Failed to generate optout signature: %s", err)
				} else {
					ci.accounts[id] = append(ci.accounts[id],
						mandrill.Var{
							Name: "OptoutURL",
							Content: "https://" + w.webDomain + "/e/optout?" + url.Values{
								"type": []string{ci.emailType},
								"id":   []string{strconv.FormatInt(id, 10)},
								"sig":  []string{string(sig)},
							}.Encode(),
						},
					)
				}
			}
			if ci.msg == nil {
				ci.msg = &mandrill.Message{}
			}
			res, err := w.emailService.Send(to, ci.emailType, ci.accounts, ci.msg, email.CanOptOut|email.Async|email.OnlyOnce)
			if err != nil {
				golog.Errorf("Failed to send email for campaign %T (%+v): %s", c, res, err)
			}
			if err := w.dataAPI.EmailRecordSend(to, ci.emailType); err != nil {
				golog.Errorf("Failed to record email send for campaign %T: %s", c, err)
			}
		}
	}
	return nil
}

// Reset is used by tests to reset the campaigns
func (w *Worker) Reset() {
	for _, c := range w.campaigns {
		c.Reset()
	}
}
