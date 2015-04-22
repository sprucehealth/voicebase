package email

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/mandrill"
)

type Option int

const (
	CanOptOut Option = 1 << iota
	OnlyOnce
	Async
)

func (o Option) has(opt Option) bool {
	return o&opt == opt
}

type Mandrill interface {
	SendMessageTemplate(name string, content []mandrill.Var, msg *mandrill.Message, async bool) ([]*mandrill.SendMessageResponse, error)
}

type Service interface {
	Send(accountIDs []int64, emailType string, vars map[int64][]mandrill.Var, msg *mandrill.Message, opt Option) ([]*mandrill.SendMessageResponse, error)
}

type OptoutChecker struct {
	dataAPI    api.DataAPI
	svc        Mandrill
	dispatcher dispatch.Publisher
}

func NewOptoutChecker(dataAPI api.DataAPI, svc Mandrill, dispatcher dispatch.Publisher) *OptoutChecker {
	return &OptoutChecker{
		dataAPI:    dataAPI,
		svc:        svc,
		dispatcher: dispatcher,
	}
}

func (oc *OptoutChecker) Send(accountIDs []int64, emailType string, vars map[int64][]mandrill.Var, msg *mandrill.Message, opt Option) ([]*mandrill.SendMessageResponse, error) {
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
	msg.GlobalMergeVars = append(msg.GlobalMergeVars,
		mandrill.Var{
			Name:    "Env",
			Content: environment.GetCurrent(),
		},
	)
	msg.To = make([]*mandrill.Recipient, len(rcpt))
	for _, r := range rcpt {
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
	res, err := oc.svc.SendMessageTemplate(emailType, nil, msg, opt.has(Async))
	if err != nil {
		oc.dispatcher.PublishAsync(&SendEvent{
			Recipients: rcpt,
			Type:       emailType,
		})
	}
	return res, err
}
