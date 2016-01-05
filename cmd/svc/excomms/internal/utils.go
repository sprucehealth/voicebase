package internal

import (
	"encoding/base64"
	"encoding/json"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"

	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
)

type marshaller interface {
	Marshal() ([]byte, error)
}

func publishToSNSTopic(snsCLI snsiface.SNSAPI, topic string, m marshaller) {
	conc.Go(func() {

		data, err := m.Marshal()
		if err != nil {
			golog.Errorf(err.Error())
			return
		}

		msgData, err := json.Marshal(&struct {
			Default string `json:"default"`
		}{
			Default: base64.StdEncoding.EncodeToString(data),
		})
		if err != nil {
			golog.Errorf(err.Error())
			return
		}

		_, err = snsCLI.Publish(&sns.PublishInput{
			Message:          ptr.String(string(msgData)),
			MessageStructure: ptr.String("json"),
			TopicArn:         ptr.String(topic),
		})
		if err != nil {
			golog.Errorf("Unable to publish message to topic: %s", err.Error())
			return
		}
	})
}
