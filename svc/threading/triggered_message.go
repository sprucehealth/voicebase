package threading

import (
	"fmt"

	"github.com/sprucehealth/backend/svc/directory"
)

func AwayMessageSubkey(postingEntityType directory.EntityType, threadType ThreadType, channel *Endpoint_Channel) string {
	key := fmt.Sprintf("%s:%s", postingEntityType, threadType)
	if channel != nil {
		key = fmt.Sprintf("%s:%s", key, channel)
	}
	return key
}

func WelcomeMessageSubkey(source *directory.EntitySource) string {
	return directory.FlattenEntitySource(source)
}
