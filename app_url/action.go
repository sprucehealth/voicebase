package app_url

import (
	"fmt"
	"github.com/sprucehealth/backend/libs/golog"
	"net/url"
	"strconv"
	"strings"
)

type SpruceAction struct {
	name   string
	params url.Values
}

func (s SpruceAction) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(spruceActionUrl)+len(s.name)+len(s.params.Encode())+3)
	b = append(b, '"')
	b = append(b, []byte(spruceActionUrl)...)
	b = append(b, []byte(s.name)...)
	if len(s.params) > 0 {
		b = append(b, '?')
		b = append(b, []byte(s.params.Encode())...)
	}

	b = append(b, '"')

	return b, nil
}

func (s SpruceAction) String() string {
	if len(s.params) > 0 {
		return fmt.Sprintf("%s%s?%s", spruceActionUrl, s.name, s.params.Encode())
	}
	return spruceActionUrl + s.name
}

func (s *SpruceAction) UnmarshalJSON(data []byte) error {
	if len(data) < 3 {
		return nil
	}

	incomingUrl := string(data[1 : len(data)-1])
	spruceUrlComponents, err := url.Parse(incomingUrl)
	if err != nil {
		golog.Errorf("Unable to parse url for spruce action %s", err)
		return err
	}
	pathComponents := strings.Split(spruceUrlComponents.Path, "/")
	if len(pathComponents) < 3 {
		golog.Errorf("Unable to break path %#v into its components when attempting to unmarshal %s", pathComponents, incomingUrl)
		return nil
	}
	s.name = pathComponents[2]

	s.params, err = url.ParseQuery(spruceUrlComponents.RawQuery)
	if err != nil {
		return err
	}
	return nil
}

func ClaimPatientCaseAction(patientCaseId int64) *SpruceAction {
	params := url.Values{}
	params.Set("case_id", strconv.FormatInt(patientCaseId, 10))
	return &SpruceAction{
		name:   "claim_patient_case",
		params: params,
	}
}

func BeginPatientVisitReviewAction(patientId, patientVisitId, patientCaseId int64) *SpruceAction {
	params := url.Values{}
	params.Set("patient_visit_id", strconv.FormatInt(patientVisitId, 10))
	params.Set("patient_id", strconv.FormatInt(patientId, 10))
	params.Set("case_id", strconv.FormatInt(patientCaseId, 10))
	return &SpruceAction{
		name:   "begin_patient_visit",
		params: params,
	}
}

func ViewCompletedPatientVisitAction(patientId, patientVisitId, patientCaseId int64) *SpruceAction {
	params := url.Values{}
	params.Set("patient_visit_id", strconv.FormatInt(patientVisitId, 10))
	params.Set("patient_id", strconv.FormatInt(patientId, 10))
	params.Set("case_id", strconv.FormatInt(patientCaseId, 10))
	return &SpruceAction{
		name:   "view_completed_patient_visit",
		params: params,
	}
}

func ViewRefillRequestAction(patientId, refillRequestId int64) *SpruceAction {
	params := url.Values{}
	params.Set("refill_request_id", strconv.FormatInt(refillRequestId, 10))
	params.Set("patient_id", strconv.FormatInt(patientId, 10))
	return &SpruceAction{
		name:   "view_refill_request",
		params: params,
	}
}

func ViewTransmissionErrorAction(patientId, treatmentId int64) *SpruceAction {
	params := url.Values{}
	params.Set("treatment_id", strconv.FormatInt(treatmentId, 10))
	params.Set("patient_id", strconv.FormatInt(patientId, 10))
	return &SpruceAction{
		name:   "view_transmission_error",
		params: params,
	}
}

func ViewPatientTreatmentsAction(patientId int64) *SpruceAction {
	params := url.Values{}
	params.Set("patient_id", strconv.FormatInt(patientId, 10))
	return &SpruceAction{
		name:   "view_patient_treatments",
		params: params,
	}
}

func ViewPatientConversationsAction(patientId, conversationId int64) *SpruceAction {
	params := url.Values{}
	params.Set("conversation_id", strconv.FormatInt(conversationId, 10))
	params.Set("patient_id", strconv.FormatInt(patientId, 10))
	return &SpruceAction{
		name:   "view_patient_conversations",
		params: params,
	}
}

func ViewPatientVisitAction(patientVisitId int64) *SpruceAction {
	params := url.Values{}
	params.Set("visit_id", strconv.FormatInt(patientVisitId, 10))
	return &SpruceAction{
		name:   "view_visit",
		params: params,
	}
}

func ContinueVisitAction(patientVisitId int64) *SpruceAction {
	params := url.Values{}
	params.Set("visit_id", strconv.FormatInt(patientVisitId, 10))
	return &SpruceAction{
		name:   "continue_visit",
		params: params,
	}
}

func ViewTreatmentPlanAction(treatmentPlanId int64) *SpruceAction {
	params := url.Values{}
	params.Set("treatment_plan_id", strconv.FormatInt(treatmentPlanId, 10))
	return &SpruceAction{
		name:   "view_treatment_plan",
		params: params,
	}
}

func ViewMessagesAction(conversationId int64) *SpruceAction {
	params := url.Values{}
	params.Set("conversation_id", strconv.FormatInt(conversationId, 10))
	return &SpruceAction{
		name:   "view_messages",
		params: params,
	}
}

func ViewCareTeam() *SpruceAction {
	return &SpruceAction{
		name: "view_care_team",
	}
}

func ViewTreatmentGuideAction(treatmentId int64) *SpruceAction {
	params := url.Values{}
	params.Set("treatment_id", strconv.FormatInt(treatmentId, 10))
	return &SpruceAction{
		name:   "view_treatment_guide",
		params: params,
	}
}

func MessageAction() *SpruceAction {
	return &SpruceAction{
		name: "message",
	}
}
