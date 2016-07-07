package manager

// conditionType represents the supported conditions on the platform
type conditionType string

func (c conditionType) String() string {
	return string(c)
}

const (
	conditionTypeAnswerEqualsExact         conditionType = "answer_equals_exact"
	conditionTypeAnswerContainsAny         conditionType = "answer_contains_any"
	conditionTypeanswerContainsAll         conditionType = "answer_contains_all"
	conditionTypeIntegerEqualTo            conditionType = "integer_equal_to"
	conditionTypeIntegerLessThan           conditionType = "integer_less_than"
	conditionTypeIntegerLessThanEqualTo    conditionType = "integer_less_than_or_equal_to"
	conditionTypeIntegerGreaterThan        conditionType = "integer_greater_than"
	conditionTypeIntegerGreaterThanEqualTo conditionType = "integer_greater_than_or_equal_to"
	conditionTypeGenderEquals              conditionType = "gender_equals"
	conditionTypeAND                       conditionType = "and"
	conditionTypeOR                        conditionType = "or"
	conditionTypeNOT                       conditionType = "not"
	conditionTypeBooleanEquals             conditionType = "boolean_equals"
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
	mustRegisterCondition(conditionTypeIntegerEqualTo.String(), &integerCondition{})
	mustRegisterCondition(conditionTypeIntegerGreaterThan.String(), &integerCondition{})
	mustRegisterCondition(conditionTypeIntegerGreaterThanEqualTo.String(), &integerCondition{})
	mustRegisterCondition(conditionTypeIntegerLessThan.String(), &integerCondition{})
	mustRegisterCondition(conditionTypeIntegerLessThanEqualTo.String(), &integerCondition{})
	mustRegisterCondition(conditionTypeBooleanEquals.String(), &boolCondition{})
}
