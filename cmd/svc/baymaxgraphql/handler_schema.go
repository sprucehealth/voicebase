package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqlintrospect"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/graphql"
)

type schemaHandler struct{}

func newSchemaHandler() httputil.ContextHandler {
	return schemaHandler{}
}

func (schemaHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	res := graphql.Do(graphql.Params{
		Schema:        gqlSchema,
		RequestString: gqlintrospect.Query,
	})
	w.Header().Set("Content-Type", "text/plain")
	if res.HasErrors() {
		writeInternalError(w, fmt.Errorf("%+v", res.Errors))
		return
	}
	b, err := json.Marshal(res.Data)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	var schema gqlintrospect.Schema
	if err := json.Unmarshal(b, &struct {
		Schema *gqlintrospect.Schema `json:"__schema"`
	}{
		Schema: &schema,
	}); err != nil {
		writeInternalError(w, err)
		return
	}
	if err := schema.Fdump(w); err != nil {
		writeInternalError(w, err)
	}
}

func writeInternalError(w http.ResponseWriter, err error) {
	golog.Errorf("Failed to generate schema: %s", err)
	http.Error(w, "Internal Error", http.StatusInternalServerError)
}
