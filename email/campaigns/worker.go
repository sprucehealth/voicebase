package campaigns

import (
	"net/url"
	"strconv"
	"time"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/mandrill"
	"github.com/sprucehealth/backend/libs/sig"
)

var workerPeriod = time.Minute * 15

type Worker struct {
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
		tc := time.NewTicker(workerPeriod)
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
	for _, c := range CampaignRegistry {
		ci, err := c.Run(w.dataAPI, snap)
		if err != nil {
			w.statCampaignFailed.Inc(1)
			golog.Errorf("Failed to run email campaign %T: %s", c, err)
			continue
		}
		w.statCampaignSucceeded.Inc(1)
		if ci != nil && len(ci.Accounts) != 0 {
			to := make([]int64, 0, len(ci.Accounts))
			for id := range ci.Accounts {
				to = append(to, id)
				ci.Accounts[id] = append(ci.Accounts[id], VarsForAccount(id, c.Key(), w.signer, w.webDomain)...)
			}
			if ci.Msg == nil {
				ci.Msg = &mandrill.Message{}
			}
			res, err := w.emailService.Send(to, c.Key(), ci.Accounts, ci.Msg, email.CanOptOut|email.Async|email.OnlyOnce)
			if err != nil {
				golog.Errorf("Failed to send email for campaign %T (%+v): %s", c, res, err)
			}
			if err := w.dataAPI.EmailRecordSend(to, c.Key()); err != nil {
				golog.Errorf("Failed to record email send for campaign %T: %s", c, err)
			}
		}
	}
	return nil
}

func VarsForAccount(accountID int64, campaignKey string, signer *sig.Signer, webDomain string) []mandrill.Var {
	sig, err := signer.Sign([]byte("optout:" + strconv.FormatInt(accountID, 10)))
	if err != nil {
		golog.Errorf("Failed to generate optout signature: %s", err)
		return nil
	}
	return []mandrill.Var{
		{
			Name: "OptoutURL",
			Content: "https://" + webDomain + "/e/optout?" + url.Values{
				"type": []string{campaignKey},
				"id":   []string{strconv.FormatInt(accountID, 10)},
				"sig":  []string{string(sig)},
			}.Encode(),
		},
	}
}
