package client

import (
	"encoding/json"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/mapstructure"
)

func Decode(answersJSON string) (map[string]Answer, error) {
	var questionToAnswerMap map[string]Answer
	decoderConfig := &mapstructure.DecoderConfig{
		Result:   &questionToAnswerMap,
		TagName:  "json",
		Registry: *typeRegistry,
	}

	d, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var jsonMap map[string]interface{}
	if err := json.Unmarshal([]byte(answersJSON), &jsonMap); err != nil {
		return nil, errors.Trace(err)
	}

	if err := d.Decode(jsonMap); err != nil {
		return nil, errors.Trace(err)
	}
	return questionToAnswerMap, nil
}
