package gql

import (
	"encoding/json"
	"net/http"
	"strings"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/query"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
	"github.com/sprucehealth/graphql/language/parser"
	"github.com/sprucehealth/graphql/language/source"
)

// TODO: Libify this aspect of the GQL stuff
type gqlReq struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type gqlHandler struct {
	behindProxy bool
	schema      graphql.Schema
}

// New returns an initialized instance of *gqlHandler
func New(behindProxy bool) (httputil.ContextHandler, graphql.Schema) {
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: query.NewRoot(),
	})
	if err != nil {
		golog.Fatalf("Failed to initialized gqlHandler: %s", err.Error())
	}
	return &gqlHandler{
		behindProxy: behindProxy,
		schema:      schema,
	}, schema
}

func (h *gqlHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// TODO: Libify this aspect of the GQL stuff
	var req gqlReq
	if r.Method == "GET" {
		req.Query = r.FormValue("query")
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Failed to decode body", http.StatusBadRequest)
			return
		}
	}

	result := conc.NewMap()
	response := h.graphqlDo(graphql.Params{
		Schema:         h.schema,
		RequestString:  req.Query,
		VariableValues: req.Variables,
		Context:        ctx,
		RootObject: map[string]interface{}{
			"remoteAddr": httputil.RemoteAddrFromRequest(r, h.behindProxy),
			"userAgent":  r.UserAgent(),
			"result":     result,
		},
	})

	if len(response.Errors) != 0 {
		errorTypes := make([]string, len(response.Errors))
		for i, e := range response.Errors {
			errorTypes[i] = e.Type
			if e.StackTrace != "" {
				golog.Errorf("[%s] %s\n%s", e.Type, e.Message, e.StackTrace)
				// The stack trace shouldn't be serialized in the response but clear it out just to be sure
				e.StackTrace = ""
			} else {
				golog.Warningf("GraphQL error response %s: %s (%s)\n%s", e.Type, e.Message, e.UserMessage, req.Query)
			}
			// TODO: Libify common aspects of errors baymaxgraphql internal package
			// Wrap any non well formed gql errors as internal
			// if errors.Type(e) == errors.ErrTypeUnknown {
			//	response.Errors[i] = errors.InternalError(ctx, e).(gqlerrors.FormattedError)
			// }
		}
		httputil.CtxLogMap(ctx).Set("GraphQLErrors", strings.Join(errorTypes, " "))
	}
	httputil.JSONResponse(w, http.StatusOK, response)
}

func (h *gqlHandler) graphqlDo(p graphql.Params) *graphql.Result {
	source := source.New("GraphQL request", p.RequestString)
	ast, err := parser.Parse(parser.ParseParams{Source: source})
	if err != nil {
		return &graphql.Result{
			Errors: gqlerrors.FormatErrors(err),
		}
	}
	return graphql.Execute(graphql.ExecuteParams{
		Schema:        p.Schema,
		Root:          p.RootObject,
		AST:           ast,
		OperationName: p.OperationName,
		Args:          p.VariableValues,
		Context:       p.Context,
	})
}
