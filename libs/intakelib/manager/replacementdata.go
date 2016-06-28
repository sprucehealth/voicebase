package manager

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type idReplacementData struct {
	replacementData protobufUnmarshaller
}

func (i *idReplacementData) unmarshalProtobuf(data []byte) error {
	var id intake.IDReplacementData
	if err := proto.Unmarshal(data, &id); err != nil {
		return err
	}

	switch *id.Type {
	case intake.IDReplacementData_PHOTO_ID:
		i.replacementData = &photoIDReplacement{}
	default:
		return fmt.Errorf("Unable to determine type: %s", *id.Type)
	}

	return i.replacementData.unmarshalProtobuf(id.Data)
}
