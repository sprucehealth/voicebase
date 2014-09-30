package doctor_queue

import (
	"errors"
	"math/rand"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/doctor"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/notify"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

var (
	noDoctorFound                          = errors.New("No doctor found to notify")
	timePeriodBetweenNotificationChecks    = time.Minute
	timePeriodBeforeNotifyingSameDoctor    = time.Hour
	timePeriodBetweenNotifyingAtStateLevel = 15 * time.Minute
)

type Worker struct {
	dataAPI                                api.DataAPI
	notificationManager                    *notify.NotificationManager
	lockAPI                                api.LockAPI
	stopChan                               chan bool
	statNotificationCycle                  metrics.Counter
	statNoDoctorsToNotify                  metrics.Counter
	timePeriodBetweenChecks                time.Duration
	timePeriodBetweenNotifyingAtStateLevel time.Duration
	timePeriodBeforeNotifyingSameDoctor    time.Duration
}

func StartWorker(dataAPI api.DataAPI, lockAPI api.LockAPI,
	notificationManager *notify.NotificationManager,
	metricsRegistry metrics.Registry) *Worker {

	statNotificationCycle := metrics.NewCounter()
	statNoDoctorsToNotify := metrics.NewCounter()

	metricsRegistry.Add("cycle", statNotificationCycle)
	metricsRegistry.Add("nodoctors", statNoDoctorsToNotify)

	w := &Worker{
		dataAPI:                                dataAPI,
		notificationManager:                    notificationManager,
		lockAPI:                                lockAPI,
		statNotificationCycle:                  statNotificationCycle,
		statNoDoctorsToNotify:                  statNoDoctorsToNotify,
		stopChan:                               make(chan bool),
		timePeriodBetweenChecks:                timePeriodBetweenNotificationChecks,
		timePeriodBeforeNotifyingSameDoctor:    timePeriodBeforeNotifyingSameDoctor,
		timePeriodBetweenNotifyingAtStateLevel: timePeriodBetweenNotifyingAtStateLevel,
	}
	w.start()
	return w
}

func (w *Worker) start() {
	go func() {
		for {
			defer w.lockAPI.Release()
			if !w.lockAPI.Wait() {
				return
			}

			select {
			case <-w.stopChan:
				return
			default:
			}

			if err := w.notifyDoctorsOfUnclaimedCases(); err != nil {
				golog.Errorf(err.Error())
			}
			w.statNotificationCycle.Inc(1)
			time.Sleep(w.timePeriodBetweenChecks)
		}
	}()
}

func (w *Worker) Stop() {
	close(w.stopChan)
}

func (w *Worker) notifyDoctorsOfUnclaimedCases() error {

	// identify the distinct states in which we currently have unclaimed cases
	careProvidingStateIDs, err := w.dataAPI.CareProvidingStatesWithUnclaimedCases()
	if err != nil {
		return err
	}

	// iterate through the states to notify a doctor per state
	for i, careProvidingStateID := range careProvidingStateIDs {

		doctorToNotify, err := w.pickDoctorRegisteredInState(careProvidingStateID, careProvidingStateIDs[:i])
		if err == noDoctorFound {
			continue
		} else if err != nil {
			return err
		}

		accountID, err := w.dataAPI.GetAccountIDFromDoctorID(doctorToNotify)
		if err != nil {
			return err
		}

		if err := w.notificationManager.NotifyDoctor(api.DOCTOR_ROLE, doctorToNotify, accountID, &doctor.NotifyDoctorOfUnclaimedCaseEvent{
			DoctorID: doctorToNotify,
		}); err != nil {
			return err
		}

		if err := w.dataAPI.RecordDoctorNotifiedOfUnclaimedCases(doctorToNotify); err != nil {
			return err
		}
	}

	return nil
}

func (w *Worker) pickDoctorRegisteredInState(careProvidingStateID int64, statesToAvoid []int64) (int64, error) {

	// only notify at a state level once per 15 minute period
	lastNotifiedTime, err := w.dataAPI.LastNotifiedTimeForCareProvidingState(careProvidingStateID)
	if err != api.NoRowsError && err != nil {
		return 0, err
	} else if err != api.NoRowsError && !lastNotifiedTime.Add(w.timePeriodBetweenNotifyingAtStateLevel).Before(time.Now()) {
		return 0, noDoctorFound
	}

	// don't notify the same doctor within the specified period
	timeThreshold := time.Now().Add(-w.timePeriodBeforeNotifyingSameDoctor)
	for i := len(statesToAvoid); i >= 0; i-- {

		elligibleDoctors, err := w.dataAPI.DoctorsToNotifyInCareProvidingState(careProvidingStateID, statesToAvoid[:i], timeThreshold)
		if err != nil {
			return 0, err
		} else if len(elligibleDoctors) == 0 {
			continue
		}

		// populate all doctors that have never been notified so as to give preference to picking these
		// doctors before we start to pick from doctors that have already been notified
		doctorsNeverNotified := make([]*api.DoctorNotify, 0, len(elligibleDoctors))
		for _, dNotify := range elligibleDoctors {
			if dNotify.LastNotified == nil {
				doctorsNeverNotified = append(doctorsNeverNotified, dNotify)
			}
		}
		if len(doctorsNeverNotified) > 0 {
			return doctorsNeverNotified[rand.Intn(len(doctorsNeverNotified))].DoctorID, nil
		}

		// randomly pick one of the doctors
		return elligibleDoctors[rand.Intn(len(elligibleDoctors))].DoctorID, nil
	}

	w.statNoDoctorsToNotify.Inc(1)
	return 0, noDoctorFound
}
