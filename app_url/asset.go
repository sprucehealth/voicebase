package app_url

import (
	"net/url"
	"strings"

	"github.com/sprucehealth/backend/libs/golog"
)

// Image assets available in the app. Only these assets should ever be used.
var (
	IconBlueTreatmentPlan       = &SpruceAsset{name: "icon_blue_treatment_plan"}
	IconBlueTriage              = &SpruceAsset{name: "icon_blue_triage"}
	IconBlueSuccess             = &SpruceAsset{name: "icon_blue_success"}
	IconCaseLarge               = &SpruceAsset{name: "icon_case_large"}
	IconCaseSmall               = &SpruceAsset{name: "icon_case_small"}
	IconConsentLarge            = &SpruceAsset{name: "icon_consent_large"}
	IconFAQ                     = &SpruceAsset{name: "icon_faq_large"}
	IconGuide                   = &SpruceAsset{name: "icon_guide"}
	IconHomeConversationNormal  = &SpruceAsset{name: "icon_home_conversation_normal"}
	IconHomeTreatmentPlanNormal = &SpruceAsset{name: "icon_home_treatmentplan_normal"}
	IconHomeVisitNormal         = &SpruceAsset{name: "icon_home_visit_normal"}
	IconLearnSpruce             = &SpruceAsset{name: "icon_learn_spruce"}
	IconMessage                 = &SpruceAsset{name: "icon_message"}
	IconMessagesLarge           = &SpruceAsset{name: "icon_messages_large"}
	IconMessagesSmall           = &SpruceAsset{name: "icon_messages_small"}
	IconOTCLarge                = &SpruceAsset{name: "icon_otc_large"}
	IconPrescriptionOral        = &SpruceAsset{name: "icon_oral_prescription"}
	IconPrescriptionTopical     = &SpruceAsset{name: "icon_topical_prescription"}
	IconProfileEducation        = &SpruceAsset{name: "icon_profile_education"}
	IconProfileExperience       = &SpruceAsset{name: "icon_profile_experience"}
	IconProfileQualifications   = &SpruceAsset{name: "icon_profile_qualifications"}
	IconProfileSpruceLogo       = &SpruceAsset{name: "icon_profile_spruce_logo"}
	IconPromo10                 = &SpruceAsset{name: "icon_promo_10"}
	IconPromoLogo               = &SpruceAsset{name: "icon_promo_logo"}
	IconReferLarge              = &SpruceAsset{name: "icon_refer_large"}
	IconRegimen                 = &SpruceAsset{name: "icon_regimen"}
	IconRegimenMorning          = &SpruceAsset{name: "icon_morning_regimen"}
	IconRegimenNight            = &SpruceAsset{name: "icon_night_regimen"}
	IconReply                   = &SpruceAsset{name: "icon_reply"}
	IconResourceLibrary         = &SpruceAsset{name: "icon_resources"}
	IconRX                      = &SpruceAsset{name: "icon_rx"}
	IconRXGuide                 = &SpruceAsset{name: "icon_rx_guide"}
	IconRXLarge                 = &SpruceAsset{name: "icon_rx_large"}
	IconSpruceDoctors           = &SpruceAsset{name: "icon_spruce_doctors"}
	IconSupport                 = &SpruceAsset{name: "icon_support"}
	IconTreatmentPlanBlueButton = &SpruceAsset{name: "icon_treatment_plan_blue_button"}
	IconTreatmentPlanLarge      = &SpruceAsset{name: "icon_treatment_plan_large"}
	IconTreatmentPlanSmall      = &SpruceAsset{name: "icon_treatment_plan_small"}
	IconVisitLarge              = &SpruceAsset{name: "icon_visit_large"}
	IconVisitSubmitted          = &SpruceAsset{name: "icon_visit_submitted"}
	IconWhiteCase               = &SpruceAsset{name: "icon_white_case"}
	PatientVisitQueueIcon       = &SpruceAsset{name: "patient_visit_queue_icon"}
	TmpSignature                = &SpruceAsset{name: "tmp_signature"}
	Treatment                   = &SpruceAsset{name: "treatment"}
	IconTickLarge               = &SpruceAsset{name: "icon_tick_large"}
)

type SpruceAsset struct {
	name string
}

func (s SpruceAsset) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(spruceImageURL)+len(s.name)+2)
	b = append(b, '"')
	b = append(b, []byte(spruceImageURL)...)
	b = append(b, []byte(s.name)...)
	b = append(b, '"')

	return b, nil
}

func (s SpruceAsset) String() string {
	return spruceImageURL + s.name
}

func (s *SpruceAsset) UnmarshalJSON(data []byte) error {
	if len(data) < 3 {
		return nil
	}
	incomingURL := string(data[1 : len(data)-1])
	spruceURLComponents, err := url.Parse(incomingURL)
	if err != nil {
		golog.Errorf("Unable to parse url for spruce asset %s", err)
		return err
	}
	pathComponents := strings.Split(spruceURLComponents.Path, "/")
	if len(pathComponents) < 3 {
		golog.Errorf("Unable to break path %#v into its components when attempting to unmarshal %s", pathComponents, incomingURL)
		return nil
	}
	s.name = pathComponents[2]
	return nil
}
