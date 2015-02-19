package treatment_plan

import (
	"errors"
	"time"

	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/pharmacy"
	"github.com/sprucehealth/backend/views"
)

const (
	smallGrayStyle            = "small_gray"
	treatmentViewNamespace    = "treatment"
	captionRegularItalicStyle = "caption_regular_italic"
	bulletedStyle             = "bulleted"
	numberedStyle             = "numbered"
)

type tpLargeIconTextButtonView struct {
	Type       string                `json:"type"`
	Text       string                `json:"text"`
	IconURL    string                `json:"icon_url"`
	IconWidth  int                   `json:"icon_width"`
	IconHeight int                   `json:"icon_height"`
	TapURL     *app_url.SpruceAction `json:"tap_url"`
}

type tpHeroHeaderView struct {
	Type            string `json:"type"`
	Title           string `json:"title"`
	Subtitle        string `json:"subtitle"`
	CreatedDateText string `json:"created_date_text"`
}

func NewTPHeroHeaderView(title, subtitle string) views.View {
	return &tpHeroHeaderView{
		Title:    title,
		Subtitle: subtitle,
	}
}

type tpCardView struct {
	Type  string       `json:"type"`
	Views []views.View `json:"views"`
}

func NewTPCardView(views []views.View) views.View {
	return &tpCardView{Views: views}
}

type tpCardTitleView struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	IconURL     string `json:"icon_url"`
	RoundedIcon bool   `json:"rounded_icon,omitempty"`
}

func NewTPCardTitleView(title, iconURL string, roundedIcon bool) views.View {
	return &tpCardTitleView{
		Title:       title,
		IconURL:     iconURL,
		RoundedIcon: roundedIcon,
	}
}

type tpTextView struct {
	Type  string `json:"type"`
	Style string `json:"style"`
	Text  string `json:"text"`
}

func NewTPTextView(style, text string) views.View {
	return &tpTextView{
		Style: style,
		Text:  text,
	}
}

type tpTextDisclosureButtonView struct {
	Type   string                `json:"type"`
	Style  string                `json:"style"`
	Text   string                `json:"text"`
	TapURL *app_url.SpruceAction `json:"tap_url"`
}

type tpImageView struct {
	Type        string `json:"type"`
	ImageWidth  int    `json:"image_width"`
	ImageHeight int    `json:"image_height"`
	ImageURL    string `json:"image_url"`
	Insets      string `json:"insets"`
}

type tpIconTitleSubtitleView struct {
	Type     string               `json:"type"`
	IconURL  *app_url.SpruceAsset `json:"icon_url"`
	Title    string               `json:"title"`
	Subtitle string               `json:"subtitle"`
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

type tpSnippetDetailsView struct {
	Type    string `json:"type"`
	Snippet string `json:"snippet"`
	Details string `json:"details"`
}

type tpListElementView struct {
	Type         string `json:"type"`
	ElementStyle string `json:"element_style"` // numbered, dont
	Number       int    `json:"number,omitempty"`
	Text         string `json:"text"`
}

func NewTPListElement(elementStyle, text string, number int) views.View {
	return &tpListElementView{
		ElementStyle: elementStyle,
		Text:         text,
		Number:       number,
	}
}

type tpPlainButtonView struct {
	Type   string                `json:"type"`
	Text   string                `json:"text"`
	TapURL *app_url.SpruceAction `json:"tap_url"`
}

type tpButtonView struct {
	Type    string                `json:"type"`
	Text    string                `json:"text"`
	TapURL  *app_url.SpruceAction `json:"tap_url"`
	IconURL *app_url.SpruceAsset  `json:"icon_url"`
}

type tpPharmacyView struct {
	Type     string                 `json:"type"`
	Text     string                 `json:"text"`
	TapURL   *app_url.SpruceAction  `json:"tap_url"`
	Pharmacy *pharmacy.PharmacyData `json:"pharmacy"`
}

func NewPharmacyView(text string, tapURL *app_url.SpruceAction, pharmacy *pharmacy.PharmacyData) views.View {
	return &tpPharmacyView{
		Text:     text,
		TapURL:   tapURL,
		Pharmacy: pharmacy,
	}
}

type tpPrescriptionView struct {
	Type              string               `json:"type"`
	IconURL           *app_url.SpruceAsset `json:"icon_url"`
	IconWidth         int                  `json:"icon_width"`
	IconHeight        int                  `json:"icon_height"`
	Title             string               `json:"title"`
	Description       string               `json:"description"`
	Subtitle          string               `json:"subtitle"`
	SubtitleHasTokens bool                 `json:"subtitle_has_tokens"`
	Timestamp         *time.Time           `json:"timestamp,omitempty"`
	Buttons           []views.View         `json:"buttons,omitempty"`
}

type tpPrescriptionButtonView struct {
	Type    string                `json:"type"`
	Text    string                `json:"text"`
	IconURL *app_url.SpruceAsset  `json:"icon_url"`
	TapURL  *app_url.SpruceAction `json:"tap_url"`
}

type tpButtonFooterView struct {
	Type             string                `json:"type"`
	FooterText       string                `json:"footer_text"`
	ButtonText       string                `json:"button_text"`
	IconURL          *app_url.SpruceAsset  `json:"icon_url"`
	TapURL           *app_url.SpruceAction `json:"tap_url"`
	CenterFooterText bool                  `json:"center_footer_text"`
}

func (v *tpLargeIconTextButtonView) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	return nil
}

