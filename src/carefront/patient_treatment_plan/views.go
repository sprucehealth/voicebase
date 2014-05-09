package patient_treatment_plan

import (
	"carefront/app_url"
	"carefront/libs/pharmacy"
	"errors"
)

const (
	sectionHeaderStyle     = "section_header"
	smallGrayStyle         = "small_gray"
	subheaderStyle         = "subheader"
	treatmentViewNamespace = "treatment"
	timeFormatlayout       = "January 2 at 3:04pm"
)

type TPView interface {
	Validate() error
}

type TPVisitHeaderView struct {
	Type     string               `json:"type"`
	ImageURL *app_url.SpruceAsset `json:"image_url"`
	Title    string               `json:"title"`
	Subtitle string               `json:"subtitle"`
}

func (v *TPVisitHeaderView) Validate() error {
	v.Type = treatmentViewNamespace + ":visit_header"
	return nil
}

type TPSmallDividerView struct {
	Type string `json:"type"`
}

func (v *TPSmallDividerView) Validate() error {
	v.Type = treatmentViewNamespace + ":small_divider"
	return nil
}

type TPLargeDividerView struct {
	Type string `json:"type"`
}

func (v *TPLargeDividerView) Validate() error {
	v.Type = treatmentViewNamespace + ":large_divider"
	return nil
}

type TPImageView struct {
	Type        string `json:"type"`
	ImageWidth  int    `json:"image_width"`
	ImageHeight int    `json:"image_height"`
	ImageURL    string `json:"image_url"`
	Insets      string `json:"insets"`
}

func (v *TPImageView) Validate() error {
	v.Type = treatmentViewNamespace + ":image"
	return nil
}

type TPIconTitleSubtitleView struct {
	Type     string               `json:"type"`
	IconURL  *app_url.SpruceAsset `json:"icon_url"`
	Title    string               `json:"title"`
	Subtitle string               `json:"subtitle"`
}

func (v *TPIconTitleSubtitleView) Validate() error {
	v.Type = treatmentViewNamespace + ":icon_title_subtitle_view"
	return nil
}

type TPTextView struct {
	Type  string `json:"type"`
	Style string `json:"style,omitempty"`
	Text  string `json:"text"`
}

func (v *TPTextView) Validate() error {
	v.Type = treatmentViewNamespace + ":text"
	return nil
}

type TPIconTextView struct {
	Type       string               `json:"type"`
	IconURL    *app_url.SpruceAsset `json:"icon_url"`
	IconWidth  int                  `json:"icon_width,omitempty"`
	IconHeight int                  `json:"icon_height,omitempty"`
	Style      string               `json:"style,omitempty"`
	Text       string               `json:"text"`
	TextStyle  string               `json:"text_style,omitempty"`
}

func (v *TPIconTextView) Validate() error {
	v.Type = treatmentViewNamespace + ":icon_text_view"
	return nil
}

type TPSnippetDetailsView struct {
	Type    string `json:"type"`
	Snippet string `json:"snippet"`
	Details string `json:"details"`
}

func (v *TPSnippetDetailsView) Validate() error {
	v.Type = treatmentViewNamespace + ":snippet_details"
	return nil
}

type TPListElementView struct {
	Type         string `json:"type"`
	ElementStyle string `json:"element_style"` // numbered, dont
	Number       int    `json:"number,omitempty"`
	Text         string `json:"text"`
}

func (v *TPListElementView) Validate() error {
	if v.ElementStyle != "numbered" && v.ElementStyle != "dont" && v.ElementStyle != "buletted" {
		return errors.New("ListElementView expects ElementStyle of numbered or dont, not " + v.ElementStyle)
	}
	v.Type = treatmentViewNamespace + ":list_element"
	return nil
}

type TPPlainButtonView struct {
	Type   string                `json:"type"`
	Text   string                `json:"text"`
	TapURL *app_url.SpruceAction `json:"tap_url"`
}

func (v *TPPlainButtonView) Validate() error {
	v.Type = treatmentViewNamespace + ":plain_button"
	return nil
}

type TPButtonView struct {
	Type    string                `json:"type"`
	Text    string                `json:"text"`
	TapURL  *app_url.SpruceAction `json:"tap_url"`
	IconURL *app_url.SpruceAsset  `json:"icon_url"`
}

func (v *TPButtonView) Validate() error {
	v.Type = treatmentViewNamespace + ":button"
	return nil
}

type TPPrescriptionView struct {
	Type        string                `json:"type"`
	IconURL     *app_url.SpruceAsset  `json:"icon_url"`
	Title       string                `json:"title"`
	Description string                `json:"description"`
	ButtonTitle string                `json:"button_title,omitempty"`
	TapURL      *app_url.SpruceAction `json:"tap_url,omitempty"`
}

func (v *TPPrescriptionView) Validate() error {
	v.Type = treatmentViewNamespace + ":prescription"
	return nil
}

type TPPharmacyMapView struct {
	Type     string                 `json:"type"`
	Pharmacy *pharmacy.PharmacyData `json:"pharmacy"`
}

func (v *TPPharmacyMapView) Validate() error {
	v.Type = treatmentViewNamespace + ":pharmacy_map"
	return nil
}

type TPTreatmentListView struct {
	Type       string            `json:"type"`
	Treatments []*TPIconTextView `json:"treatments"`
}

func (v *TPTreatmentListView) Validate() error {
	v.Type = treatmentViewNamespace + ":treatment_list"
	return nil
}

type TPButtonFooterView struct {
	Type       string                `json:"type"`
	FooterText string                `json:"footer_text"`
	ButtonText string                `json:"button_text"`
	IconURL    *app_url.SpruceAsset  `json:"icon_url"`
	TapURL     *app_url.SpruceAction `json:"tap_url"`
}

func (v *TPButtonFooterView) Validate() error {
	v.Type = treatmentViewNamespace + ":button_footer"
	return nil
}
