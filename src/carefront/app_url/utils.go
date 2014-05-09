package app_url

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type SpruceUrl interface {
	unexportableInterface() bool
	String() string
	json.Marshaler
	json.Unmarshaler
}

const (
	spruceUrlScheme = "spruce:///"
	spruceImageUrl  = spruceUrlScheme + "image/"
	spruceActionUrl = spruceUrlScheme + "action/"
)

type SpruceAsset struct {
	Name string
}

func (s SpruceAsset) unexportableInterface() bool {
	return true
}

func (s SpruceAsset) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0)
	b = append(b, '"')
	b = append(b, []byte(spruceActionUrl)...)
	b = append(b, '/')
	b = append(b, []byte(s.Name)...)
	b = append(b, '"')

	return b, nil
}

func (s SpruceAsset) String() string {
	return fmt.Sprintf("%s/%s", spruceImageUrl, s.Name)
}

func (s SpruceAsset) UnmarshalJSON([]byte) error {
	return nil
}

type SpruceAction struct {
	ActionName string
	params     url.Values
}

func (s SpruceAction) unexportableInterface() bool {
	return true
}

func (s SpruceAction) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0)
	b = append(b, '"')
	b = append(b, []byte(spruceActionUrl)...)
	b = append(b, '/')
	b = append(b, []byte(s.ActionName)...)
	if len(s.params) == 0 {
		b = append(b, '?')
		b = append(b, []byte(s.params.Encode())...)
	}

	b = append(b, '"')

	return b, nil
}

func (s SpruceAction) String() string {
	return fmt.Sprintf("%s/%s?%s", spruceActionUrl, s.ActionName, s.params.Encode())
}

func (s SpruceAction) UnmarshalJSON([]byte) error {
	return nil
}
