package www

import (
	"io/ioutil"
	"testing"
)

func TestInternalErrorTemplate(t *testing.T) {
	if err := internalErrorTemplate.Execute(ioutil.Discard, &internalErrorContext{Message: ""}); err != nil {
		t.Fatal(err)
	}
}
