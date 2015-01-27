package handlers

import "testing"

func TestCodeResemblance(t *testing.T) {
	checkQuery("l", false, t)
	checkQuery("L", false, t)
	checkQuery("", false, t)

	checkQuery("v8", true, t)
	checkQuery("V89", true, t)
	checkQuery("V89.", true, t)
	checkQuery("V89.1", true, t)
	checkQuery("V89.1X", true, t)
	checkQuery("V89.1XX", true, t)
	checkQuery("V89.1XXA", true, t)
	checkQuery("V89.1xxA", true, t)
	checkQuery("V89.1xx9", true, t)

	checkQuery("L9.", false, t)
	checkQuery("L9.X", false, t)
	checkQuery("L9.XT", false, t)
	checkQuery("V89.1xx99", false, t)
}

func checkQuery(query string, expected bool, t *testing.T) {
	if resemblesCode(query) != expected {
		t.Fatalf("Expected %s to return %#v but returned %#v", query, expected, resemblesCode(query))
	}
}
