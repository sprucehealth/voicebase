package app_url

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/libs/golog"
)

type SpruceAction struct {
	name   string
	params url.Values
}

func (s SpruceAction) MarshalJSON() ([]byte, error) {
	params := s.params.Encode()
	b := make([]byte, 0, len(spruceActionURL)+len(s.name)+len(params)+3)
	b = append(b, '"')
	b = append(b, []byte(spruceActionURL)...)
	b = append(b, []byte(s.name)...)
	if len(s.params) > 0 {
		b = append(b, '?')
		b = append(b, []byte(params)...)
	}

	b = append(b, '"')

	return b, nil
}

func (s SpruceAction) String() string {
	if len(s.params) > 0 {
		return fmt.Sprintf("%s%s?%s", spruceActionURL, s.name, s.params.Encode())
	}
	return spruceActionURL + s.name
}

func (s *SpruceAction) UnmarshalJSON(data []byte) error {
	if len(data) < 3 {
		return nil
	}

	incomingURL := string(data[1 : len(data)-1])
	spruceURLComponents, err := url.Parse(incomingURL)
	if err != nil {
		golog.Errorf("Unable to parse url for spruce action %s", err)
		return err
	}
	pathComponents := strings.Split(spruceURLComponents.Path, "/")
	if len(pathComponents) < 3 {
		golog.Errorf("Unable to break path %#v into its components when attempting to unmarshal %s", pathComponents, incomingURL)
		return nil
	}
	s.name = pathComponents[2]

	s.params, err = url.ParseQuery(spruceURLComponents.RawQuery)
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

func ViewPatientVisitInfoAction(patientId, patientVisitId, patientCaseId int64) *SpruceAction {
	params := url.Values{}
	params.Set("patient_visit_id", strconv.FormatInt(patientVisitId, 10))
	params.Set("patient_id", strconv.FormatInt(patientId, 10))
	params.Set("case_id", strconv.FormatInt(patientCaseId, 10))
	return &SpruceAction{
		name:   "view_patient_visit",
		params: params,
	}
}

func ViewCompletedTreatmentPlanAction(patientId, treatmentPlanId, patientCaseId int64) *SpruceAction {
	params := url.Values{}
	params.Set("treatment_plan_id", strconv.FormatInt(treatmentPlanId, 10))
	params.Set("patient_id", strconv.FormatInt(patientId, 10))
	params.Set("case_id", strconv.FormatInt(patientCaseId, 10))
	return &SpruceAction{
		name:   "view_treatment_plan",
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

func ViewDNTFTransmissionErrorAction(patientId, treatmentId int64) *SpruceAction {
	params := url.Values{}
	params.Set("unlinked_dntf_treatment_id", strconv.FormatInt(treatmentId, 10))
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

func ViewPatientMessagesAction(patientId, patientCaseId int64) *SpruceAction {
	params := url.Values{}
	params.Set("case_id", strconv.FormatInt(patientCaseId, 10))
	params.Set("patient_id", strconv.FormatInt(patientId, 10))
	return &SpruceAction{
		name:   "view_patient_messages",
		params: params,
	}
}

func ViewCaseMessageAction(messageId, patientCaseId int64) *SpruceAction {
	params := url.Values{}
	params.Set("message_id", strconv.FormatInt(messageId, 10))
	params.Set("case_id", strconv.FormatInt(patientCaseId, 10))
	return &SpruceAction{
		name:   "view_case_message",
		params: params,
	}
}

func ViewCaseMessageThreadAction(patientCaseId int64) *SpruceAction {
	params := url.Values{}
	params.Set("case_id", strconv.FormatInt(patientCaseId, 10))
	return &SpruceAction{
		name:   "view_case_message",
		params: params,
	}
}

func ViewTreatmentPlanMessageAction(messageId, treatmentPlanId, patientCaseId int64) *SpruceAction {
	params := url.Values{}
	params.Set("message_id", strconv.FormatInt(messageId, 10))
	params.Set("treatment_plan_id", strconv.FormatInt(treatmentPlanId, 10))
	params.Set("case_id", strconv.FormatInt(patientCaseId, 10))
	return &SpruceAction{
		name:   "view_treatment_plan_message",
		params: params,
	}
}

func SendCaseMessageAction(patientCaseId int64) *SpruceAction {
	params := url.Values{}
	params.Set("case_id", strconv.FormatInt(patientCaseId, 10))
	return &SpruceAction{
		name:   "send_case_message",
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

func ViewTreatmentPlanForCaseAction(patientCaseID int64) *SpruceAction {
	params := url.Values{}
	params.Set("case_id", strconv.FormatInt(patientCaseID, 10))
	return &SpruceAction{
		name:   "view_treatment_plan",
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

func ViewCaseAction(patientCaseId int64) *SpruceAction {
	params := url.Values{}
	params.Set("case_id", strconv.FormatInt(patientCaseId, 10))
	return &SpruceAction{
		name:   "view_case",
		params: params,
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

func ViewPreferredPharmacyAction() *SpruceAction {
	return &SpruceAction{
		name: "view_preferred_pharmacy",
	}
}

func ViewSampleDoctorProfilesAction() *SpruceAction {
	return &SpruceAction{
		name: "view_sample_doctor_profiles",
	}
}

func ViewTutorialAction() *SpruceAction {
	return &SpruceAction{
		name: "view_tutorial",
	}
}

func ViewSampleTreatmentPlanAction() *SpruceAction {
	return &SpruceAction{
		name: "view_sample_treatment_plan",
	}
}

func StartVisitAction() *SpruceAction {
	return &SpruceAction{
		name: "start_visit",
	}
}

func EmailSupportAction() *SpruceAction {
	return &SpruceAction{
		name: "email_support",
	}
}

func ViewResourceLibraryAction() *SpruceAction {
	return &SpruceAction{
		name: "view_resource_library",
	}
}

func ViewPharmacyInMapAction() *SpruceAction {
	return &SpruceAction{
		name: "view_pharmacy_in_map",
	}
}

func ViewSpruceFAQAction() *SpruceAction {
	return &SpruceAction{
		name: "view_faq",
	}
}
