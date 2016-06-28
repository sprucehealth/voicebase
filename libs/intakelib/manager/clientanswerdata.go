package manager

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

// clientAnswerData is used as a wrapper for the data to be communicated
// to the client.
type clientAnswerData struct {
	answerJSON   []byte
	questionID   string
	questionType string
}

func (c *clientAnswerData) marshalProtobuf() ([]byte, error) {
	qType := questionTypeToProtoBufType[c.questionType]
	if qType == nil {
		return nil, fmt.Errorf("Unable to determine protocol buffer type for question %s", c.questionType)
	}

	cd := &intake.ClientAnswerData{
		QuestionId:       proto.String(c.questionID),
		Type:             qType,
		ClientAnswerJson: c.answerJSON,
	}

	return proto.Marshal(cd)
}
