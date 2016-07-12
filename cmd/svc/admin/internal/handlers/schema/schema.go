package schema

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqlintrospect"
	"github.com/sprucehealth/graphql"
)

// TODO: Libify this with the schema handler in baymaxgraphql

type schemaHandler struct {
	schema graphql.Schema
}

// New returns an iniitalized instance of schemaHandler
func New(schema graphql.Schema) http.Handler {
	return &schemaHandler{
		schema: schema,
	}
}

func (h *schemaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	res := graphql.Do(graphql.Params{
		Schema:        h.schema,
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
