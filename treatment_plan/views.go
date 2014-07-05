package treatment_plan

import (
	"errors"

	"github.com/sprucehealth/backend/app_url"
)

const (
	sectionHeaderStyle        = "section_header"
	smallGrayStyle            = "small_gray"
	subheaderStyle            = "subheader"
	treatmentViewNamespace    = "treatment"
	captionRegularItalicStyle = "caption_regular_italic"
	bulletedStyle             = "buletted"
	numberedStyle             = "numbered"
)

type tpView interface {
	Validate() error
}

type tpHeroHeaderView struct {
	Type    string               `json:"type"`
	Title   string               `json:"title"`
	IconURL *app_url.SpruceAsset `json:"icon_url"`
}

func (v *tpHeroHeaderView) Validate() error {
	v.Type = treatmentViewNamespace + ":hero_header"
	return nil
}

type tpSmallDividerView struct {
	Type string `json:"type"`
}

func (v *tpSmallDividerView) Validate() error {
	v.Type = treatmentViewNamespace + ":small_divider"
	return nil
}

type tpSmallHeaderView struct {
	Type        string               `json:"type"`
	Title       string               `json:"title"`
	IconURL     *app_url.SpruceAsset `json:"icon_url"`
	RoundedIcon bool                 `json:"rounded_icon"`
}

func (v *tpSmallHeaderView) Validate() error {
	v.Type = treatmentViewNamespace + ":small_header"
	return nil
}

type tpCardView struct {
	Type  string   `json:"type"`
	Views []tpView `json:"view"`
}

func (v *tpCardView) Validate() error {
	v.Type = treatmentViewNamespace + ":card_view"
	for _, subView := range v.Views {
		if err := subView.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type tpCardTitleView struct {
	Type        string               `json:"type"`
	Title       string               `json:"title"`
	IconURL     *app_url.SpruceAsset `json:"icon_url"`
	RoundedIcon bool                 `json:"rounded_icon,omitempty"`
}

func (v *tpCardTitleView) Validate() error {
	v.Type = treatmentViewNamespace + ":card_title_view"
	return nil
}

type tpTextDisclosureButtonView struct {
	Type   string                `json:"type"`
	Style  string                `json:"style"`
	Text   string                `json:"text"`
	TapURL *app_url.SpruceAction `json:"tap_url"`
}

func (v *tpTextDisclosureButtonView) Validate() error {
	v.Type = treatmentViewNamespace + ":text_disclosure_button"
	return nil
}

type tpLargeDividerView struct {
	Type string `json:"type"`
}

func (v *tpLargeDividerView) Validate() error {
	v.Type = treatmentViewNamespace + ":large_divider"
	return nil
}

type tpImageView struct {
	Type        string `json:"type"`
	ImageWidth  int    `json:"image_width"`
	ImageHeight int    `json:"image_height"`
	ImageURL    string `json:"image_url"`
	Insets      string `json:"insets"`
}

func (v *tpImageView) Validate() error {
	v.Type = treatmentViewNamespace + ":image"
	return nil
}

type tpIconTitleSubtitleView struct {
	Type     string               `json:"type"`
	IconURL  *app_url.SpruceAsset `json:"icon_url"`
	Title    string               `json:"title"`
	Subtitle string               `json:"subtitle"`
}

func (v *tpIconTitleSubtitleView) Validate() error {
	v.Type = treatmentViewNamespace + ":icon_title_subtitle_view"
	return nil
}

type tpTextView struct {
	Type  string `json:"type"`
	Style string `json:"style,omitempty"`
	Text  string `json:"text"`
}

func (v *tpTextView) Validate() error {
	v.Type = treatmentViewNamespace + ":text"
	return nil
}

type tpIconTextView struct {
	Type       string               `json:"type"`
	IconURL    *app_url.SpruceAsset `json:"icon_url"`
	IconWidth  int                  `json:"icon_width,omitempty"`
	IconHeight int                  `json:"icon_height,omitempty"`
	Style      string               `json:"style,omitempty"`
	Text       string               `json:"text"`
	TextStyle  string               `json:"text_style,omitempty"`
}

func (v *tpIconTextView) Validate() error {
	v.Type = treatmentViewNamespace + ":icon_text_view"
	return nil
}

type tpSnippetDetailsView struct {
	Type    string `json:"type"`
	Snippet string `json:"snippet"`
	Details string `json:"details"`
}

func (v *tpSnippetDetailsView) Validate() error {
	v.Type = treatmentViewNamespace + ":snippet_details"
	return nil
}

type tpListElementView struct {
	Type         string `json:"type"`
	ElementStyle string `json:"element_style"` // numbered, dont
	Number       int    `json:"number,omitempty"`
	Text         string `json:"text"`
}

func (v *tpListElementView) Validate() error {
	if v.ElementStyle != "numbered" && v.ElementStyle != "dont" && v.ElementStyle != "buletted" {
		return errors.New("ListElementView expects ElementStyle of numbered or dont, not " + v.ElementStyle)
	}
	v.Type = treatmentViewNamespace + ":list_element"
	return nil
}

type tpPlainButtonView struct {
	Type   string                `json:"type"`
	Text   string                `json:"text"`
	TapURL *app_url.SpruceAction `json:"tap_url"`
}

func (v *tpPlainButtonView) Validate() error {
	v.Type = treatmentViewNamespace + ":plain_button"
	return nil
}

type tpButtonView struct {
	Type    string                `json:"type"`
	Text    string                `json:"text"`
	TapURL  *app_url.SpruceAction `json:"tap_url"`
	IconURL *app_url.SpruceAsset  `json:"icon_url"`
}

func (v *tpButtonView) Validate() error {
	v.Type = treatmentViewNamespace + ":button"
	return nil
}

type tpPrescriptionView struct {
	Type            string               `json:"type"`
	IconURL         *app_url.SpruceAsset `json:"icon_url"`
	Title           string               `json:"title"`
	Description     string               `json:"description"`
	SmallHeaderText string               `json:"small_header_text"`
	Buttons         []tpView             `json:"buttons,omitempty"`
}

func (v *tpPrescriptionView) Validate() error {
	v.Type = treatmentViewNamespace + ":prescription"

	for _, subView := range v.Buttons {
		if err := subView.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type tpPrescriptionButtonView struct {
	Type    string                `json:"type"`
	Text    string                `json:"text"`
	IconURL *app_url.SpruceAsset  `json:"icon_url"`
	TapURL  *app_url.SpruceAction `json:"tap_url"`
}

func (v *tpPrescriptionButtonView) Validate() error {
	v.Type = treatmentViewNamespace + ":prescription_button"
	return nil
}

type tpButtonFooterView struct {
	Type       string                `json:"type"`
	FooterText string                `json:"footer_text"`
	ButtonText string                `json:"button_text"`
	IconURL    *app_url.SpruceAsset  `json:"icon_url"`
	TapURL     *app_url.SpruceAction `json:"tap_url"`
}

func (v *tpButtonFooterView) Validate() error {
	v.Type = treatmentViewNamespace + ":button_footer"
	return nil
}
