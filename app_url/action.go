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

func ParseSpruceAction(s string) (SpruceAction, error) {
	sa := SpruceAction{}

	ur, err := url.Parse(s)
	if err != nil {
		return sa, fmt.Errorf("app_url: unable to parse URL for spruce action '%s': %s", s, err)
	}
	pathComponents := strings.Split(ur.Path, "/")
	if len(pathComponents) < 3 {
		return sa, fmt.Errorf("app_url: cannot parse path for spruce action '%s'", s)
	}
	sa.name = pathComponents[2]
	sa.params, err = url.ParseQuery(ur.RawQuery)
	return sa, err
}

func (s SpruceAction) IsZero() bool {
	return s.name == ""
}

func (s SpruceAction) String() string {
	if len(s.params) > 0 {
		return spruceActionURL + s.name + "?" + s.params.Encode()
	}
	return spruceActionURL + s.name
}

func (s SpruceAction) MarshalText() ([]byte, error) {
	b, err := s.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return b[1 : len(b)-1], nil
}

func (s *SpruceAction) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return nil
	}

	sa, err := ParseSpruceAction(string(text))
	if err != nil {
		golog.Errorf(err.Error())
		return nil
	}

	*s = sa
	return nil
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

func (s *SpruceAction) UnmarshalJSON(data []byte) error {
	if len(data) < 3 {
		return nil
	}
	return s.UnmarshalText(data[1 : len(data)-1])
}

func ClaimPatientCaseAction(patientCaseID int64) *SpruceAction {
	params := url.Values{}
	params.Set("case_id", strconv.FormatInt(patientCaseID, 10))
	return &SpruceAction{
		name:   "claim_patient_case",
		params: params,
	}
}

func ViewPatientVisitInfoAction(patientID, patientVisitID, patientCaseID int64) *SpruceAction {
	params := url.Values{}
	params.Set("patient_visit_id", strconv.FormatInt(patientVisitID, 10))
	params.Set("patient_id", strconv.FormatInt(patientID, 10))
	params.Set("case_id", strconv.FormatInt(patientCaseID, 10))
	return &SpruceAction{
		name:   "view_patient_visit",
		params: params,
	}
}

func ViewCompletedTreatmentPlanAction(patientID, treatmentPlanID, patientCaseID int64) *SpruceAction {
	params := url.Values{}
	params.Set("treatment_plan_id", strconv.FormatInt(treatmentPlanID, 10))
	params.Set("patient_id", strconv.FormatInt(patientID, 10))
	params.Set("case_id", strconv.FormatInt(patientCaseID, 10))
	return &SpruceAction{
		name:   "view_treatment_plan",
		params: params,
	}
}

func ViewRefillRequestAction(patientID, refillRequestID int64) *SpruceAction {
	params := url.Values{}
	params.Set("refill_request_id", strconv.FormatInt(refillRequestID, 10))
	params.Set("patient_id", strconv.FormatInt(patientID, 10))
	return &SpruceAction{
		name:   "view_refill_request",
		params: params,
	}
}

func ViewTransmissionErrorAction(patientID, treatmentID int64) *SpruceAction {
	params := url.Values{}
	params.Set("treatment_id", strconv.FormatInt(treatmentID, 10))
	params.Set("patient_id", strconv.FormatInt(patientID, 10))
	return &SpruceAction{
		name:   "view_transmission_error",
		params: params,
	}
}

func ViewDNTFTransmissionErrorAction(patientID, treatmentID int64) *SpruceAction {
	params := url.Values{}
	params.Set("unlinked_dntf_treatment_id", strconv.FormatInt(treatmentID, 10))
	params.Set("patient_id", strconv.FormatInt(patientID, 10))
	return &SpruceAction{
		name:   "view_transmission_error",
		params: params,
	}
}

func ViewPatientTreatmentsAction(patientID int64) *SpruceAction {
	params := url.Values{}
	params.Set("patient_id", strconv.FormatInt(patientID, 10))
	return &SpruceAction{
		name:   "view_patient_treatments",
		params: params,
	}
}

func ViewPatientMessagesAction(patientID, patientCaseID int64) *SpruceAction {
	params := url.Values{}
	params.Set("case_id", strconv.FormatInt(patientCaseID, 10))
	params.Set("patient_id", strconv.FormatInt(patientID, 10))
	return &SpruceAction{
		name:   "view_patient_messages",
		params: params,
	}
}

