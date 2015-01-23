package views

import (
	"fmt"

	"github.com/sprucehealth/backend/app_url"
)

type TextStyle string

const (
	SectionHeaderStyle TextStyle = "section_header"
	SubheaderStyle     TextStyle = "subheader"
)

type View interface {
	TypeName() string
	Validate(namespace string) error
}

type Card struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	Views []View `json:"views"`
}

type CheckboxTextList struct {
	Type   string   `json:"type"`
	Titles []string `json:"titles"`
}

type FilledButton struct {
	Type   string                `json:"type"`
	Title  string                `json:"title"`
	TapURL *app_url.SpruceAction `json:"tap_url"`
}

type DoctorProfilePhotos struct {
	Type      string   `json:"type"`
	PhotoURLs []string `json:"photo_urls"`
}

type OutlinedButton struct {
	Type   string                `json:"type"`
	Title  string                `json:"title"`
	TapURL *app_url.SpruceAction `json:"tap_url"`
}

type BodyText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Text struct {
	Type  string    `json:"type"`
	Text  string    `json:"text"`
	Style TextStyle `json:"style,omitempty"`
}

type SmallDivider struct {
	Type string `json:"type"`
}

type LargeDivider struct {
	Type string `json:"type"`
}

func (v *Card) TypeName() string {
	return "card_view"
}

func (v *Card) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	if v.Title == "" {
		return fmt.Errorf("card_view.tile required")
	}
	return Validate(v.Views, namespace)
}

func (v *CheckboxTextList) TypeName() string {
	return "checkbox_text_list_view"
}

func (v *CheckboxTextList) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	if len(v.Titles) == 0 {
		return fmt.Errorf("checkbox_text_list_view.titled required and must not be empty")
	}
	return nil
}

func (v *FilledButton) TypeName() string {
	return "filled_button_view"
}

func (v *FilledButton) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	if v.Title == "" {
		return fmt.Errorf("filled_button_view.title required")
	}
	if v.TapURL == nil {
		return fmt.Errorf("filled_button_view.tap_url required")
	}
	return nil
}

func (v *OutlinedButton) TypeName() string {
	return "outlined_button_view"
}

func (v *OutlinedButton) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	if v.Title == "" {
		return fmt.Errorf("outlined_button_view.title required")
	}
	if v.TapURL == nil {
		return fmt.Errorf("outlined_button_view.tap_url required")
	}
	return nil
}

func (v *DoctorProfilePhotos) TypeName() string {
	return "doctor_profile_photos_view"
}

func (v *DoctorProfilePhotos) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	if len(v.PhotoURLs) == 0 {
		return fmt.Errorf("doctor_profile_photos_view.photo_urls required and may not be empty")
	}
	return nil
}

func (v *BodyText) TypeName() string {
	return "body_text_view"
}

func (v *BodyText) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	if v.Text == "" {
		return fmt.Errorf("body_text_view.text required")
	}
	return nil
}

func (v *Text) TypeName() string {
	return "text"
}

func (v *Text) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	if v.Text == "" {
		return fmt.Errorf("text_view.text required")
	}
	return nil
}

func (v *SmallDivider) TypeName() string {
	return "small_divider"
}

func (v *SmallDivider) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	return nil
}

func (v *LargeDivider) TypeName() string {
	return "large_divider"
}

func (v *LargeDivider) Validate(namespace string) error {
	v.Type = namespace + ":" + v.TypeName()
	return nil
}

func Validate(views []View, namespace string) error {
	for _, v := range views {
		if err := v.Validate(namespace); err != nil {
			return err
		}
	}
	return nil
}
