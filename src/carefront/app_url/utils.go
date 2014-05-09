package app_url

import (
	"carefront/libs/golog"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type spruceUrl interface {
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
	b = append(b, []byte(spruceImageUrl)...)
	b = append(b, []byte(s.Name)...)
	b = append(b, '"')

	return b, nil
}

func (s SpruceAsset) String() string {
	return fmt.Sprintf("%s/%s", spruceImageUrl, s.Name)
}

func (s *SpruceAsset) UnmarshalJSON(data []byte) error {
	if len(data) < 3 {
		return nil
	}
	incomingUrl := string(data[1 : len(data)-1])
	fmt.Println("incoming url " + incomingUrl)
	spruceUrlComponents, err := url.Parse(incomingUrl)
	if err != nil {
		golog.Errorf("Unable to parse url for spruce asset %s", err)
		return err
	}
	pathComponents := strings.Split(spruceUrlComponents.Path, "/")
	if len(pathComponents) < 3 {
		golog.Errorf("Unable to break path %#v into its components when attempting to unmarshal %s", pathComponents, incomingUrl)
		return nil
	}
	s.Name = pathComponents[2]
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
	b = append(b, []byte(s.ActionName)...)
	if len(s.params) > 0 {
		b = append(b, '?')
		b = append(b, []byte(s.params.Encode())...)
	}

	b = append(b, '"')

	return b, nil
}

func (s SpruceAction) String() string {
	return fmt.Sprintf("%s/%s?%s", spruceActionUrl, s.ActionName, s.params.Encode())
}

func (s *SpruceAction) UnmarshalJSON(data []byte) error {
	if len(data) < 3 {
		return nil
	}

	incomingUrl := string(data[1 : len(data)-1])
	fmt.Println("incoming url " + incomingUrl)
	spruceUrlComponents, err := url.Parse(incomingUrl)
	if err != nil {
		golog.Errorf("Unable to parse url for spruce action %s", err)
		return err
	}
	pathComponents := strings.Split(spruceUrlComponents.Path, "/")
	if len(pathComponents) < 3 {
		golog.Errorf("Unable to break path %#v into its components when attempting to unmarshal %s", pathComponents, incomingUrl)
		return nil
	}
	s.ActionName = pathComponents[2]

	s.params, err = url.ParseQuery(spruceUrlComponents.RawQuery)
	if err != nil {
		return err
	}
	fmt.Printf("%#v\n", s)
	return nil
}
