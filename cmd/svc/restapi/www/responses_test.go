package www

import (
	"io/ioutil"
	"testing"
)

func TestInternalErrorTemplate(t *testing.T) {
	if err := errorTemplate.Execute(ioutil.Discard, &errorContext{Title: "Error", Message: ""}); err != nil {
		t.Fatal(err)
	}
}
