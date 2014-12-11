package diagnosis

import (
	"reflect"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
)

type QuestionIntake []*info_intake.Question

func (q *QuestionIntake) TypeName() string {
	return "diagnosis:modifier_questions_array"
}

func (q *QuestionIntake) Questions() []*info_intake.Question {
	return []*info_intake.Question(*q)
}

func NewQuestionIntake(questions []*info_intake.Question) QuestionIntake {
	return QuestionIntake(questions)
}

func init() {
	registerDetailsType(&QuestionIntake{})
}

var DetailTypes = make(map[string]reflect.Type)

func registerDetailsType(m common.Typed) {
	DetailTypes[m.TypeName()] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(m)).Interface())
}
