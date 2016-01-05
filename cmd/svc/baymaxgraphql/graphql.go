package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/libs/conc"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
)

type ctxKey int

const (
	ctxAccount ctxKey = 0
)

func ctxWithAccount(ctx context.Context, acc *account) context.Context {
	return context.WithValue(ctx, ctxAccount, acc)
}

// accountFromContext returns the account from the context which may be nil
func accountFromContext(ctx context.Context) *account {
	acc, _ := ctx.Value(ctxAccount).(*account)
	return acc
}

func serviceFromParams(p graphql.ResolveParams) *service {
	return p.Info.RootValue.(map[string]interface{})["service"].(*service)
}

func contextFromParams(p graphql.ResolveParams) context.Context {
	return p.Info.RootValue.(map[string]interface{})["context"].(context.Context)
}

var nodeInterfaceType = graphql.NewInterface(
	graphql.InterfaceConfig{
		Name: "Node",
		Fields: graphql.Fields{
			"id": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		},
	},
)

var gqlSchema graphql.Schema

func init() {
	var err error
	gqlSchema, err = graphql.NewSchema(
		graphql.SchemaConfig{
			Query:    queryType,
			Mutation: mutationType,
		},
	)
	if err != nil {
		panic(err)
	}
}

type user struct {
	account  *account
	email    string
	password string
}

type graphQLHandler struct {
	service *service
}

func NewGraphQL(authClient auth.AuthClient, directoryClient directory.DirectoryClient, threadingClient threading.ThreadsClient, exComms excomms.ExCommsClient) httputil.ContextHandler {
	return &graphQLHandler{
		service: &service{
			auth:      authClient,
			directory: directoryClient,
			threading: threadingClient,
			exComms:   exComms,
		},
	}
}

type gqlReq struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

const (
	authTokenCookieName       = "at"
	defaultAuthCookieDuration = time.Hour * 24 * 30
)

func setAuthCookie(w http.ResponseWriter, token string, expires time.Time) {
	if expires.IsZero() {
		expires = time.Now().Add(defaultAuthCookieDuration)
	}
	http.SetCookie(w, &http.Cookie{
		Name:   authTokenCookieName,
		Value:  token,
		Path:   "/",
		MaxAge: int(expires.Sub(time.Now()).Nanoseconds() / 1e9),
		// Secure: true, TODO
		HttpOnly: true,
	})
}

func removeAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   authTokenCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
		// Secure: true, TODO
		HttpOnly: true,
	})
}

func (h *graphQLHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// TODO: should set the deadline earlier in the HTTP handler stack
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var req gqlReq
	if r.Method == "GET" {
		req.Query = r.FormValue("query")
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Failed to decode body", http.StatusBadRequest)
			return
		}
	}

	var acc *account
	if c, err := r.Cookie(authTokenCookieName); err == nil && c.Value != "" {
		res, err := h.service.auth.CheckAuthentication(ctx,
			&auth.CheckAuthenticationRequest{Token: c.Value},
		)
		if err != nil {
			golog.Errorf("Failed to check auth token: %s", err)
		} else if res.IsAuthenticated {
			// If token changed then update the cookie
			if res.Token.Value != c.Value {
				var expires time.Time
				if res.Token.ExpirationEpoch > 0 {
					expires = time.Unix(int64(res.Token.ExpirationEpoch), 0)
				}
				setAuthCookie(w, res.Token.Value, expires)
			}
			acc = &account{
				ID: res.Account.ID,
			}
		} else {
			removeAuthCookie(w)
		}
	}

	ctx = ctxWithAccount(ctx, acc)

	result := conc.NewMap()
	response := graphql.Do(graphql.Params{
		Schema:         gqlSchema,
		RequestString:  req.Query,
		VariableValues: req.Variables,
		RootObject: map[string]interface{}{
			"context": ctx,
			"service": h.service,
			// result is used to pass values from the executor to the top level (e.g. auth token)
			"result": result,
		},
	})

	if token, ok := result.Get("auth_token").(string); ok {
		expires, _ := result.Get("auth_expiration").(time.Time)
		if expires.Before(time.Now()) {
			expires = time.Time{}
		}
		setAuthCookie(w, token, expires)
	}
	if unauth, ok := result.Get("unauthenticated").(bool); ok && unauth {
		removeAuthCookie(w)
	}

	httputil.JSONResponse(w, http.StatusOK, response)
}

// internalError logs the provided internal error and returns a sanitized
// versions since we don't want internal details leaking over graphql errors.
func internalError(err error) error {
	golog.LogDepthf(1, golog.ERR, err.Error())
	if environment.IsDev() {
		return fmt.Errorf("internal error: %s\n", err)
	}
	return errors.New("internal error") // TODO: attach request ID or error ID or something
}
