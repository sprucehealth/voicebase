package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	gqlsettings "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/settings"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/directory/cache"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/patientsync"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
	"github.com/sprucehealth/graphql/language/parser"
	"github.com/sprucehealth/graphql/language/source"
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
		if value == nil {
			return nil
		}

		switch value.(type) {
		case *models.Call:
			return callType
		case *models.CarePlan:
			return carePlanType
		case *models.Entity:
			return entityType
		case *models.Organization:
			return organizationType
		case *models.PaymentRequest:
			return paymentRequestType
		case *models.Profile:
			return profileType
		case *models.ProviderAccount:
			return providerAccountType
		case *models.PatientAccount:
			return patientAccountType
		case *models.SavedThreadQuery:
			return savedThreadQueryType
		case *models.Thread:
			return threadType
		case *models.ThreadItem:
			return threadItemType
		case *models.VisitCategory:
			return visitCategoryType
		case *models.VisitLayout:
			return visitLayoutType
		case *models.VisitLayoutVersion:
			return visitLayoutVersionType
		case *models.Visit:
			return visitType
		}
		panic(fmt.Sprintf("Unknown type for value: %T", value))
	}
}

type graphQLHandler struct {
	auth                    auth.AuthClient
	ram                     raccess.ResourceAccessor
	service                 *service
	statRequests            *metrics.Counter
	statResponseErrors      *metrics.Counter
	statErrNotAuthorized    *metrics.Counter
	statErrNotAuthenticated *metrics.Counter
	statLatency             metrics.Histogram
	statGQLParseLatency     metrics.Histogram
	statGQLValidateLatency  metrics.Histogram
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
	care care.CareClient,
	media media.MediaClient,
	payments payments.PaymentsClient,
	patientSyncClient patientsync.PatientSyncClient,
	layoutStore layout.Storage,
	emailDomain string,
	webDomain string,
	mediaAPIDomain string,
	inviteAPIDomain string,
	serviceNumber phone.Number,
	spruceOrgID string,
	staticURLPrefix string,
	sns snsiface.SNSAPI,
	supportMessageTopicARN string,
	emailTemplateIDs emailTemplateIDs,
	metricsRegistry metrics.Registry,
	transactionalEmailSender string,
	stripeConnectURL string,
	hintConnectURL string,
) http.Handler {
	statRequests := metrics.NewCounter()
	statResponseErrors := metrics.NewCounter()
	statErrNotAuthorized := metrics.NewCounter()
	statErrNotAuthenticated := metrics.NewCounter()
	statLatency := metrics.NewUnbiasedHistogram()
	statGQLParseLatency := metrics.NewUnbiasedHistogram()
	statGQLValidateLatency := metrics.NewUnbiasedHistogram()
	metricsRegistry.Add("requests", statRequests)
	metricsRegistry.Add("response_errors", statResponseErrors)
	metricsRegistry.Add("user_error/not_authorized", statErrNotAuthorized)
	metricsRegistry.Add("user_error/not_authenticated", statErrNotAuthenticated)
	metricsRegistry.Add("latency_us", statLatency)
	metricsRegistry.Add("gql_parse_latency_us", statGQLParseLatency)
	metricsRegistry.Add("gql_validate_latency_us", statGQLValidateLatency)
	return &graphQLHandler{
		auth: authClient,
		ram:  raccess.New(authClient, directoryClient, threadingClient, exComms, layout, care, media, payments, patientSyncClient),
		service: &service{
			notification:     notificationClient,
			emailDomain:      emailDomain,
			webDomain:        webDomain,
			mediaAPIDomain:   mediaAPIDomain,
			inviteAPIDomain:  inviteAPIDomain,
			serviceNumber:    serviceNumber,
			settings:         settings,
			invite:           invite,
			layout:           layout,
			care:             care,
			spruceOrgID:      spruceOrgID,
			stripeConnectURL: stripeConnectURL,
			hintConnectURL:   hintConnectURL,
			staticURLPrefix:  staticURLPrefix,
			sns:              sns,
			supportMessageTopicARN:   supportMessageTopicARN,
			layoutStore:              layoutStore,
			emailTemplateIDs:         emailTemplateIDs,
			transactionalEmailSender: transactionalEmailSender,
		},
		statRequests:            statRequests,
		statResponseErrors:      statResponseErrors,
		statErrNotAuthorized:    statErrNotAuthorized,
		statErrNotAuthenticated: statErrNotAuthenticated,
		statLatency:             statLatency,
		statGQLParseLatency:     statGQLParseLatency,
		statGQLValidateLatency:  statGQLValidateLatency,
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

func rootDomain(host string) string {
	domain := host
	idx := strings.IndexByte(domain, '.')
	if idx != -1 {
		domain = domain[idx:]
	}
	idx = strings.IndexByte(domain, ':')
	if idx != -1 {
		domain = domain[:idx]
	}

	return domain
}

func setAuthCookie(w http.ResponseWriter, domain, token string, expires time.Time) {
	if expires.IsZero() {
		expires = time.Now().Add(defaultAuthCookieDuration)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     authTokenCookieName,
		Domain:   rootDomain(domain),
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
		Domain:   rootDomain(domain),
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   !environment.IsDev(),
		HttpOnly: true,
	})
}

func (h *graphQLHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
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

	logCtx := []interface{}{
		"Query", req.Query,
		"AppType", sHeaders.AppType,
		"Platform", sHeaders.Platform,
		"DeviceID", sHeaders.DeviceID,
	}
	if acc != nil {
		logCtx = append(logCtx, "AccountID", acc.ID)
	}
	if sHeaders.AppVersion != nil {
		logCtx = append(logCtx, "AppVersion", sHeaders.AppVersion.String())
	}
	ctx = golog.WithLogger(ctx, golog.ContextLogger(ctx).Context(logCtx...))

	httputil.CtxLogMap(ctx).Transact(func(m map[interface{}]interface{}) {
		m["Query"] = req.Query
		if acc != nil {
			m["AccountID"] = acc.ID
		}
		m["AppType"] = sHeaders.AppType
		m["Platform"] = sHeaders.Platform
		if sHeaders.AppVersion != nil {
			m["AppVersion"] = sHeaders.AppVersion.String()
		}
	})
	// Bootstrap the entity cache
	ctx = cache.InitEntityCache(ctx)
	ctx = settings.InitContextCache(ctx)
	ctx = devicectx.WithSpruceHeaders(ctx, sHeaders)
	ctx = gqlctx.WithQuery(ctx, req.Query)
	// TODO: the video calling feature flags rely on the provider only being part of one organization. This
	//       may not be true in the future. The problem however, is that there's no context of what the org
	//       is when this flag is needed so this is the simplest place to have this without major modifications
	//       to the structure of this service.
	ctx = gqlctx.WithLazyFeature(ctx, gqlctx.VideoCalling, func(ctx context.Context) bool {
		acc := gqlctx.Account(ctx)
		if acc == nil || acc.Type != auth.AccountType_PROVIDER {
			return false
		}

		eres, err := h.ram.Entities(ctx, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: acc.ID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth:             0,
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
			},
			RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
			ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
		}, raccess.EntityQueryOptionUnathorized)
		if err != nil {
			golog.ContextLogger(ctx).Errorf("Failed to lookup entities for account %s: %s", acc.ID, err)
			return false
		}
		if len(eres) != 1 {
			golog.ContextLogger(ctx).Errorf("No entities found for account %s", acc.ID)
			return false
		}

		var org *directory.Entity
		for _, em := range eres[0].Memberships {
			if em.Type == directory.EntityType_ORGANIZATION {
				org = em
				break
			}
		}
		if org == nil {
			golog.ContextLogger(ctx).Errorf("No org found for account %s entity %s", acc.ID, eres[0].ID)
			return false
		}

		res, err := h.service.settings.GetValues(ctx, &settings.GetValuesRequest{
			NodeID: org.ID,
			Keys:   []*settings.ConfigKey{{Key: gqlsettings.ConfigKeyVideoCalling}},
		})
		if err != nil {
			golog.ContextLogger(ctx).Errorf("Failed to lookup setting %s for org %s: %s", gqlsettings.ConfigKeyVideoCalling, org.ID, err)
			return false
		}
		return res.Values[0].GetBoolean().Value
	})

	result := conc.NewMap()
	response := h.graphqlDo(graphql.Params{
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
		errorTypes := make([]string, len(response.Errors))
		for i, e := range response.Errors {
			errorTypes[i] = e.Type
			if e.StackTrace != "" {
				golog.ContextLogger(ctx).Errorf("[%s] %s\n%s", e.Type, e.Message, e.StackTrace)
				// The stack trace shouldn't be serialized in the response but clear it out just to be sure
				e.StackTrace = ""
			} else {
				golog.ContextLogger(ctx).Warningf("GraphQL error response %s: %s (%s)", e.Type, e.Message, e.UserMessage)
			}
			switch errors.ErrorType(e.Type) {
			case errors.ErrTypeNotAuthorized:
				h.statErrNotAuthorized.Inc(1)
			case errors.ErrTypeNotAuthenticated:
				h.statErrNotAuthenticated.Inc(1)
			}
			// Wrap any non well formed gql errors as internal
			if errors.Type(e) == errors.ErrTypeUnknown {
				response.Errors[i] = errors.InternalError(ctx, e).(gqlerrors.FormattedError)
			}
		}
		httputil.CtxLogMap(ctx).Set("GraphQLErrors", strings.Join(errorTypes, " "))
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

func (h *graphQLHandler) orgToEntityMapForAccount(ctx context.Context, acc *auth.Account) (map[string][]*directory.Entity, error) {
	if acc == nil {
		return make(map[string][]*directory.Entity), nil
	}

	entities, err := h.ram.Entities(ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	entMap := make(map[string][]*directory.Entity, len(entities))
	for _, ent := range entities {
		for _, membership := range ent.Memberships {
			if membership.Type == directory.EntityType_ORGANIZATION {
				entMap[membership.ID] = []*directory.Entity{ent}
			}
		}
	}
	return entMap, nil
}

func (h *graphQLHandler) graphqlDo(p graphql.Params) *graphql.Result {
	st := time.Now()
	source := source.New("GraphQL request", p.RequestString)
	ast, err := parser.Parse(parser.ParseParams{Source: source})
	h.statGQLParseLatency.Update(time.Since(st).Nanoseconds() / 1e3)
	if err != nil {
		return &graphql.Result{
			Errors: gqlerrors.FormatErrors(err),
		}
	}
	st = time.Now()
	validationResult := graphql.ValidateDocument(&p.Schema, ast, nil)
	h.statGQLValidateLatency.Update(time.Since(st).Nanoseconds() / 1e3)
	if !validationResult.IsValid {
		return &graphql.Result{
			Errors: validationResult.Errors,
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
