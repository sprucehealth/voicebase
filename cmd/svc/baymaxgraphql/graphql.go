package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/samuel/go-metrics/metrics"
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
	lmedia "github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/layout"
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
		case *models.ProviderAccount:
			return providerAccountType
		case *models.PatientAccount:
			return patientAccountType
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
	auth               auth.AuthClient
	ram                raccess.ResourceAccessor
	service            *service
	statRequests       *metrics.Counter
	statResponseErrors *metrics.Counter
	statLatency        metrics.Histogram
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
	layout layout.LayoutClient,
	layoutStore layout.Storage,
	mediaSigner *media.Signer,
	emailDomain string,
	webDomain string,
	serviceNumber phone.Number,
	spruceOrgID string,
	staticURLPrefix string,
	segmentClient *analytics.Client,
	media *lmedia.Service,
	sns snsiface.SNSAPI,
	supportMessageTopicARN string,
	metricsRegistry metrics.Registry,
) httputil.ContextHandler {
	statRequests := metrics.NewCounter()
	statResponseErrors := metrics.NewCounter()
	statLatency := metrics.NewUnbiasedHistogram()
	metricsRegistry.Add("requests", statRequests)
	metricsRegistry.Add("response_errors", statResponseErrors)
	metricsRegistry.Add("latency_us", statLatency)
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
			layout:          layout,
			spruceOrgID:     spruceOrgID,
			staticURLPrefix: staticURLPrefix,
			segmentio:       &segmentIOWrapper{Client: segmentClient},
			media:           media,
			sns:             sns,
			supportMessageTopicARN: supportMessageTopicARN,
			layoutStore:            layoutStore,
		},
		statRequests:       statRequests,
		statResponseErrors: statResponseErrors,
		statLatency:        statLatency,
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
	h.statRequests.Inc(1)
	st := time.Now()
	defer func() {
		h.statLatency.Update(time.Since(st).Nanoseconds() / 1e3)
	}()

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

	var acc *auth.Account
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
			acc = res.Account
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

	// Since we are authenticated, cache a collection of entity information for the account orgs
	eMap, err := h.orgToEntityMapForAccount(ctx, acc)
	if err != nil {
		// Don't hard fail here since we want functionality not involving this context to still work
		golog.Errorf("Failed to collect org to entity map for account %s: %s", acc.ID, err)
	}
	ctx = gqlctx.WithAccountEntities(ctx, gqlctx.NewEntityCache(eMap))
	// Bootstrap the entity cache with our account entity information
	ctx = gqlctx.WithEntities(ctx, gqlctx.NewEntityCache(func(ini map[string]*directory.Entity) map[string]*directory.Entity {
		eCache := make(map[string]*directory.Entity, len(eMap))
		for _, e := range eMap {
			eCache[e.ID] = e
		}
		return eCache
	}(eMap)))

	requestID, err := idgen.NewID()
	if err != nil {
		golog.Errorf("failed to generate request ID: %s", err)
	}
	ctx = gqlctx.WithRequestID(ctx, requestID)
	ctx = gqlctx.WithSpruceHeaders(ctx, sHeaders)
	ctx = gqlctx.WithQuery(ctx, req.Query)

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

	if len(response.Errors) != 0 {
		h.statResponseErrors.Inc(1)
	}
	for i, e := range response.Errors {
		if e.StackTrace != "" {
			golog.Errorf("[%s] %s\n%s", e.Type, e.Message, e.StackTrace)
			// The stack trace shouldn't be serialized in the response but clear it out just to be sure
			e.StackTrace = ""
		} else {
			golog.Warningf("GraphQL error response %s: %s (%s)\n%s", e.Type, e.Message, e.UserMessage, req.Query)
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

func (h *graphQLHandler) orgToEntityMapForAccount(ctx context.Context, acc *auth.Account) (map[string]*directory.Entity, error) {
	if acc == nil {
		return make(map[string]*directory.Entity), nil
	}
	entities, err := h.ram.EntitiesForExternalID(
		ctx,
		acc.ID,
		[]directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		0,
		[]directory.EntityStatus{directory.EntityStatus_ACTIVE})
	if err != nil {
		return nil, errors.Trace(err)
	}
	entMap := make(map[string]*directory.Entity, len(entities))
	for _, ent := range entities {
		for _, membership := range ent.Memberships {
			if membership.Type == directory.EntityType_ORGANIZATION {
				entMap[membership.ID] = ent
			}
		}
	}
	return entMap, nil
}
