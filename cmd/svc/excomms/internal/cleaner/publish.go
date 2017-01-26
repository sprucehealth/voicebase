package cleaner

import (
	"encoding/base64"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
)

// Publish posts an event to an SNS topic
func Publish(sn snsiface.SNSAPI, topicARN string, req *models.DeleteResourceRequest) {

	if environment.IsTest() {
		return
	}

	data, err := req.Marshal()
	if err != nil {
		golog.Errorf("failed to marshal data: %s", err)
		return
	}

	conc.Go(func() {
		if _, err := sn.Publish(&sns.PublishInput{
			Message:  ptr.String(base64.StdEncoding.EncodeToString(data)),
			TopicArn: ptr.String(topicARN),
		}); err != nil {
			golog.Errorf("failed to publish event: %s", err)
		}
	})
}
