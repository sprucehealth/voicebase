package manager

func init() {
	mustRegisterAnswer(questionTypeAutocomplete.String(), &autocompleteAnswer{})
	mustRegisterAnswer(questionTypeFreeText.String(), &freeTextAnswer{})
	mustRegisterAnswer(questionTypeMedia.String(), &mediaSectionAnswer{})
	mustRegisterAnswer(questionTypeMultipleChoice.String(), &multipleChoiceAnswer{})
	mustRegisterAnswer(questionTypeSegmentedControl.String(), &segmentedControlAnswer{})
	mustRegisterAnswer(questionTypeSingleEntry.String(), &singleEntryAnswer{})
	mustRegisterAnswer(questionTypeSingleSelect.String(), &singleSelectAnswer{})
}
