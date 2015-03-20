package saml

import (
	"errors"
	"regexp"
	"strings"
)

var (
	reTagRemove = regexp.MustCompile(`[^\w\s-]`)
	reTagSpaces = regexp.MustCompile(`[-\s]+`)
)

func tagFromText(v string) string {
	v = reTagRemove.ReplaceAllString(v, "")
	v = strings.ToLower(v)
	v = reTagSpaces.ReplaceAllString(v, "_")
	return v
}

func boolPtr(b bool) *bool {
	return &b
}

func validateQuestion(q *Question) error {
	switch q.Details.Type {
	case "q_type_single_select", "q_type_multiple_choice":
		if q.Details.Summary == "" {
			return errors.New("missing summary text")
		}
		if len(q.Details.Answers) == 0 && len(q.Details.AnswerGroups) == 0 {
			return errors.New("missing potential answers")
		}
	case "q_type_free_text":
		if len(q.Details.Answers) != 0 || len(q.Details.AnswerGroups) != 0 {
			return errors.New("free text questions cannot have potential answers")
		}
	}
	return nil
}