func ViewCaseMessageAction(messageId, patientCaseID int64) *SpruceAction {
	params := url.Values{}
	params.Set("message_id", strconv.FormatInt(messageId, 10))
	params.Set("case_id", strconv.FormatInt(patientCaseID, 10))
	return &SpruceAction{
		name:   "view_case_message",
		params: params,
	}
}

func ViewCaseMessageThreadAction(patientCaseID int64) *SpruceAction {
	params := url.Values{}
	params.Set("case_id", strconv.FormatInt(patientCaseID, 10))
	return &SpruceAction{
		name:   "view_case_message",
		params: params,
	}
}

func ViewTreatmentPlanMessageAction(messageId, treatmentPlanID, patientCaseID int64) *SpruceAction {
	params := url.Values{}
	params.Set("message_id", strconv.FormatInt(messageId, 10))
	params.Set("treatment_plan_id", strconv.FormatInt(treatmentPlanID, 10))
	params.Set("case_id", strconv.FormatInt(patientCaseID, 10))
	return &SpruceAction{
		name:   "view_treatment_plan_message",
		params: params,
	}
}

func SendCaseMessageAction(patientCaseID int64) *SpruceAction {
	params := url.Values{}
	params.Set("case_id", strconv.FormatInt(patientCaseID, 10))
	return &SpruceAction{
		name:   "send_case_message",
		params: params,
	}
}

func ViewPatientVisitAction(patientVisitID int64) *SpruceAction {
	params := url.Values{}
	params.Set("visit_id", strconv.FormatInt(patientVisitID, 10))
	return &SpruceAction{
		name:   "view_visit",
		params: params,
	}
}

func ContinueVisitAction(patientVisitID int64) *SpruceAction {
	params := url.Values{}
	params.Set("patient_visit_id", strconv.FormatInt(patientVisitID, 10))
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

func ViewTreatmentPlanAction(treatmentPlanID int64) *SpruceAction {
	params := url.Values{}
	params.Set("treatment_plan_id", strconv.FormatInt(treatmentPlanID, 10))
	return &SpruceAction{
		name:   "view_treatment_plan",
		params: params,
	}
}

func ViewCaseAction(patientCaseID int64) *SpruceAction {
	params := url.Values{}
	params.Set("case_id", strconv.FormatInt(patientCaseID, 10))
	return &SpruceAction{
		name:   "view_case",
		params: params,
	}
}

func ViewTreatmentGuideAction(treatmentID int64) *SpruceAction {
	params := url.Values{}
	params.Set("treatment_id", strconv.FormatInt(treatmentID, 10))
	return &SpruceAction{
		name:   "view_treatment_guide",
		params: params,
	}
}

func ViewRXGuideGuideAction(genericName, route, form, dosage string) *SpruceAction {
	params := url.Values{}
	params.Set("generic_name", genericName)
	params.Set("route", route)
	params.Set("form", form)
	params.Set("dosage", dosage)
	return &SpruceAction{
		name:   "view_rx_guide",
		params: params,
	}
}

func ViewResourceGuideAction(guideID int64) *SpruceAction {
	params := url.Values{
		"guide_id": []string{strconv.FormatInt(guideID, 10)},
	}
	return &SpruceAction{
		name:   "view_resource_library_guide",
		params: params,
	}
}

func ViewPathwayFAQ(pathwayTag string) *SpruceAction {
	params := url.Values{
		"pathway_id": []string{pathwayTag},
	}
	return &SpruceAction{
		name:   "view_pathway_faq",
		params: params,
	}
}

func ViewSampleTreatmentPlanAction(pathwayTag string) *SpruceAction {
	params := url.Values{
		"pathway_id": []string{pathwayTag},
	}
	return &SpruceAction{
		name:   "view_sample_treatment_plan",
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

func StartVisitAction() *SpruceAction {
	return &SpruceAction{
		name: "start_visit",
	}
}

func ViewSupportAction() *SpruceAction {
	return &SpruceAction{
		name: "view_support",
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

func ViewPricingFAQAction() *SpruceAction {
	return &SpruceAction{
		name: "view_pricing_faq",
	}
}

func ViewReferFriendAction() *SpruceAction {
	return &SpruceAction{
		name: "view_refer_friend",
	}
}

func ViewHomeAction() *SpruceAction {
	return &SpruceAction{name: "view_home"}
}

func ViewChooseDoctorScreen() *SpruceAction {
	return &SpruceAction{name: "view_choose_doctor_screen"}
}

func NotifyMeAction() *SpruceAction {
	return &SpruceAction{name: "notify_when_available"}
}
