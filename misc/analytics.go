package misc

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

const (
	maxItems = 3
)

func StartWorker(dataAPI api.DataAPI, metricsRegistry metrics.Registry) {

	statOldestPVs := make([]metrics.IntegerGauge, maxItems)
	statOldestTPs := make([]metrics.IntegerGauge, maxItems)

	for i := 0; i < maxItems; i++ {
		statOldestPVs[i] = metrics.NewIntegerGauge()
		statOldestTPs[i] = metrics.NewIntegerGauge()
		metricsRegistry.Add(fmt.Sprintf("oldest/visit/%d", i), statOldestPVs[i])
		metricsRegistry.Add(fmt.Sprintf("oldest/treatment_plan/%d", i), statOldestTPs[i])
	}

	go func() {
		for {
			// get oldest visits
			patientVisitAges, err := dataAPI.GetOldestVisitsInStatuses(maxItems,
				[]string{common.PVStatusSubmitted,
					common.PVStatusReviewing,
					common.PVStatusCharged,
					common.PVStatusRouted})
			if err != nil {
				golog.Errorf("Unable to get the oldest patient visits: %s", err)
			}

			for i, visitAge := range patientVisitAges {
				statOldestPVs[i].Set(int64(visitAge.Age / time.Second))
			}

			tpAges, err := dataAPI.GetOldestTreatmentPlanInStatuses(maxItems,
				[]common.TreatmentPlanStatus{
					common.TPStatusSubmitted,
					common.TPStatusRXStarted})
			if err != nil {
				golog.Errorf("Unable to get the oldest treatment plans: %s", err)
			}

			for i, tpAge := range tpAges {
				statOldestTPs[i].Set(int64(tpAge.Age / time.Second))
			}

			time.Sleep(time.Minute)
		}
	}()

}
