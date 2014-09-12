package schedmsg

import (
	"bytes"
	"fmt"
	"time"

	texttemplate "text/template"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
)

var Events map[string]bool = map[string]bool{}

func MustRegisterEvent(event string) {
	if Events[event] == true {
		panic(event + " already registered")
	}
	Events[event] = true
}

type CaseContext struct {
	PatientID     int64
	PatientCaseID int64
	SenderRole    string
	ProviderID    int64
	PersonID      int64
}

func ScheduleInAppMessage(dataAPI api.DataAPI, event string, ctxt interface{}, caseCtxt *CaseContext) error {

	if !Events[event] {
		return fmt.Errorf("Unregistered event %s", event)
	}

	// look up any existing templates
	templates, err := dataAPI.ScheduledMessageTemplates(event)
	if err == api.NoRowsError {
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
			Event:       event,
			PatientID:   caseCtxt.PatientID,
			MessageType: common.SMCaseMessageType,
			MessageJSON: &caseMessage{
				Message:        b.String(),
				PatientCaseID:  caseCtxt.PatientCaseID,
				SenderPersonID: caseCtxt.PersonID,
				SenderRole:     caseCtxt.SenderRole,
				ProviderID:     caseCtxt.ProviderID,
			},
			Scheduled: time.Now().Add(time.Duration(template.SchedulePeriod) * time.Second),
			Status:    common.SMScheduled,
		}

		if err := dataAPI.CreateScheduledMessage(scheduledMessage); err != nil {
			golog.Errorf("Unable to create scheduled message: %s", err)
			return err
		}

		b.Reset()
	}
	return nil
}
