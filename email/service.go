package email

import (
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/mandrill"
)

var whitelistDef = &cfg.ValueDef{
	Name:        "Email.Whitelist",
	Description: "New-line separated list of addresses that can receive email",
	Type:        cfg.ValueTypeString,
	Default:     "",
	Multi:       true,
}

// Option represents email related options
type Option int

const (
	// CanOptOut is an Option the indicates that the recipient can have opted out of reciept
	CanOptOut Option = 1 << iota

	// OnlyOnce is an Option for asserting a message is only sent once
	OnlyOnce

	// Async is an Option for performing the email dispatch asynchronously
	Async
)

func (o Option) has(opt Option) bool {
	return o&opt == opt
}

// Mandrill describes methods needed to interact with Mandrill correctly
type Mandrill interface {
	SendMessageTemplate(name string, content []mandrill.Var, msg *mandrill.Message, async bool) ([]*mandrill.SendMessageResponse, error)
}

// Service describes the methods needed to provide email services
type Service interface {
	Send(accountIDs []int64, emailType string, vars map[int64][]mandrill.Var, msg *mandrill.Message, opt Option) ([]*mandrill.SendMessageResponse, error)
}

// OptoutChecker represents a type intended to perform the opt out check before sending
type OptoutChecker struct {
	dataAPI    api.DataAPI
	svc        Mandrill
	cfg        cfg.Store
	wl         atomic.Value // stores a whitelist of emails as []*regexp.Regexp
	wlVal      atomic.Value // stores last whitelist string value for looking for changes
	dispatcher dispatch.Publisher
}

// NewOptoutChecker returns an initialized instance of OptoutChecker
func NewOptoutChecker(dataAPI api.DataAPI, svc Mandrill, cfgStore cfg.Store, dispatcher dispatch.Publisher) *OptoutChecker {
	cfgStore.Register(whitelistDef)
	oc := &OptoutChecker{
		dataAPI:    dataAPI,
		svc:        svc,
		cfg:        cfgStore,
		dispatcher: dispatcher,
	}
	oc.wl.Store([]*regexp.Regexp(nil))
	oc.wlVal.Store("")
	return oc
}

// Send performs the actual sending operation associated with the email distribution
func (oc *OptoutChecker) Send(accountIDs []int64, emailType string, vars map[int64][]mandrill.Var, msg *mandrill.Message, opt Option) ([]*mandrill.SendMessageResponse, error) {
	whitelist := oc.wl.Load().([]*regexp.Regexp)
	if v := oc.cfg.Snapshot().String(whitelistDef.Name); v != oc.wlVal.Load().(string) {
		l := strings.Split(v, "\n")
		whitelist = make([]*regexp.Regexp, 0, len(l))
		for _, s := range l {
			s = strings.TrimSpace(s)
			if s != "" {
				re, err := regexp.Compile(s)
				if err != nil {
					golog.Warningf("Failed to parse email whitelist regex '%s': %s", s, err)
				} else {
					whitelist = append(whitelist, re)
				}
			}
		}
		oc.wl.Store(whitelist)
		oc.wlVal.Store(v)
	}

	var err error
	var rcpt []*api.Recipient
	if opt.has(CanOptOut) {
		rcpt, err = oc.dataAPI.EmailRecipientsWithOptOut(accountIDs, emailType, opt.has(OnlyOnce))
	} else {
		rcpt, err = oc.dataAPI.EmailRecipients(accountIDs)
	}
	if err != nil {
		return nil, err
	}
	if len(rcpt) == 0 {
		return nil, nil
	}
	if msg.MergeLanguage == "" {
		msg.MergeLanguage = "handlebars"
	}
	msg.GlobalMergeVars = append(msg.GlobalMergeVars,
		mandrill.Var{
			Name:    "Env",
			Content: environment.GetCurrent(),
		},
	)
	msg.To = make([]*mandrill.Recipient, 0, len(rcpt))
	for _, r := range rcpt {
		if len(whitelist) != 0 {
			matched := false
			for _, re := range whitelist {
				if re.MatchString(r.Email) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		msg.To = append(msg.To, &mandrill.Recipient{
			Name:  r.Name,
			Email: r.Email,
		})
		if vs := vars[r.AccountID]; vs != nil {
			msg.MergeVars = append(msg.MergeVars,
				mandrill.MergeVar{
					Rcpt: r.Email,
					Vars: vs,
				})
		}
	}
	if msg.ViewContentLink == nil {
		// In production, by default don't store content for emails on Mandrill
		// For other environments log the content for debugging.
		b := !environment.IsProd()
		msg.ViewContentLink = &b
	}
	if msg.PreserveRecipients == nil {
		// By default don't show all recipients in To
		b := false
		msg.PreserveRecipients = &b
	}
	res, err := oc.svc.SendMessageTemplate(emailType, nil, msg, opt.has(Async))
	if err != nil {
		oc.dispatcher.PublishAsync(&SendEvent{
			Recipients: rcpt,
			Type:       emailType,
		})
	}
	return res, err
}
