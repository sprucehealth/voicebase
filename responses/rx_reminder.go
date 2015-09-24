package responses

import (
	"strings"

	"github.com/sprucehealth/backend/common"
)

// RXReminder represents the information sent back to the caller when requesting information about an teatment's rx reminders
type RXReminder struct {
	TreatmentID  int64    `json:"treatment_id,string"`
	ReminderText string   `json:"reminder_text"`
	Interval     string   `json:"interval"`
	Days         []string `json:"days,omitempty"`
	Times        []string `json:"times"`
	CreationDate int64    `json:"creation_date"`
}

// TransformRXReminder transforms the provided RXReminder model into the client response
func TransformRXReminder(r *common.RXReminder) *RXReminder {
	return &RXReminder{
		TreatmentID:  r.TreatmentID,
		ReminderText: r.ReminderText,
		Interval:     r.Interval.String(),
		Days:         common.RXRDaySlice(r.Days).Strings(),
		Times:        strings.Split(r.Times, `,`),
		CreationDate: r.CreationDate.Unix(),
	}
}
