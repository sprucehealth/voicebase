package sns

import (
	"encoding/base64"

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

	_, err = snsCLI.Publish(&sns.PublishInput{
		Message:  ptr.String(base64.StdEncoding.EncodeToString(data)),
		TopicArn: ptr.String(topic),
	})
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}
