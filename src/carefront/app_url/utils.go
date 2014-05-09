package app_url

import (
	"encoding/json"
	"net/url"
)

type SpruceUrl interface {
	unexportableInterface() bool
	json.Marshaler
}

const (
	spruceUrlScheme = "spruce:///"
	spruceImageUrl  = spruceUrlScheme + "image/"
	spruceActionUrl = spruceUrlScheme + "action/"
)

type spruceAsset struct {
	Name string
}

func (s spruceAsset) unexportableInterface() bool {
	return true
}

func (s spruceAsset) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0)
	b = append(b, '"')
	b = append(b, []byte(spruceActionUrl)...)
	b = append(b, '/')
	b = append(b, []byte(s.Name)...)
	b = append(b, '"')

	return b, nil
}

type spruceAction struct {
	ActionName string
	params     url.Values
}

func (s spruceAction) unexportableInterface() bool {
	return true
}

func (s spruceAction) MarshalJSON() ([]byte, error) {
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
