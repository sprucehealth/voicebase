package saml

import (
	"fmt"
	"reflect"
	"testing"
)

func TestClone(t *testing.T) {
	scr := &Screen{
		Title: "simple string",
		Condition: &Condition{
			Op: "and",
			Operands: []*Condition{
				{Op: "answer_contains_any", Question: "one", PotentialAnswers: []string{"abc", "123"}},
				{Op: "answer_contains_any", Question: "two", PotentialAnswers: []string{"xyz", "789"}},
			},
		},
		Questions: []*Question{
			{
				Details: &QuestionDetails{
					Text: "Hello. Is there anybody out there?",
				},
			},
		},
	}
	scr2 := clone(scr).(*Screen)
	if !reflect.DeepEqual(scr, scr2) {
		t.Fatalf("Not equal:\n%+v\n%+v", scr, scr2)
	}
	if scr.Condition == scr2.Condition {
		t.Fatal("Conditions pointers should not match but they do.")
	}
	if fmt.Sprintf("%p", scr.Questions) == fmt.Sprintf("%p", scr2.Questions) {
		t.Fatal("Questions slice pointers should not match but they do.")
	}
}
