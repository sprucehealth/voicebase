package campaigns

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/mandrill"
)

var abandonedVisitAfterDef = &cfg.ValueDef{
	Name:        "Email.Campaign.AbandonedVisit.After",
	Description: "Time after which an open visit is considered abandoned. Set to 0 to disable.",
	Type:        cfg.ValueTypeDuration,
	Default:     time.Hour * 24 * 7,
}

type campaign interface {
	// Run executes the campaign (e.g. by running queries to list receipients)
	Run(cfgSnap cfg.Snapshot) (*campaignInfo, error)
	// Reset is used in tests to get to an initial state
	Reset()
}

type campaignInfo struct {
	emailType string
	accounts  map[int64][]mandrill.Var
	msg       *mandrill.Message
}

// AbandonedVisit is sent when a visit is not completed after a
// configured duration.
type abandonedVisitCampaign struct {
	dataAPI api.DataAPI

	lastEndTime time.Time
}

func newAbandonedVisitCampaign(dataAPI api.DataAPI, cfgStore cfg.Store) *abandonedVisitCampaign {
	cfgStore.Register(abandonedVisitAfterDef)
	return &abandonedVisitCampaign{
		dataAPI: dataAPI,
	}
}

// Run implements the Campaign interface
func (av *abandonedVisitCampaign) Run(cfgSnap cfg.Snapshot) (*campaignInfo, error) {
	after := cfgSnap.Duration(abandonedVisitAfterDef.Name)
	if after == 0 {
		return nil, nil
	}

	// Fetch all open visits older than After. Remember the last range
	// checked to avoid querying the same visits again.
	startTime := av.lastEndTime
	endTime := time.Now().Add(-after)
	visits, err := av.dataAPI.VisitSummaries([]string{common.PVStatusOpen}, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("campaigns: failed to get open visits: %s", err)
	}
	av.lastEndTime = endTime

	ci := &campaignInfo{
		emailType: "abandoned-visit",
		accounts:  make(map[int64][]mandrill.Var),
	}
	for _, v := range visits {
		ci.accounts[v.PatientAccountID] = []mandrill.Var{
			{Name: "CaseName", Content: v.CaseName},
		}
	}
	return ci, err
}

// Run implements the Campaign interface
func (av *abandonedVisitCampaign) Reset() {
	av.lastEndTime = time.Time{}
}
