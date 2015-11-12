package common

import "time"

type TreatmentPlanScheduledMessage struct {
	ID                 int64                    `json:"id,string"`
	TreatmentPlanID    int64                    `json:"treatment_plan_id,string"`
	ScheduledDays      int                      `json:"scheduled_days"`
	Message            string                   `json:"message"`
	Attachments        []*CaseMessageAttachment `json:"attachments"`
	ScheduledMessageID *int64                   `json:"scheduled_message_id,string,omitempty"`
	Cancelled          bool                     `json:"cancelled"`
	CancelledTime      *time.Time               `json:"cancelled_time"`
	SentTime           *time.Time               `json:"sent_time"`
}

func (m *TreatmentPlanScheduledMessage) Equal(to *TreatmentPlanScheduledMessage) bool {
	if m.Message != to.Message {
		return false
	}
	if m.ScheduledDays != to.ScheduledDays {
		return false
	}
	if len(m.Attachments) != len(to.Attachments) {
		return false
	}

	for _, a1 := range m.Attachments {
		matched := false
		for _, a2 := range to.Attachments {
			if a1.ItemType == a2.ItemType && a1.ItemID == a2.ItemID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// TreatmentPlanScheduledMessageCancellable returns true if the treatment plan is active, the message has not yet been sent
// and the message has not been cancelled
func TreatmentPlanScheduledMessageCancellable(tp *TreatmentPlan, m *TreatmentPlanScheduledMessage, accountRole string) bool {
	return tp.Status == TPStatusActive && !m.Cancelled && m.SentTime == nil && accountRole == "MA"
}
