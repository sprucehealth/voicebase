package tagging

import (
	"time"

	"github.com/sprucehealth/backend/cmd/svc/restapi/tagging/model"
	"github.com/sprucehealth/backend/libs/ptr"
)

func ApplyCaseTag(client Client, text string, caseID int64, trigger *time.Time, ops TaggingOption) error {
	_, err := client.InsertTagAssociation(&model.Tag{Text: text}, &model.TagMembership{
		CaseID:      ptr.Int64(caseID),
		TriggerTime: trigger,
		Hidden:      ops.Has(TOHidden),
	})
	return err
}
