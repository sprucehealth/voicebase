package doctor_queue

import (
	"math"
	"math/rand"
	"time"

	"github.com/sprucehealth/backend/api"
)

// DoctorNotifyPicker is an interface to provide different
// ways in which to pick a doctor to notify for a paritcular care providing state
type DoctorNotifyPickerConfig struct {
	CareProvidingStateID                   int64
	MinimumTimeBeforeNotifyingForSameState time.Duration
	MinimumTimeBeforeNotifyingSameDoctor   time.Duration
	StatesToAvoid                          []int64
}

type DoctorNotifyPicker interface {
	PickDoctorToNotify(config *DoctorNotifyPickerConfig) (int64, error)
}

// defaultDoctorPicker picks a doctor to notify of a case in a state when:
// a) No doctor has been notified of a case in that state for the specified time period
// b) There is a doctor that either:
// 		b.1) Has never been notified of a case OR
// 		b.2) Has been notified, but not within the minimum time required before notifying the same doctor
// 		WHILE also biasing towards doctors that are not registered in previous states for which a doctor
// 		was just notified
type defaultDoctorPicker struct {
	dataAPI api.DataAPI
	authAPI api.AuthAPI
}

func (d *defaultDoctorPicker) PickDoctorToNotify(config *DoctorNotifyPickerConfig) (int64, error) {

	lastNotifiedTime, err := d.dataAPI.LastNotifiedTimeForCareProvidingState(config.CareProvidingStateID)
	if !api.IsErrNotFound(err) && err != nil {
		return 0, err
	} else if !api.IsErrNotFound(err) &&
		time.Since(lastNotifiedTime) <= config.MinimumTimeBeforeNotifyingForSameState {
		return 0, nil
	}

	// don't notify the same doctor within the specified period
	// and try to pick a doctor that is not in the states for which we just notified doctors
	// while relaxing the constraint if no doctors are found
	timeThreshold := time.Now().Add(-config.MinimumTimeBeforeNotifyingSameDoctor)
	for i := len(config.StatesToAvoid); i >= 0; i-- {

		elligibleDoctors, err := d.dataAPI.DoctorsToNotifyInCareProvidingState(config.CareProvidingStateID,
			config.StatesToAvoid[:i], timeThreshold)
		if err != nil {
			return 0, err
		} else if len(elligibleDoctors) == 0 {
			continue
		}

		var doctorToNotify int64
		doctorsTried := make(map[int64]bool)
		for j := 0; j < 3; j++ {
			potentialDoctorToNotify := elligibleDoctors[rand.Intn(len(elligibleDoctors))].DoctorID
			if doctorsTried[potentialDoctorToNotify] {
				continue
			}

			if withinSnoozePeriod, err := d.isDoctorWithinSnoozePeriod(potentialDoctorToNotify); err != nil {
				return 0, err
			} else if !withinSnoozePeriod {
				doctorToNotify = potentialDoctorToNotify
				break
			}

			doctorsTried[potentialDoctorToNotify] = true
		}

		// no doctor identified that is not within a snooze period
		if doctorToNotify == 0 {
			return 0, nil
		}

		if err := d.dataAPI.RecordCareProvidingStateNotified(config.CareProvidingStateID); err != nil {
			return 0, err
		}

		if err := d.dataAPI.RecordDoctorNotifiedOfUnclaimedCases(doctorToNotify); err != nil {
			return 0, err
		}

		return doctorToNotify, nil
	}

	return 0, noDoctorFound
}

func (d *defaultDoctorPicker) isDoctorWithinSnoozePeriod(doctorID int64) (bool, error) {
	accountID, err := d.dataAPI.GetAccountIDFromDoctorID(doctorID)
	if err != nil {
		return false, err
	}

	// if the doctor has requested notifications to be snoozed in their respective
	// timezones, then ignore the doctor
	tzName, err := d.authAPI.TimezoneForAccount(accountID)
	if api.IsErrNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	location, err := time.LoadLocation(tzName)
	if err != nil {
		return false, err
	}
	timeInDoctorLocation := time.Now().In(location)
	currentHourInDoctorLocation := timeInDoctorLocation.Hour()

	// get snooze configs for doctor
	snoozeConfigs, err := d.dataAPI.SnoozeConfigsForAccount(accountID)
	if err != nil {
		return false, err
	}

	for _, config := range snoozeConfigs {

		startHour := config.StartHour
		endHour := config.StartHour + config.NumHours

		// if the game between the current and end time is > 24, then add
		// 24 to the current time to bring it within the period
		// for instance, if the snooze period is (startHour = 23, numHours = 7)
		// and the current time is 5AM in the doctor's timezone, we need to add 24
		// to the currentHour as Math.abs(5 - 30) > 24. This normalization helps
		// handle snoozer periods that extend beyond midnight into the next day
		currentHour := currentHourInDoctorLocation
		if math.Abs(float64(currentHourInDoctorLocation-endHour)) >= 24 {
			currentHour += 24
		}

		if currentHour >= startHour && currentHour < endHour {
			return true, nil
		}
	}

	return false, nil
}
