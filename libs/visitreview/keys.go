package visitreview

func AnswersKey(tag string) string {
	return tag + ":answers"
}

func QuestionSummaryKey(tag string) string {
	return tag + ":question_summary"
}

func EmptyStateTextKey(tag string) string {
	return tag + ":empty_state_text"
}

func PhotosKey(tag string) string {
	return tag + ":photos"
}
