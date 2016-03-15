package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/media"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
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
		case *models.Account:
			return accountType
		case *models.Entity:
			return entityType
		case *models.Organization:
			return organizationType
		case *models.SavedThreadQuery:
			return savedThreadQueryType
		case *models.Thread:
			return threadType
		case *models.ThreadItem:
			return threadItemType
		}
		return nil
	}
}

type graphQLHandler struct {
	auth    auth.AuthClient
	ram     raccess.ResourceAccessor
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
	invite invite.InviteClient,
	mediaSigner *media.Signer,
	emailDomain string,
	webDomain string,
	serviceNumber phone.Number,
	spruceOrgID string,
	staticURLPrefix string,
	segmentClient *analytics.Client,
) httputil.ContextHandler {
	return &graphQLHandler{
		auth: authClient,
		ram:  raccess.New(authClient, directoryClient, threadingClient, exComms),
		service: &service{
			notification:    notificationClient,
			mediaSigner:     mediaSigner,
			emailDomain:     emailDomain,
			webDomain:       webDomain,
			serviceNumber:   serviceNumber,
			settings:        settings,
			invite:          invite,
			spruceOrgID:     spruceOrgID,
			staticURLPrefix: staticURLPrefix,
			segmentio:       &segmentIOWrapper{Client: segmentClient},
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

var (
	ipAddressPrefixes = []string{
		// Wuhan, Hubei, China
		"27.16.107",
		// Huanggang, Hubei, China
		"121.62.232",
	}
)

func setAuthCookie(w http.ResponseWriter, domain, token string, expires time.Time) {
	idx := strings.IndexByte(domain, ':')
	if idx != -1 {
		domain = domain[:idx]
	}
	if expires.IsZero() {
		expires = time.Now().Add(defaultAuthCookieDuration)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     authTokenCookieName,
		Domain:   domain,
		Value:    token,
		Path:     "/",
		MaxAge:   int(expires.Sub(time.Now()).Nanoseconds() / 1e9),
		Secure:   !environment.IsDev(),
		HttpOnly: true,
	})
}

func removeAuthCookie(w http.ResponseWriter, domain string) {
	http.SetCookie(w, &http.Cookie{
		Name:     authTokenCookieName,
		Domain:   domain,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   !environment.IsDev(),
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

	sHeaders := device.ExtractSpruceHeaders(w, r)
	remoteAddr := remoteAddrFromRequest(r, *flagBehindProxy)

	var acc *models.Account
	if c, err := r.Cookie(authTokenCookieName); err == nil && c.Value != "" {
		res, err := h.auth.CheckAuthentication(ctx,
			&auth.CheckAuthenticationRequest{
				Token: c.Value,
			},
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
			acc = &models.Account{
				ID: res.Account.ID,
			}
			ctx = gqlctx.WithClientEncryptionKey(ctx, res.Token.ClientEncryptionKey)
			ctx = gqlctx.WithAuthToken(ctx, c.Value)
		} else {
			removeAuthCookie(w, r.Host)
		}
	}

	// The account needs to exist in the context even when not authenticated. This is
	// so that if the request is a mutation that authenticates (authenticate, createAccount)
	// then the account can be updated in the context.
	ctx = gqlctx.WithAccount(ctx, acc)

	requestID, err := idgen.NewID()
	if err != nil {
		golog.Errorf("failed to generate request ID: %s", err)
	}
	ctx = gqlctx.WithRequestID(ctx, requestID)
	ctx = gqlctx.WithSpruceHeaders(ctx, sHeaders)

	result := conc.NewMap()
	response := graphql.Do(graphql.Params{
		Schema:         gqlSchema,
		RequestString:  req.Query,
		VariableValues: req.Variables,
		Context:        ctx,
		RootObject: map[string]interface{}{
			"service":    h.service,
			"remoteAddr": remoteAddr,
			"userAgent":  r.UserAgent(),
			// result is used to pass values from the executor to the top level (e.g. auth token)
			"result": result,
			// ram represents the resource access manager that fetches remote resources that require authorization
			raccess.ParamKey: h.ram,
		},
	})

	for i, e := range response.Errors {
		if e.StackTrace != "" {
			golog.Errorf("[%s] %s\n%s", e.Type, e.Message, e.StackTrace)
			// The stack trace shouldn't be serialized in the response but clear it out just to be sure
			e.StackTrace = ""
		}
		// Wrap any non well formed gql errors as internal
		if errors.Type(e) == errors.ErrTypeUnknown {
			response.Errors[i] = errors.InternalError(ctx, e).(gqlerrors.FormattedError)
		}
	}

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
