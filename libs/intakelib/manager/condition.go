package manager

// conditionType represents the supported conditions on the platform
type conditionType string

func (c conditionType) String() string {
	return string(c)
}

const (
	conditionTypeAnswerEqualsExact conditionType = "answer_equals_exact"
	conditionTypeAnswerContainsAny conditionType = "answer_contains_any"
	conditionTypeanswerContainsAll conditionType = "answer_contains_all"
	conditionTypeGenderEquals      conditionType = "gender_equals"
	conditionTypeAND               conditionType = "and"
	conditionTypeOR                conditionType = "or"
	conditionTypeNOT               conditionType = "not"
)

func init() {
	// register all supported conditions so that they can be used to determine the concrete condition type
	// to use when unmarshalling any layout object.
	mustRegisterCondition(conditionTypeanswerContainsAll.String(), &answerContainsAllCondition{})
	mustRegisterCondition(conditionTypeAnswerContainsAny.String(), &answerContainsAnyCondition{})
	mustRegisterCondition(conditionTypeAnswerEqualsExact.String(), &answerEqualsExactCondition{})
	mustRegisterCondition(conditionTypeGenderEquals.String(), &genderCondition{})
	mustRegisterCondition(conditionTypeAND.String(), &andCondition{})
	mustRegisterCondition(conditionTypeOR.String(), &orCondition{})
	mustRegisterCondition(conditionTypeNOT.String(), &notCondition{})

}
