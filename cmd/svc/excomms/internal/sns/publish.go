package sns

import (
	"encoding/base64"
	"encoding/json"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/ptr"
)

type marshaller interface {
	Marshal() ([]byte, error)
}

func Publish(snsCLI snsiface.SNSAPI, topic string, m marshaller) error {
	data, err := m.Marshal()
	if err != nil {
		return errors.Trace(err)
	}

	msgData, err := json.Marshal(&struct {
		Default string `json:"default"`
	}{
		Default: base64.StdEncoding.EncodeToString(data),
	})
	if err != nil {
		return errors.Trace(err)
	}

	_, err = snsCLI.Publish(&sns.PublishInput{
		Message:          ptr.String(string(msgData)),
		MessageStructure: ptr.String("json"),
		TopicArn:         ptr.String(topic),
	})
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}