func (v *tpLargeIconTextButtonView) TypeName() string {
	return "large_icon_text_button"
}

func (v *tpHeroHeaderView) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	return nil
}

func (v *tpHeroHeaderView) TypeName() string {
	return "hero_header"
}

func (v *tpCardView) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	return views.Validate(v.Views, namespace)
}

func (v *tpCardView) TypeName() string {
	return "card_view"
}

func (v *tpCardTitleView) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	return nil
}

func (v *tpCardTitleView) TypeName() string {
	return "card_title_view"
}

func (v *tpTextDisclosureButtonView) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	return nil
}

func (v *tpTextDisclosureButtonView) TypeName() string {
	return "text_disclosure_button"
}

func (v *tpImageView) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	return nil
}

func (v *tpImageView) TypeName() string {
	return "image"
}

func (v *tpIconTitleSubtitleView) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	return nil
}

func (v *tpIconTitleSubtitleView) TypeName() string {
	return "icon_title_subtitle_view"
}

func (v *tpIconTextView) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	return nil
}

func (v *tpIconTextView) TypeName() string {
	return "icon_text_view"
}

func (v *tpSnippetDetailsView) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	return nil
}

func (v *tpSnippetDetailsView) TypeName() string {
	return "snippet_details"
}

func (v *tpListElementView) Validate(namespace string) error {
	if v.ElementStyle != bulletedStyle && v.ElementStyle != numberedStyle {
		return errors.New("ListElementView expects ElementStyle of numbered or bulleted, not " + v.ElementStyle)
	}
	v.Type = namespace + ":" + v.TypeName()
	return nil
}

func (v *tpListElementView) TypeName() string {
	return "list_element"
}

func (v *tpPlainButtonView) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	return nil
}

func (v *tpPlainButtonView) TypeName() string {
	return "plain_button"
}

func (v *tpButtonView) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	return nil
}

func (v *tpButtonView) TypeName() string {
	return "button"
}

func (v *tpPharmacyView) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	return nil
}

func (v *tpPharmacyView) TypeName() string {
	return "pharmacy"
}

func (v *tpPrescriptionView) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	return views.Validate(v.Buttons, namespace)
}

func (v *tpPrescriptionView) TypeName() string {
	return "prescription"
}

func (v *tpPrescriptionButtonView) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	return nil
}

func (v *tpPrescriptionButtonView) TypeName() string {
	return "prescription_button"
}

func (v *tpButtonFooterView) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	return nil
}

func (v *tpButtonFooterView) TypeName() string {
	return "button_footer"
}

func (v *tpTextView) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	return nil
}

func (v *tpTextView) TypeName() string {
	return "text"
}
