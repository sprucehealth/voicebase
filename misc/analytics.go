package misc

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

func StartWorker(dataAPI api.DataAPI, metricsRegistry metrics.Registry) {

	statOldestPVs := []metrics.Histogram{metrics.NewBiasedHistogram(), metrics.NewBiasedHistogram(), metrics.NewBiasedHistogram()}
	statOldestTPs := []metrics.Histogram{metrics.NewBiasedHistogram(), metrics.NewBiasedHistogram(), metrics.NewBiasedHistogram()}

	for i, statPV := range statOldestPVs {
		metricsRegistry.Add(fmt.Sprintf("oldest/visit/%d", i), statPV)
	}

	for i, statTP := range statOldestTPs {
		metricsRegistry.Add(fmt.Sprintf("oldest/treatment_plan/%d", i), statTP)
	}

	go func() {
		for {

			// get oldest visits
			patientVisitAges, err := dataAPI.GetOldestVisitsInStatuses(3,
				[]string{common.PVStatusSubmitted,
					common.PVStatusReviewing,
					common.PVStatusCharged,
					common.PVStatusRouted})
			if err != nil {
				golog.Errorf("Unable to get the oldest patient visits: %s", err)
			}

			for i, visitAge := range patientVisitAges {
				statOldestPVs[i].Update(int64(visitAge.Age / time.Second))
			}

			tpAges, err := dataAPI.GetOldestTreatmentPlanInStatuses(3,
				[]common.TreatmentPlanStatus{
					common.TPStatusSubmitted,
					common.TPStatusRXStarted})
			if err != nil {
				golog.Errorf("Unable to get the oldest treatment plans: %s", err)
			}

			for i, tpAge := range tpAges {
				statOldestTPs[i].Update(int64(tpAge.Age / time.Second))
			}

		}
	}()

}
