package kinesis

import (
	"bytes"
	"encoding/json"
	"net/http"

	"carefront/libs/aws"
)

const kinesisAPIVersion = "Kinesis_20131104."

type Kinesis struct {
	aws.Region
	Client *aws.Client
}

func (kin *Kinesis) Request(action Action, request, response interface{}) error {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	if err := enc.Encode(request); err != nil {
		return err
	}
	req, err := http.NewRequest("POST", kin.Region.KinesisEndpoint, buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", kinesisAPIVersion+string(action))
	res, err := kin.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return ParseErrorResponse(res)
	}
	// Some actions only use the StatusCode with no body
	if response == nil {
		return nil
	}

	dec := json.NewDecoder(res.Body)
	return dec.Decode(response)
}
