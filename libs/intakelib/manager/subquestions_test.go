package manager

import "testing"

func TestTokenProcessing(t *testing.T) {

	checkExpectedValue(t, "Hi how are you spruce?", processTokenInString("Hi how are you <parent_answer_text>?", "spruce"))

	checkExpectedValue(t, "Hi how are you spruce?", processTokenInString("Hi how are you <lowercase_parent_answer_text>?", "SpRUcE"))
	checkExpectedValue(t, "Hi how are you spruce?", processTokenInString("Hi how are you <lowercase_text>?", "SpRUcE"))

	checkExpectedValue(t, "Hi how are you Doing Today Spruce?", processTokenInString("Hi how are you <capitalized_parent_answer_text>", "DOIng today sPruce?"))
	checkExpectedValue(t, "Hi how are you Doing Today Spruce?", processTokenInString("Hi how are you <capitalized_text>", "DOIng today sPruce?"))

	checkExpectedValue(t, "Hi how are you Doing    Today    Spruce?", processTokenInString("Hi how are you <capitalized_parent_answer_text>", "DOIng    today    sPruce?"))
	checkExpectedValue(t, "Hi how are you Doing    Today    Spruce?", processTokenInString("Hi how are you <capitalized_text>", "DOIng    today    sPruce?"))

	checkExpectedValue(t, "Hi how are you Doing today spruce?", processTokenInString("Hi how are you <sentence_case_parent_answer_text>", "dOING TODAY sPruce?"))
	checkExpectedValue(t, "Hi how are you Doing today spruce?", processTokenInString("Hi how are you <sentence_case_text>", "dOING TODAY sPruce?"))

	checkExpectedValue(t, "Hi how are you <dkghag>", processTokenInString("Hi how are you <dkghag>", "today"))
}

func checkExpectedValue(t *testing.T, expected, actual string) {
	if actual != expected {
		t.Fatalf("Expected: %s, got: %s", expected, actual)
	}
}
