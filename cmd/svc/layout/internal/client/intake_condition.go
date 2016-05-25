package client

import (
	"github.com/sprucehealth/backend/saml"
	"github.com/sprucehealth/backend/svc/layout"
)

func transformCondition(cond *saml.Condition) *layout.Condition {
	if cond == nil {
		return nil
	}

	tCondition := &layout.Condition{
		Operation:          cond.Op,
		GenderField:        cond.Gender,
		QuestionID:         cond.Question,
		PotentialAnswersID: cond.PotentialAnswers,
		Operands:           make([]*layout.Condition, len(cond.Operands)),
		IntValue:           cond.IntValue,
		StringValue:        cond.StringValue,
		BoolValue:          cond.BoolValue,
		DataSource:         cond.DataSource,
	}

	for i, operand := range cond.Operands {
		tCondition.Operands[i] = transformCondition(operand)
	}

	return tCondition
}
