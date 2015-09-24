package common

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/sprucehealth/backend/libs/errors"
)

var rxrTimePattern = regexp.MustCompile(`^\d{2}:\d{2}$`)

// RXRTime represents a time value supplied to a rx reminders
type RXRTime string

// ParseRXRTime returns the ParseRXRTime that maps to the provided string
func ParseRXRTime(s string) (RXRTime, error) {
	if rxrTimePattern.Match([]byte(s)) {
		return RXRTime(s), nil
	}
	return RXRTime(""), fmt.Errorf(`%s is not a valid RXRTime of format ^\d{2}:\d{2}$`, s)
}

func (r RXRTime) String() string {
	return string(r)
}

// RXRInterval represents the lifecycle of a rx_reminder
type RXRInterval string

const (
	// RXRIntervalEveryDay represents the interval in which a reminder is daily
	RXRIntervalEveryDay RXRInterval = "EVERY_DAY"

	// RXRIntervalEveryOtherDay represents the interval in which a reminder is every other day
	RXRIntervalEveryOtherDay RXRInterval = "EVERY_OTHER_DAY"

	// RXRIntervalCustom represents the interval in which a reminder is a custom configuration
	RXRIntervalCustom RXRInterval = "CUSTOM"
)

// ParseRXRInterval returns the RXRInterval that maps to the provided string
func ParseRXRInterval(s string) (RXRInterval, error) {
	switch rs := RXRInterval(strings.ToUpper(s)); rs {
	case RXRIntervalEveryDay, RXRIntervalEveryOtherDay, RXRIntervalCustom:
		return rs, nil
	}

	return RXRInterval(""), fmt.Errorf("%s is not a RXRInterval", s)
}

// Scan allows for RXRInterval to be utilized in database queries and conforms the sql.Scanner interface
func (r *RXRInterval) Scan(src interface{}) error {

	str, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("scan: Cannot scan type %T into RXRInterval when string expected", src)
	}

	var err error
	*r, err = ParseRXRInterval(string(str))

	return err
}

func (r RXRInterval) String() string {
	return string(r)
}

// RXRDay represents valid
type RXRDay string

const (
	// RXRDaySunday represents the interval in which a reminder is on a Sunday
	RXRDaySunday RXRDay = "SUNDAY"

	// RXRDayMonday represents the interval in which a reminder is on a Monday
	RXRDayMonday RXRDay = "MONDAY"

	// RXRDayTuesday represents the interval in which a reminder is on a Tuesday
	RXRDayTuesday RXRDay = "TUESDAY"

	// RXRDayWednesday represents the interval in which a reminder is on a Wednesday
	RXRDayWednesday RXRDay = "WEDNESDAY"

	// RXRDayThursday represents the interval in which a reminder is on a Thursday
	RXRDayThursday RXRDay = "THURSDAY"

	// RXRDayFriday represents the interval in which a reminder is on a Friday
	RXRDayFriday RXRDay = "FRIDAY"

	// RXRDaySaturday represents the interval in which a reminder is on a Saturday
	RXRDaySaturday RXRDay = "SATURDAY"
)

// ParseRXRDay returns the RXRDay that maps to the provided string
func ParseRXRDay(s string) (RXRDay, error) {
	switch rs := RXRDay(strings.ToUpper(s)); rs {
	case RXRDaySunday, RXRDayMonday, RXRDayTuesday, RXRDayWednesday, RXRDayThursday, RXRDayFriday, RXRDaySaturday:
		return rs, nil
	}

	return RXRDay(""), fmt.Errorf("%s is not a RXRDay", s)
}

func (r RXRDay) String() string {
	return string(r)
}

const (
	rxrDaySep = `,`
)

// SplitRXRDayString splits the provided string using the expected seperator into a slice of type RXRDay
func SplitRXRDayString(s string) ([]RXRDay, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	days := strings.Split(s, rxrDaySep)
	rxrDays := make([]RXRDay, len(days))
	for i, d := range days {
		rxrDay, err := ParseRXRDay(d)
		if err != nil {
			return nil, errors.Trace(err)
		}
		rxrDays[i] = rxrDay
	}
	return rxrDays, nil
}

// JoinRXRDaySlice joins the provided slice using the expected seperator intro a string for storage
func JoinRXRDaySlice(rxrDays []RXRDay) string {
	if len(rxrDays) == 0 {
		return ""
	} else if len(rxrDays) == 1 {
		return rxrDays[0].String()
	}
	args := make([]interface{}, len(rxrDays))
	for i, v := range rxrDays {
		args[i] = v
	}
	return fmt.Sprintf(strings.Repeat("%s"+rxrDaySep, len(rxrDays)-1)+"%s", args...)
}

// RXRDaySlice aliases a slice of RXRDays
type RXRDaySlice []RXRDay

// Strings returns the list of RXRDays as a list of the appropriate strings
func (s RXRDaySlice) Strings() []string {
	ss := make([]string, len(s))
	for i, d := range []RXRDay(s) {
		ss[i] = d.String()
	}
	return ss
}

// RXReminder represents the data layer representation of a rx_reminder
type RXReminder struct {
	TreatmentID  int64
	ReminderText string
	Interval     RXRInterval
	Days         []RXRDay
	Times        string
	CreationDate time.Time
}
