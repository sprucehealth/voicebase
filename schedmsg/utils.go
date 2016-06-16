package schedmsg

import (
	"bytes"
	"fmt"
	"time"

	texttemplate "text/template"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
)

var Events = map[string]bool{}

func MustRegisterEvent(event string) {
	if Events[event] == true {
		panic(event + " already registered")
	}
	Events[event] = true
}

// CaseInfo is a struct representing the case information
// for which a message is intended to be scheduled.
type CaseInfo struct {
	// PatientID is the unique id of the patient for which the message is to go out.
	PatientID common.PatientID
	// PatientCaseID is the unique if of the case to identify the thread in which to insert the message.
	PatientCaseID int64
	// SenderRole identifies the type of provider (CC or DOCTOR).
	SenderRole string
	// ProviderID is the unique ID of the provider sending the message.
	ProviderID int64
	// PersonID is the unique ID of the sender in the message thread.
	PersonID int64
	// IsAutomated indicates whether or not the message being sent out was automatically (true) or manuall/explcitly (false)
	// scheduled by the sender.
	IsAutomated bool
}

// ScheduleInAppMessage queues up a case message to be sent as defined by the context and the case info.
func ScheduleInAppMessage(dataAPI api.DataAPI, event string, ctxt interface{}, caseCtxt *CaseInfo) error {
	if !Events[event] {
		return fmt.Errorf("Unregistered event %s", event)
	}

	// look up any existing templates
	templates, err := dataAPI.ScheduledMessageTemplates(event)
	if api.IsErrNotFound(err) {
		// nothing to do for this event if no templates exist
		return nil
	} else if err != nil {
		return err
	}

	var b bytes.Buffer

	// create a scheduled message for every template
	for _, template := range templates {
		msgTemplate, err := texttemplate.New("").Parse(template.Message)
		if err != nil {
			return err
		}
		if err := msgTemplate.Execute(&b, ctxt); err != nil {
			return err
		}

		scheduledMessage := &common.ScheduledMessage{
			Event:     event,
			PatientID: caseCtxt.PatientID,
			Message: &CaseMessage{
				Message:        b.String(),
				PatientCaseID:  caseCtxt.PatientCaseID,
				SenderPersonID: caseCtxt.PersonID,
				SenderRole:     caseCtxt.SenderRole,
				ProviderID:     caseCtxt.ProviderID,
				IsAutomated:    caseCtxt.IsAutomated,
			},
			Scheduled: time.Now().Add(time.Duration(template.SchedulePeriod) * time.Second),
			Status:    common.SMScheduled,
		}

		if _, err := dataAPI.CreateScheduledMessage(scheduledMessage); err != nil {
			golog.Errorf("Unable to create scheduled message: %s", err)
			return err
		}

		b.Reset()
	}
	return nil
}
