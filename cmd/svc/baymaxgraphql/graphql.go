package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/media"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
)

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
	// This is done here rather than at declaration time to avoid an unresolvable compile time decleration loop
	nodeInterfaceType.ResolveType = func(value interface{}, info graphql.ResolveInfo) *graphql.Object {
		switch value.(type) {
		case *account:
			return accountType
		case *entity:
			return entityType
		case *organization:
			return organizationType
		case *savedThreadQuery:
			return savedThreadQueryType
		case *thread:
			return threadType
		case *threadItem:
			return threadItemType
		}
		return nil
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

// NewGraphQL returns an initialized instance of graphQLHandler
func NewGraphQL(
	authClient auth.AuthClient,
	directoryClient directory.DirectoryClient,
	threadingClient threading.ThreadsClient,
	exComms excomms.ExCommsClient,
	notificationClient notification.Client,
	settings settings.SettingsClient,
	mediaSigner *media.Signer,
	emailDomain string,
	serviceNumber phone.Number) httputil.ContextHandler {
	return &graphQLHandler{
		service: &service{
			auth:          authClient,
			directory:     directoryClient,
			threading:     threadingClient,
			exComms:       exComms,
			notification:  notificationClient,
			mediaSigner:   mediaSigner,
			emailDomain:   emailDomain,
			serviceNumber: serviceNumber,
			settings:      settings,
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

func setAuthCookie(w http.ResponseWriter, domain, token string, expires time.Time) {
	if expires.IsZero() {
		expires = time.Now().Add(defaultAuthCookieDuration)
	}
	http.SetCookie(w, &http.Cookie{
		Name:   authTokenCookieName,
		Domain: domain,
		Value:  token,
		Path:   "/",
		MaxAge: int(expires.Sub(time.Now()).Nanoseconds() / 1e9),
		// Secure: true, TODO
		HttpOnly: true,
	})
}

func removeAuthCookie(w http.ResponseWriter, domain string) {
	http.SetCookie(w, &http.Cookie{
		Name:   authTokenCookieName,
		Domain: domain,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
		// Secure: true, TODO
		HttpOnly: true,
	})
}

func (h *graphQLHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// TODO: should set the deadline earlier in the HTTP handler stack
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
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
				setAuthCookie(w, r.Host, res.Token.Value, expires)
			}
			acc = &account{
				ID: res.Account.ID,
			}
		} else {
			removeAuthCookie(w, r.Host)
		}
	}

	ctx = ctxWithAccount(ctx, acc)

	sHeaders := apiservice.ExtractSpruceHeaders(r)
	ctx = ctxWithSpruceHeaders(ctx, sHeaders)

	result := conc.NewMap()
	response := graphql.Do(graphql.Params{
		Schema:         gqlSchema,
		RequestString:  req.Query,
		VariableValues: req.Variables,
		Context:        ctx,
		RootObject: map[string]interface{}{
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
		setAuthCookie(w, r.Host, token, expires)
	}
	if unauth, ok := result.Get("unauthenticated").(bool); ok && unauth {
		removeAuthCookie(w, r.Host)
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
