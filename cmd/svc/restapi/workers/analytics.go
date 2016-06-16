package workers

import (
	"fmt"
	"time"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	maxAnalyticsItems = 3
)

func StartAnalyticsWorker(dataAPI api.DataAPI, metricsRegistry metrics.Registry) {
	statOldestPVs := make([]*metrics.IntegerGauge, maxAnalyticsItems)
	stateOldestUnclaimedCases := make([]*metrics.IntegerGauge, maxAnalyticsItems)
	statOldestTPs := make([]*metrics.IntegerGauge, maxAnalyticsItems)

	for i := 0; i < maxAnalyticsItems; i++ {
		statOldestPVs[i] = metrics.NewIntegerGauge()
		statOldestTPs[i] = metrics.NewIntegerGauge()
		stateOldestUnclaimedCases[i] = metrics.NewIntegerGauge()
		metricsRegistry.Add(fmt.Sprintf("oldest/visit/%d", i), statOldestPVs[i])
		metricsRegistry.Add(fmt.Sprintf("oldest/treatment_plan/%d", i), statOldestTPs[i])
		metricsRegistry.Add(fmt.Sprintf("oldest/unclaimed_case/%d", i), stateOldestUnclaimedCases[i])
	}

	go func() {
		for {
			// get oldest visits
			patientVisitAges, err := dataAPI.GetOldestVisitsInStatuses(maxAnalyticsItems,
				[]string{common.PVStatusSubmitted,
					common.PVStatusReviewing,
					common.PVStatusCharged,
					common.PVStatusRouted})
			if err != nil {
				golog.Errorf("Unable to get the oldest patient visits: %s", err)
			}
			for i, visitAge := range patientVisitAges {
				statOldestPVs[i].Set(int64(visitAge.Age.Seconds()))
			}
			for i := len(patientVisitAges); i < len(statOldestPVs); i++ {
				statOldestPVs[i].Set(0)
			}

			caseAges, err := dataAPI.OldestUnclaimedItems(maxAnalyticsItems)
			if err != nil {
				golog.Errorf("Unable to get the oldest cases: %s", err)
			}
			for i, caseAge := range caseAges {
				stateOldestUnclaimedCases[i].Set(int64(caseAge.Age.Seconds()))
			}
			for i := len(caseAges); i < len(stateOldestUnclaimedCases); i++ {
				stateOldestUnclaimedCases[i].Set(0)
			}

			tpAges, err := dataAPI.GetOldestTreatmentPlanInStatuses(maxAnalyticsItems,
				[]common.TreatmentPlanStatus{
					common.TPStatusSubmitted,
					common.TPStatusRXStarted})
			if err != nil {
				golog.Errorf("Unable to get the oldest treatment plans: %s", err)
			}

			for i, tpAge := range tpAges {
				statOldestTPs[i].Set(int64(tpAge.Age / time.Second))
			}
			for i := len(tpAges); i < len(statOldestTPs); i++ {
				statOldestTPs[i].Set(0)
			}

			time.Sleep(time.Minute)
		}
	}()
}
