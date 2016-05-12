package client

import (
	"encoding/json"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/mapstructure"
)

func Decode(answersJSON string) (*VisitAnswers, error) {
	var visitAnswers VisitAnswers
	decoderConfig := &mapstructure.DecoderConfig{
		Result:   &visitAnswers,
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
	return &visitAnswers, nil
}
