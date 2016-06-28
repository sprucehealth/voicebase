package manager

import (
	"strings"
	"unicode"
)

// topLevelAnswerItem is an interface conformed to by an object that represents a patient answer item
// as a container of subAnswers and subScreens. (eg. autocomplete answer item and multiple choice answer item)
type topLevelAnswerItem interface {
	text() string
	potentialAnswerID() string
	subAnswers() []patientAnswer
	setSubscreens([]screen)
	subscreens() []screen
}

// topLevelAnswerWithSubAnswersContainer is an interface conformed to
// by any patient answer that contains top level answer items
type topLevelAnswerWithSubScreensContainer interface {
	patientAnswer
	topLevelAnswers() []topLevelAnswerItem
}

type subscreensContainer interface {
	subscreens() []screen
}

// sanitizeQuestionID returns the server provided questionID without the prepended subquestion
// related information
func sanitizeQuestionID(questionID string) string {
	if idx := strings.IndexRune(questionID, '_'); idx >= 0 {
		return questionID[idx+1:]
	}
	return questionID
}

const (
	tokenParentAnswerText             = "<parent_answer_text>"
	tokenCapitalizedParentAnswerText  = "<capitalized_parent_answer_text>"
	tokenCapitalizedText              = "<capitalized_text>"
	tokenLowercaseParentAnswerText    = "<lowercase_parent_answer_text>"
	tokenLowercaseText                = "<lowercase_text>"
	tokenSentenceCaseParentAnswerText = "<sentence_case_parent_answer_text>"
	tokenSentenceCaseText             = "<sentence_case_text>"
)

var tokenTypeMapper = map[string]func(tokenType, str, val string) string{
	tokenParentAnswerText:             replaceText,
	tokenCapitalizedText:              replaceWithCapitalizedText,
	tokenCapitalizedParentAnswerText:  replaceWithCapitalizedText,
	tokenLowercaseText:                replaceWithLowerCaseText,
	tokenLowercaseParentAnswerText:    replaceWithLowerCaseText,
	tokenSentenceCaseText:             replaceWithSentenceCaseText,
	tokenSentenceCaseParentAnswerText: replaceWithSentenceCaseText,
}

// replaceText returns a string with the specified token
// replaced for the provided value.
func replaceText(tokenType, str, val string) string {
	return strings.Replace(str, tokenType, val, -1)
}

// replaceWithLowerCaseText returns a string with the specified
// token replaced for a lowercase representation of the provided value.
func replaceWithLowerCaseText(tokenType, str, val string) string {
	return strings.Replace(str, tokenType, strings.ToLower(val), -1)
}

// replaceWithCapitalizedText returns a string with the specified
// token replaced for a first-letter-of-each-word capitalized representation of
// the provided value.
func replaceWithCapitalizedText(tokenType, str, val string) string {
	titleLetter := true

	return strings.Replace(str, tokenType,
		strings.Map(func(r rune) rune {
			if titleLetter && r != ' ' {
				titleLetter = false
				return unicode.ToTitle(r)
			}

			if r == ' ' {
				titleLetter = true
			} else {
				titleLetter = false
			}

			return unicode.ToLower(r)
		}, val), -1)
}

// replaceWithSentenceCaseText returns a string with the specified
// token replaced for a capitalized-first-letter-of-the-string
// representation of the provided value.
func replaceWithSentenceCaseText(tokenType, str, val string) string {
	var firstLetterSeen bool

	return strings.Replace(str, tokenType,
		strings.Map(func(r rune) rune {
			if !firstLetterSeen && r != ' ' {
				firstLetterSeen = true
				return unicode.ToTitle(r)
			}

			return unicode.ToLower(r)
		}, val), -1)
}

// processTokenInString determines the tokenType
// present in the string to use the appropriate replacer
// function to process the token and return a result.
// If no token is found, the string is returned as is.
func processTokenInString(str, value string) string {
	for tokenType, replacer := range tokenTypeMapper {
		if strings.Contains(str, tokenType) {
			return replacer(tokenType, str, value)
		}
	}
	return str
}
