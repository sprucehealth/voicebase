package campaigns

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/mandrill"
)

var abandonedVisitAfterDef = &cfg.ValueDef{
	Name:        "Email.Campaign.AbandonedVisit.After",
	Description: "Age of an open visit after which it's considered abandoned. Set to 0 to disable.",
	Type:        cfg.ValueTypeDuration,
	Default:     time.Hour * 24 * 7,
}

func init() {
	config.MustRegisterCfgDef(abandonedVisitAfterDef)
}

var CampaignRegistry = []Campaign{
	abandonedVisitCampaign{},
}

type Campaign interface {
	// Key returns the unique key for the campaign.
	Key() string
	// Run executes the campaign (e.g. by running queries to list receipients)
	Run(dataAPI api.DataAPI, cfgSnap cfg.Snapshot) (*CampaignInfo, error)
}

type CampaignInfo struct {
	Accounts map[int64][]mandrill.Var
	Msg      *mandrill.Message
}

// AbandonedVisit is sent when a visit is not completed after a
// configured duration.
type abandonedVisitCampaign struct{}

type abandonedVisitCampaignState struct {
	LastTime time.Time `json:"last_time"`
}

func (abandonedVisitCampaign) Key() string {
	return "abandoned-visit"
}

// Run implements the Campaign interface
func (av abandonedVisitCampaign) Run(dataAPI api.DataAPI, cfgSnap cfg.Snapshot) (*CampaignInfo, error) {
	after := cfgSnap.Duration(abandonedVisitAfterDef.Name)
	if after == 0 {
		return nil, nil
	}

	// Fetch all open visits older than After. Remember the last range
	// checked to avoid querying the same visits again.
	var state abandonedVisitCampaignState
	if b, err := dataAPI.EmailCampaignState(av.Key()); err != nil {
		return nil, errors.Trace(fmt.Errorf("email/campaigns: failed to get campaign state for %s: %s", av.Key(), err))
	} else if len(b) != 0 {
		if err := json.Unmarshal(b, &state); err != nil {
			return nil, errors.Trace(fmt.Errorf("email/campaigns: failed to unmarshal campaign state for %s: %s", av.Key(), err))
		}
	}

	startTime := state.LastTime
	endTime := time.Now().Add(-after)
	visits, err := dataAPI.VisitSummaries([]string{common.PVStatusOpen}, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("campaigns: failed to get open visits: %s", err)
	}
	state.LastTime = endTime

	ci := &CampaignInfo{
		Accounts: make(map[int64][]mandrill.Var),
	}
	for _, v := range visits {
		ci.Accounts[v.PatientAccountID] = []mandrill.Var{
			{Name: "CaseName", Content: v.CaseName},
		}
	}

	if b, err := json.Marshal(&state); err != nil {
		golog.Errorf("Failed to marshal campaign state for %s: %s", av.Key(), err)
	} else if err := dataAPI.UpdateEmailCampaignState(av.Key(), b); err != nil {
		golog.Errorf("Failed to update campaign state for %s: %s", av.Key(), err)
	}

	return ci, nil
}
