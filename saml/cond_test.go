package saml

import (
	"reflect"
	"testing"
)

func TestConditionTokenize(t *testing.T) {
	tokens := tokenizeCondition("One and (Two or not Three)")
	exp := []string{"One", "and", "(", "Two", "or", "not", "Three", ")"}
	if !reflect.DeepEqual(tokens, exp) {
		t.Logf("Expected %+v got %+v", exp, tokens)
	}
}

func TestConditionParsing(t *testing.T) {
	one := &Condition{Op: "answer_contains_any", Question: "one", PotentialAnswers: []string{"aaa", "bbb"}}
	two := &Condition{Op: "answer_contains_any", Question: "two", PotentialAnswers: []string{"111"}}
	three := &Condition{Op: "answer_contains_any", Question: "three", PotentialAnswers: []string{"zzz"}}
	p := &parser{
		cTagsUsed: make(map[string]bool),
		cond: map[string]*Condition{
			"One":   one,
			"Two":   two,
			"Three": three,
		},
	}
	cond, targets := p.parseCondition("(One) and (Two or not Three) → Somewhere, Out-there")
	expTargets := []string{"Somewhere", "Out-there"}
	if !reflect.DeepEqual(targets, expTargets) {
		t.Errorf("Expected targets %+v got %+v", expTargets, targets)
	}
	expCond := "((one any [aaa, bbb]) AND ((two any [111]) OR (NOT (three any [zzz]))))"
	if c := cond.String(); c != expCond {
		t.Errorf("Expected condition '%s' got '%s'", expCond, c)
	}
}

func TestAgeConditionParsing(t *testing.T) {
	one := &Condition{Op: "answer_contains_any", Question: "one", PotentialAnswers: []string{"aaa", "bbb"}}
	two := &Condition{Op: "answer_contains_any", Question: "two", PotentialAnswers: []string{"111"}}
	three := &Condition{Op: "answer_contains_any", Question: "three", PotentialAnswers: []string{"zzz"}}
	p := &parser{
		cTagsUsed: make(map[string]bool),
		cond: map[string]*Condition{
			"One":   one,
			"Two":   two,
			"Three": three,
		},
	}
	cond, targets := p.parseCondition("(not Three) and (age.years > 15) and (Two or age.years < 30) and (not age.years = 29 or (age.years >= 27 and age.years <= 28))→ Somewhere, Out-there")
	expTargets := []string{"Somewhere", "Out-there"}
	if !reflect.DeepEqual(targets, expTargets) {
		t.Errorf("Expected targets %+v got %+v", expTargets, targets)
	}
	expCond := "((NOT (three any [zzz])) AND ((age_in_years > 15) AND (((two any [111]) OR (age_in_years < 30)) AND (NOT ((age_in_years == 29) OR ((age_in_years >= 27) AND (age_in_years <= 28)))))))"
	if c := cond.String(); c != expCond {
		t.Errorf("Expected condition '%s' got '%s'", expCond, c)
	}
}
