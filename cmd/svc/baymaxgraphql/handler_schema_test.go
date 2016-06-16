package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

func TestSchemaHandler(t *testing.T) {
	h := newSchemaHandler()
	w := httptest.NewRecorder()
	h.ServeHTTP(nil, w, nil)
	test.HTTPResponseCode(t, http.StatusOK, w)
	if os.Getenv("GRAPHQL_SCHEMA") != "" {
		fmt.Print(w.Body.String())
	}
}
