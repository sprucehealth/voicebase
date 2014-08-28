package treatment_plan

import (
	"errors"
	"time"

	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/pharmacy"
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
	TypeName() string
}

type tpHeroHeaderView struct {
	Type            string `json:"type"`
	Title           string `json:"title"`
	Subtitle        string `json:"subtitle"`
	CreatedDateText string `json:"created_date_text"`
}

func (v *tpHeroHeaderView) Validate() error {
	v.Type = treatmentViewNamespace + ":hero_header"
	return nil
}

func (v *tpHeroHeaderView) TypeName() string {
	return v.Type
}

type tpSmallDividerView struct {
	Type string `json:"type"`
}

func (v *tpSmallDividerView) Validate() error {
	v.Type = treatmentViewNamespace + ":small_divider"
	return nil
}

func (v *tpSmallDividerView) TypeName() string {
	return v.Type
}

type tpCardView struct {
	Type  string   `json:"type"`
	Views []tpView `json:"views"`
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

func (v *tpCardView) TypeName() string {
	return v.Type
}

type tpCardTitleView struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	IconURL     string `json:"icon_url"`
	RoundedIcon bool   `json:"rounded_icon,omitempty"`
}

func (v *tpCardTitleView) Validate() error {
	v.Type = treatmentViewNamespace + ":card_title_view"
	return nil
}

func (v *tpCardTitleView) TypeName() string {
	return v.Type
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

func (v *tpTextDisclosureButtonView) TypeName() string {
	return v.Type
}

type tpLargeDividerView struct {
	Type string `json:"type"`
}

func (v *tpLargeDividerView) Validate() error {
	v.Type = treatmentViewNamespace + ":large_divider"
	return nil
}

func (v *tpLargeDividerView) TypeName() string {
	return v.Type
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

func (v *tpImageView) TypeName() string {
	return v.Type
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

func (v *tpIconTitleSubtitleView) TypeName() string {
	return v.Type
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

func (v *tpTextView) TypeName() string {
	return v.Type
}

type tpIconTextView struct {
	Type       string `json:"type"`
	IconURL    string `json:"icon_url"`
	IconWidth  int    `json:"icon_width,omitempty"`
	IconHeight int    `json:"icon_height,omitempty"`
	Style      string `json:"style,omitempty"`
	Text       string `json:"text"`
	TextStyle  string `json:"text_style,omitempty"`
}

func (v *tpIconTextView) Validate() error {
	v.Type = treatmentViewNamespace + ":icon_text_view"
	return nil
}

func (v *tpIconTextView) TypeName() string {
	return v.Type
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

func (v *tpSnippetDetailsView) TypeName() string {
	return v.Type
}

type tpListElementView struct {
	Type         string `json:"type"`
	ElementStyle string `json:"element_style"` // numbered, dont
	Number       int    `json:"number,omitempty"`
	Text         string `json:"text"`
}

func (v *tpListElementView) Validate() error {
	if v.ElementStyle != bulletedStyle && v.ElementStyle != numberedStyle {
		return errors.New("ListElementView expects ElementStyle of numbered or bulleted, not " + v.ElementStyle)
	}
	v.Type = treatmentViewNamespace + ":list_element"
	return nil
}

func (v *tpListElementView) TypeName() string {
	return v.Type
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

func (v *tpPlainButtonView) TypeName() string {
	return v.Type
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

func (v *tpButtonView) TypeName() string {
	return v.Type
}

type tpPharmacyView struct {
	Type     string                 `json:"type"`
	Text     string                 `json:"text"`
	TapURL   *app_url.SpruceAction  `json:"tap_url"`
	Pharmacy *pharmacy.PharmacyData `json:"pharmacy"`
}

func (v *tpPharmacyView) Validate() error {
	v.Type = treatmentViewNamespace + ":pharmacy"
	return nil
}

func (v *tpPharmacyView) TypeName() string {
	return v.Type
}

type tpPrescriptionView struct {
	Type                 string               `json:"type"`
	IconURL              *app_url.SpruceAsset `json:"icon_url"`
	Title                string               `json:"title"`
	Description          string               `json:"description"`
	SmallHeaderText      string               `json:"small_header_text"`
	Timestamp            *time.Time           `json:"timestamp,omitempty"`
	SmallHeaderHasTokens bool                 `json:"small_header_text_has_tokens"`
	Buttons              []tpView             `json:"buttons,omitempty"`
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

func (v *tpPrescriptionView) TypeName() string {
	return v.Type
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

func (v *tpPrescriptionButtonView) TypeName() string {
	return v.Type
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

func (v *tpButtonFooterView) TypeName() string {
	return v.Type
}
