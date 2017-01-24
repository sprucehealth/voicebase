package raccess

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/patientsync"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// ParamKey is the name of the parameter index
const ParamKey = "ram"

// TODO: Proxy this in place of codes.NotFound
// ErrNotFound is returned if a result is expected. Can proxy codes.NotFound
var ErrNotFound = errors.New("baymaxgraphql/resource_accessor")

// ResourceAccess returns the RAL from the params
func ResourceAccess(p graphql.ResolveParams) ResourceAccessor {
	return p.Info.RootValue.(map[string]interface{})[ParamKey].(ResourceAccessor)
}

type resourceMap struct {
	rMap    map[string]map[string]struct{}
	rwMutex sync.RWMutex
}

func newResourceMap() *resourceMap {
	return &resourceMap{
		rMap: make(map[string]map[string]struct{}),
	}
}

func (m *resourceMap) Get(key string) map[string]struct{} {
	m.rwMutex.RLock()
	defer m.rwMutex.RUnlock()
	return m.rMap[key]
}

func (m *resourceMap) Set(resourceID string, orgIDs map[string]struct{}) {
	m.rwMutex.Lock()
	defer m.rwMutex.Unlock()
	m.rMap[resourceID] = orgIDs
}

// EntityQueryOption allows specifying of options when attempting to query
// for entities via the resource accessor
type EntityQueryOption int

const (
	// EntityQueryOptionUnathorized is an option used to skip authorization checks when
	// querying for entities
	EntityQueryOptionUnathorized EntityQueryOption = 1 << iota
)

type entityQueryOptions []EntityQueryOption

func (e entityQueryOptions) has(opt EntityQueryOption) bool {
	for _, o := range e {
		if o == opt {
			return true
		}
	}
	return false
}

// ResourceAccessor defines an interface for the retrieval and authorization of resources
type ResourceAccessor interface {
	Account(ctx context.Context, accountID string) (*auth.Account, error)
	LastLoginForAccount(ctx context.Context, req *auth.GetLastLoginInfoRequest) (*auth.GetLastLoginInfoResponse, error)
	AcceptPayment(ctx context.Context, req *payments.AcceptPaymentRequest) (*payments.AcceptPaymentResponse, error)
	AuthenticateLogin(ctx context.Context, email, password string, duration auth.TokenDuration) (*auth.AuthenticateLoginResponse, error)
	AuthenticateLoginWithCode(ctx context.Context, token, code string, duration auth.TokenDuration) (*auth.AuthenticateLoginWithCodeResponse, error)
	AssertIsEntity(ctx context.Context, entityID string) (*directory.Entity, error)
	BatchJobs(ctx context.Context, req *threading.BatchJobsRequest) (*threading.BatchJobsResponse, error)
	BatchPostMessages(ctx context.Context, req *threading.BatchPostMessagesRequest) (*threading.BatchPostMessagesResponse, error)
	CanPostMessage(ctx context.Context, threadID string) error
	CarePlan(ctx context.Context, id string) (*care.CarePlan, error)
	CheckPasswordResetToken(ctx context.Context, token string) (*auth.CheckPasswordResetTokenResponse, error)
	CheckVerificationCode(ctx context.Context, token, code string) (*auth.CheckVerificationCodeResponse, error)
	ClaimMedia(ctx context.Context, req *media.ClaimMediaRequest) error
	CloneAttachments(ctx context.Context, req *threading.CloneAttachmentsRequest) (*threading.CloneAttachmentsResponse, error)
	CloneMedia(ctx context.Context, req *media.CloneMediaRequest) (*media.CloneMediaResponse, error)
	ConnectVendorAccount(ctx context.Context, req *payments.ConnectVendorAccountRequest) (*payments.ConnectVendorAccountResponse, error)
	ConfigurePatientSync(ctx context.Context, req *patientsync.ConfigureSyncRequest) (*patientsync.ConfigureSyncResponse, error)
	CreateAccount(ctx context.Context, req *auth.CreateAccountRequest) (*auth.CreateAccountResponse, error)
	CreateCarePlan(ctx context.Context, req *care.CreateCarePlanRequest) (*care.CreateCarePlanResponse, error)
	CreateContact(ctx context.Context, req *directory.CreateContactRequest) (*directory.CreateContactResponse, error)
	CreateContacts(ctx context.Context, req *directory.CreateContactsRequest) (*directory.CreateContactsResponse, error)
	CreateEmptyThread(ctx context.Context, req *threading.CreateEmptyThreadRequest) (*threading.Thread, error)
	CreateEntity(ctx context.Context, req *directory.CreateEntityRequest) (*directory.Entity, error)
	CreateEntityDomain(ctx context.Context, organizationID, subdomain string) error
	CreateExternalIDs(ctx context.Context, req *directory.CreateExternalIDsRequest) error
	CreateLinkedThreads(ctx context.Context, req *threading.CreateLinkedThreadsRequest) (*threading.CreateLinkedThreadsResponse, error)
	CreateOnboardingThread(ctx context.Context, req *threading.CreateOnboardingThreadRequest) (*threading.CreateOnboardingThreadResponse, error)
	CreatePasswordResetToken(ctx context.Context, email string) (*auth.CreatePasswordResetTokenResponse, error)
	CreatePayment(ctx context.Context, req *payments.CreatePaymentRequest) (*payments.CreatePaymentResponse, error)
	CreatePaymentMethod(ctx context.Context, req *payments.CreatePaymentMethodRequest) (*payments.CreatePaymentMethodResponse, error)
	CreateScheduledMessage(ctx context.Context, req *threading.CreateScheduledMessageRequest) (*threading.CreateScheduledMessageResponse, error)
	CreateSavedMessage(ctx context.Context, orgID string, req *threading.CreateSavedMessageRequest) (*threading.CreateSavedMessageResponse, error)
	CreateSavedQuery(ctx context.Context, req *threading.CreateSavedQueryRequest) error
	CreateVerificationCode(ctx context.Context, codeType auth.VerificationCodeType, valueToVerify string) (*auth.CreateVerificationCodeResponse, error)
	CreateVisit(ctx context.Context, req *care.CreateVisitRequest) (*care.CreateVisitResponse, error)
	CreateVisitAnswers(ctx context.Context, req *care.CreateVisitAnswersRequest) (*care.CreateVisitAnswersResponse, error)
	DeleteContacts(ctx context.Context, req *directory.DeleteContactsRequest) (*directory.Entity, error)
	DeletePaymentMethod(ctx context.Context, req *payments.DeletePaymentMethodRequest) (*payments.DeletePaymentMethodResponse, error)
	DeleteScheduledMessage(ctx context.Context, req *threading.DeleteScheduledMessageRequest) (*threading.DeleteScheduledMessageResponse, error)
	DeleteSavedMessage(ctx context.Context, req *threading.DeleteSavedMessageRequest) (*threading.DeleteSavedMessageResponse, error)
	DeleteThread(ctx context.Context, threadID, entityID string) error
	DeleteVisit(ctx context.Context, req *care.DeleteVisitRequest) (*care.DeleteVisitResponse, error)
	Entities(ctx context.Context, req *directory.LookupEntitiesRequest, opts ...EntityQueryOption) ([]*directory.Entity, error)
	EntitiesByContact(ctx context.Context, req *directory.LookupEntitiesByContactRequest) ([]*directory.Entity, error)
	EntityDomain(ctx context.Context, entityID, domain string) (*directory.LookupEntityDomainResponse, error)
	GetAnswersForVisit(ctx context.Context, req *care.GetAnswersForVisitRequest) (*care.GetAnswersForVisitResponse, error)
	InitiateIPCall(ctx context.Context, req *excomms.InitiateIPCallRequest) (*excomms.InitiateIPCallResponse, error)
	InitiatePhoneCall(ctx context.Context, req *excomms.InitiatePhoneCallRequest) (*excomms.InitiatePhoneCallResponse, error)
	IPCall(ctx context.Context, id string) (*excomms.IPCall, error)
	MarkThreadsAsRead(ctx context.Context, req *threading.MarkThreadsAsReadRequest) (*threading.MarkThreadsAsReadResponse, error)
	MediaInfo(ctx context.Context, mediaID string) (*media.MediaInfo, error)
	OnboardingThreadEvent(ctx context.Context, req *threading.OnboardingThreadEventRequest) (*threading.OnboardingThreadEventResponse, error)
	Payment(ctx context.Context, req *payments.PaymentRequest) (*payments.PaymentResponse, error)
	PaymentMethods(ctx context.Context, req *payments.PaymentMethodsRequest) (*payments.PaymentMethodsResponse, error)
	PendingIPCalls(ctx context.Context) (*excomms.PendingIPCallsResponse, error)
	PostMessage(ctx context.Context, req *threading.PostMessageRequest) (*threading.PostMessageResponse, error)
	Profile(ctx context.Context, req *directory.ProfileRequest) (*directory.Profile, error)
	ProvisionEmailAddress(ctx context.Context, req *excomms.ProvisionEmailAddressRequest) (*excomms.ProvisionEmailAddressResponse, error)
	ProvisionPhoneNumber(ctx context.Context, req *excomms.ProvisionPhoneNumberRequest) (*excomms.ProvisionPhoneNumberResponse, error)
	QueryThreads(ctx context.Context, req *threading.QueryThreadsRequest) (*threading.QueryThreadsResponse, error)
	SavedMessages(ctx context.Context, req *threading.SavedMessagesRequest) (*threading.SavedMessagesResponse, error)
	SavedQueries(ctx context.Context, entityID string) ([]*threading.SavedQuery, error)
	SavedQueryTemplates(ctx context.Context, entityID string) ([]*threading.SavedQuery, error)
	SavedQuery(ctx context.Context, savedQueryID string) (*threading.SavedQuery, error)
	ScheduledMessages(ctx context.Context, req *threading.ScheduledMessagesRequest) (*threading.ScheduledMessagesResponse, error)
	SearchAllergyMedications(ctx context.Context, req *care.SearchAllergyMedicationsRequest) (*care.SearchAllergyMedicationsResponse, error)
	SearchMedications(ctx context.Context, req *care.SearchMedicationsRequest) (*care.SearchMedicationsResponse, error)
	SearchSelfReportedMedications(ctx context.Context, req *care.SearchSelfReportedMedicationsRequest) (*care.SearchSelfReportedMedicationsResponse, error)
	SendMessage(ctx context.Context, req *excomms.SendMessageRequest) error
	SerializedEntityContact(ctx context.Context, entityID string, platform directory.Platform) (*directory.SerializedClientEntityContact, error)
	SubmitCarePlan(ctx context.Context, cp *care.CarePlan, parentID string) error
	UpdateCarePlan(ctx context.Context, cp *care.CarePlan, req *care.UpdateCarePlanRequest) (*care.CarePlan, error)
	SubmitVisit(ctx context.Context, req *care.SubmitVisitRequest) (*care.SubmitVisitResponse, error)
	Tags(ctx context.Context, req *threading.TagsRequest) (*threading.TagsResponse, error)
	Thread(ctx context.Context, threadID, viewerEntityID string) (*threading.Thread, error)
	Threads(ctx context.Context, req *threading.ThreadsRequest) (*threading.ThreadsResponse, error)
	ThreadItem(ctx context.Context, threadItemID string) (*threading.ThreadItem, error)
	ThreadItems(ctx context.Context, req *threading.ThreadItemsRequest) (*threading.ThreadItemsResponse, error)
	ThreadItemViewDetails(ctx context.Context, threadItemID string) ([]*threading.ThreadItemViewDetails, error)
	ThreadFollowers(ctx context.Context, orgID string, req *threading.ThreadMembersRequest) ([]*directory.Entity, error)
	ThreadMembers(ctx context.Context, orgID string, req *threading.ThreadMembersRequest) ([]*directory.Entity, error)
	ThreadsForMember(ctx context.Context, entityID string, primaryOnly bool) ([]*threading.Thread, error)
	TriageVisit(ctx context.Context, req *care.TriageVisitRequest) (*care.TriageVisitResponse, error)
	Unauthenticate(ctx context.Context, token string) error
	UnauthorizedCreateExternalIDs(ctx context.Context, req *directory.CreateExternalIDsRequest) error
	UpdateAuthToken(ctx context.Context, req *auth.UpdateAuthTokenRequest) (*auth.AuthToken, error)
	UpdateContacts(ctx context.Context, req *directory.UpdateContactsRequest) (*directory.Entity, error)
	UpdateEntity(ctx context.Context, req *directory.UpdateEntityRequest) (*directory.Entity, error)
	UpdateIPCall(ctx context.Context, req *excomms.UpdateIPCallRequest) (*excomms.UpdateIPCallResponse, error)
	UpdateMedia(ctx context.Context, req *media.UpdateMediaRequest) (*media.MediaInfo, error)
	UpdatePassword(ctx context.Context, token, code, newPassword string) error
	UpdateProfile(ctx context.Context, req *directory.UpdateProfileRequest) (*directory.UpdateProfileResponse, error)
	UpdateSavedMessage(ctx context.Context, req *threading.UpdateSavedMessageRequest) (*threading.UpdateSavedMessageResponse, error)
	UpdateThread(ctx context.Context, req *threading.UpdateThreadRequest) (*threading.UpdateThreadResponse, error)
	VendorAccounts(ctx context.Context, req *payments.VendorAccountsRequest) (*payments.VendorAccountsResponse, error)
	UpdateSavedQuery(ctx context.Context, req *threading.UpdateSavedQueryRequest) (*threading.UpdateSavedQueryResponse, error)
	VerifiedValue(ctx context.Context, token string) (string, error)
	Visit(ctx context.Context, req *care.GetVisitRequest) (*care.GetVisitResponse, error)
	Visits(ctx context.Context, req *care.GetVisitsRequest) (*care.GetVisitsResponse, error)
	VisitLayout(ctx context.Context, req *layout.GetVisitLayoutRequest) (*layout.GetVisitLayoutResponse, error)
	VisitLayoutByVersion(ctx context.Context, req *layout.GetVisitLayoutByVersionRequest) (*layout.GetVisitLayoutByVersionResponse, error)
	VisitLayoutVersion(ctx context.Context, req *layout.GetVisitLayoutVersionRequest) (*layout.GetVisitLayoutVersionResponse, error)
	LookupExternalLinksForEntity(ctx context.Context, req *directory.LookupExternalLinksForEntityRequest) (*directory.LookupExternalLinksforEntityResponse, error)
	LookupPatientSyncConfiguration(ctx context.Context, req *patientsync.LookupSyncConfigurationRequest) (*patientsync.LookupSyncConfigurationResponse, error)
}

type resourceAccessor struct {
	rMap        *resourceMap
	auth        auth.AuthClient
	directory   directory.DirectoryClient
	threading   threading.ThreadsClient
	excomms     excomms.ExCommsClient
	layout      layout.LayoutClient
	care        care.CareClient
	media       media.MediaClient
	payments    payments.PaymentsClient
	patientsync patientsync.PatientSyncClient
}

// New returns an initialized instance of resourceAccessor
func New(
	auth auth.AuthClient,
	directory directory.DirectoryClient,
	threading threading.ThreadsClient,
	excomms excomms.ExCommsClient,
	layout layout.LayoutClient,
	care care.CareClient,
	media media.MediaClient,
	payments payments.PaymentsClient,
	patientsync patientsync.PatientSyncClient,
) ResourceAccessor {
	return &resourceAccessor{
		rMap:        newResourceMap(),
		auth:        auth,
		directory:   directory,
		threading:   threading,
		excomms:     excomms,
		layout:      layout,
		care:        care,
		media:       media,
		payments:    payments,
		patientsync: patientsync,
	}
}

// Note: Accounts are the only thing that access is based on something outside of org ownership
func (m *resourceAccessor) canAccessAccount(ctx context.Context, accountID string) error {
	acc := gqlctx.Account(ctx)
	if acc == nil {
		return errors.ErrNotAuthenticated(ctx)
	}
	if acc.ID != accountID {
		return errors.ErrNotAuthorized(ctx, accountID)
	}
	return nil
}

func (m *resourceAccessor) Account(ctx context.Context, accountID string) (*auth.Account, error) {
	if err := m.canAccessAccount(ctx, accountID); err != nil {
		return nil, err
	}
	resp, err := m.auth.GetAccount(ctx, &auth.GetAccountRequest{
		Key: &auth.GetAccountRequest_ID{
			ID: accountID,
		},
	})
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return nil, errors.ErrNotFound(ctx, accountID)
		}
		return nil, err
	}
	return resp.Account, nil
}

func (m *resourceAccessor) LastLoginForAccount(ctx context.Context, req *auth.GetLastLoginInfoRequest) (*auth.GetLastLoginInfoResponse, error) {
	// TODO: Figure out authorization
	return m.auth.GetLastLoginInfo(ctx, req)
}

func (m *resourceAccessor) AuthenticateLogin(ctx context.Context, email, password string, duration auth.TokenDuration) (*auth.AuthenticateLoginResponse, error) {
	headers := devicectx.SpruceHeaders(ctx)
	// Note: There is no authorization required for this operation.
	resp, err := m.auth.AuthenticateLogin(ctx, &auth.AuthenticateLoginRequest{
		Email:    email,
		Password: password,
		DeviceID: headers.DeviceID,
		Platform: determinePlatformForAuth(headers),
		Duration: duration,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) AuthenticateLoginWithCode(ctx context.Context, token, code string, duration auth.TokenDuration) (*auth.AuthenticateLoginWithCodeResponse, error) {
	headers := devicectx.SpruceHeaders(ctx)

	// Note: There is no authorization required for this operation.
	resp, err := m.auth.AuthenticateLoginWithCode(ctx, &auth.AuthenticateLoginWithCodeRequest{
		Token:    token,
		Code:     code,
		DeviceID: headers.DeviceID,
		Platform: determinePlatformForAuth(headers),
		Duration: duration,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func determinePlatformForAuth(headers *device.SpruceHeaders) auth.Platform {
	if headers == nil {
		return auth.Platform_UNKNOWN_PLATFORM
	}

	switch headers.Platform {
	case device.IOS:
		return auth.Platform_IOS
	case device.Android:
		return auth.Platform_ANDROID
	case device.Web:
		return auth.Platform_WEB
	}
	return auth.Platform_WEB
}

func (m *resourceAccessor) CarePlan(ctx context.Context, id string) (*care.CarePlan, error) {
	res, err := m.care.CarePlan(ctx, &care.CarePlanRequest{ID: id})
	if grpc.Code(err) == codes.NotFound {
		return nil, errors.ErrNotFound(ctx, fmt.Sprintf("care plan %s not found", id))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	if err := m.canAccessCarePlan(ctx, res.CarePlan); err != nil {
		return nil, err
	}
	return res.CarePlan, nil
}

// canAccessCarePlan performs an auth check for a care plan. If it has no parent (not yet submitted),
// then only permit the owner (match account ID) access to it. If it has been submitted, then fetch
// the object the parent ID references (e.g. thread item) and validate the account has access to the thread.
func (m *resourceAccessor) canAccessCarePlan(ctx context.Context, cp *care.CarePlan) error {
	if cp.ParentID == "" {
		acc := gqlctx.Account(ctx)
		if acc == nil || acc.ID != cp.CreatorID {
			return errors.ErrNotAuthorized(ctx, cp.ID)
		}
		return nil
	}

	if strings.HasPrefix(cp.ParentID, threading.ThreadItemIDPrefix) {
		return m.canAccessResource(ctx, cp.ParentID, m.orgsForThreadItem)
	} else if strings.HasPrefix(cp.ParentID, threading.SavedMessageIDPrefix) {
		savedMessageRes, err := m.threading.SavedMessages(ctx, &threading.SavedMessagesRequest{
			By: &threading.SavedMessagesRequest_IDs{
				IDs: &threading.IDList{
					IDs: []string{cp.ParentID},
				},
			},
		})
		if err != nil {
			return err
		}
		return m.canAccessResource(ctx, savedMessageRes.SavedMessages[0].OrganizationID, m.orgsForOrganization)
	} else if strings.HasPrefix(cp.ParentID, threading.ScheduledMessageIDPrefix) {
		scheduledMessageRes, err := m.threading.ScheduledMessages(ctx, &threading.ScheduledMessagesRequest{
			LookupKey: &threading.ScheduledMessagesRequest_ScheduledMessageID{
				ScheduledMessageID: cp.ParentID,
			},
		})
		if err != nil {
			return err
		}
		return m.CanPostMessage(ctx, scheduledMessageRes.ScheduledMessages[0].ThreadID)
	}
	return errors.Errorf("Unknown parentID type '%s' to perform authorization check for care plan '%s'", cp.ParentID, cp.ID)
}

func (m *resourceAccessor) CheckPasswordResetToken(ctx context.Context, token string) (*auth.CheckPasswordResetTokenResponse, error) {
	// Note: There is no authorization required for this operation.
	resp, err := m.auth.CheckPasswordResetToken(ctx, &auth.CheckPasswordResetTokenRequest{
		Token: token,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) CheckVerificationCode(ctx context.Context, token, code string) (*auth.CheckVerificationCodeResponse, error) {
	// Note: There is no authorization required for this operation.
	resp, err := m.auth.CheckVerificationCode(ctx, &auth.CheckVerificationCodeRequest{
		Token: token,
		Code:  code,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) CreateAccount(ctx context.Context, req *auth.CreateAccountRequest) (*auth.CreateAccountResponse, error) {
	headers := devicectx.SpruceHeaders(ctx)
	req.DeviceID = headers.DeviceID
	req.Platform = determinePlatformForAuth(headers)

	resp, err := m.auth.CreateAccount(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) CreateCarePlan(ctx context.Context, req *care.CreateCarePlanRequest) (*care.CreateCarePlanResponse, error) {
	// NOTE: There is no authorization required for this operation.
	return m.care.CreateCarePlan(ctx, req)
}

func (m *resourceAccessor) CreateContact(ctx context.Context, req *directory.CreateContactRequest) (*directory.CreateContactResponse, error) {
	if err := m.canAccessResource(ctx, req.EntityID, m.orgsForEntity); err != nil {
		return nil, err
	}
	resp, err := m.createContact(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) CreateContacts(ctx context.Context, req *directory.CreateContactsRequest) (*directory.CreateContactsResponse, error) {
	if err := m.canAccessResource(ctx, req.EntityID, m.orgsForEntity); err != nil {
		return nil, err
	}
	resp, err := m.createContacts(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) CreateEmptyThread(ctx context.Context, req *threading.CreateEmptyThreadRequest) (*threading.Thread, error) {
	if err := m.canAccessResource(ctx, req.OrganizationID, m.orgsForOrganization); err != nil {
		return nil, err
	}
	resp, err := m.createEmptyThread(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Thread, nil
}

func (m *resourceAccessor) CreateEntity(ctx context.Context, req *directory.CreateEntityRequest) (*directory.Entity, error) {
	// TODO: This authorization is interesting since we can't assert the caller belongs to the intended org, but we should be able
	// to assert some global "onBehalfOf" identity that we can asser authorization for. Be it a system entity or person doing the adding.
	resp, err := m.createEntity(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Entity, nil
}

func (m *resourceAccessor) CreateEntityDomain(ctx context.Context, organizationID, subdomain string) error {
	if err := m.canAccessResource(ctx, organizationID, m.orgsForOrganization); err != nil {
		return err
	}
	if err := m.createEntityDomain(ctx, organizationID, subdomain); err != nil {
		return err
	}
	return nil
}

func (m *resourceAccessor) CreateExternalIDs(ctx context.Context, req *directory.CreateExternalIDsRequest) error {
	if err := m.canAccessResource(ctx, req.EntityID, m.orgsForEntity); err != nil {
		return err
	}
	return m.createExternalIDs(ctx, req)
}

func (m *resourceAccessor) CreateLinkedThreads(ctx context.Context, req *threading.CreateLinkedThreadsRequest) (*threading.CreateLinkedThreadsResponse, error) {
	// Note: can't do any real validation for this since it's internal
	return m.threading.CreateLinkedThreads(ctx, req)
}

func (m *resourceAccessor) CreateOnboardingThread(ctx context.Context, req *threading.CreateOnboardingThreadRequest) (*threading.CreateOnboardingThreadResponse, error) {
	// Note: can't do any real validation for this since it's internal
	return m.threading.CreateOnboardingThread(ctx, req)
}

func (m *resourceAccessor) CreatePasswordResetToken(ctx context.Context, email string) (*auth.CreatePasswordResetTokenResponse, error) {
	// Note: There is no authorization required for this operation.
	resp, err := m.createPasswordResetToken(ctx, email)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) CreateSavedQuery(ctx context.Context, req *threading.CreateSavedQueryRequest) error {
	// Note: There is no authorization required for this operation.
	if _, err := m.createSavedQuery(ctx, req); err != nil {
		return err
	}
	return nil
}

func (m *resourceAccessor) CreateVerificationCode(ctx context.Context, codeType auth.VerificationCodeType, valueToVerify string) (*auth.CreateVerificationCodeResponse, error) {
	// Note: There is no authorization required for this operation.
	resp, err := m.createVerificationCode(ctx, codeType, valueToVerify)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) DeleteContacts(ctx context.Context, req *directory.DeleteContactsRequest) (*directory.Entity, error) {
	if err := m.canAccessResource(ctx, req.EntityID, m.orgsForEntity); err != nil {
		return nil, err
	}
	resp, err := m.deleteContacts(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Entity, nil
}

func (m *resourceAccessor) DeleteThread(ctx context.Context, threadID, entityID string) error {
	// TODO: This auth pattern isn't quite right. This asserts that the caller is in the same org as the thread and the entity
	// It does not assert that the caller is the entity
	if err := m.canAccessResource(ctx, threadID, m.orgsForThread); err != nil {
		return err
	}
	if err := m.canAccessResource(ctx, entityID, m.orgsForEntity); err != nil {
		return err
	}
	if err := m.deleteThread(ctx, threadID, entityID); err != nil {
		return err
	}
	return nil
}

func (m *resourceAccessor) EntityDomain(ctx context.Context, entityID, domain string) (*directory.LookupEntityDomainResponse, error) {
	// Only do an authorization check if they are specifying an entity id
	if entityID != "" {
		if err := m.canAccessResource(ctx, entityID, m.orgsForEntity); err != nil {
			return nil, err
		}
	}
	res, err := m.entityDomain(ctx, entityID, domain)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) EntitiesByContact(ctx context.Context, req *directory.LookupEntitiesByContactRequest) ([]*directory.Entity, error) {
	// Note: There is no authorization required for this operation.
	res, err := m.directory.LookupEntitiesByContact(ctx, req)
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	return res.Entities, nil
}

func (m *resourceAccessor) Entities(ctx context.Context, req *directory.LookupEntitiesRequest, opts ...EntityQueryOption) ([]*directory.Entity, error) {

	// auth check
	if !entityQueryOptions(opts).has(EntityQueryOptionUnathorized) {
		acc := gqlctx.Account(ctx)
		if acc == nil {
			return nil, errors.ErrNotAuthenticated(ctx)
		}

		switch key := req.Key.(type) {
		case *directory.LookupEntitiesRequest_EntityID:
			if err := m.canAccessResource(ctx, key.EntityID, m.orgsForEntity); err != nil {
				return nil, err
			}
		case *directory.LookupEntitiesRequest_ExternalID:
			if key.ExternalID != acc.ID {
				return nil, errors.ErrNotAuthenticated(ctx)
			}
		case *directory.LookupEntitiesRequest_AccountID:
			if key.AccountID != acc.ID {
				return nil, errors.ErrNotAuthenticated(ctx)
			}
		case *directory.LookupEntitiesRequest_BatchEntityID:
			// ensure that individual requesting can access each of the entities in the request
			for _, entityID := range key.BatchEntityID.IDs {
				if err := m.canAccessResource(ctx, entityID, m.orgsForEntity); err != nil {
					return nil, err
				}
			}
			// TODO: verify access to entities based on their org memberships. that's expensive so avoiding for now
		}
	}

	res, err := m.directory.LookupEntities(ctx, req)
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	return res.Entities, nil
}

func (m *resourceAccessor) InitiatePhoneCall(ctx context.Context, req *excomms.InitiatePhoneCallRequest) (*excomms.InitiatePhoneCallResponse, error) {
	// TODO: This auth pattern isn't quite right. This asserts that the caller is in the same org as the org and the entity
	// It does not assert that the caller is the entity
	if err := m.canAccessResource(ctx, req.CallerEntityID, m.orgsForEntity); err != nil {
		return nil, err
	}
	if err := m.canAccessResource(ctx, req.OrganizationID, m.orgsForOrganization); err != nil {
		return nil, err
	}
	resp, err := m.initiatePhoneCall(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) MarkThreadsAsRead(ctx context.Context, req *threading.MarkThreadsAsReadRequest) (*threading.MarkThreadsAsReadResponse, error) {
	if len(req.ThreadWatermarks) == 0 {
		return &threading.MarkThreadsAsReadResponse{}, nil
	}

	acc := gqlctx.Account(ctx)
	if acc == nil {
		return nil, errors.ErrNotAuthenticated(ctx)
	}

	entities, err := m.Entities(ctx, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_AccountID{
			AccountID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	})
	if err != nil {
		return nil, err
	}

	// ensure that one of the entities that maps to the account is indeed the entityID
	entityFound := false
	for _, entity := range entities {
		if entity.ID == req.EntityID {
			entityFound = true
			break
		}
	}

	if !entityFound {
		return nil, errors.ErrNotAuthorized(ctx, req.EntityID)
	}

	// ensure that each thread in the list belongs to the same org as that of the caller
	threadIDs := make([]string, len(req.ThreadWatermarks))
	for i, watermark := range req.ThreadWatermarks {
		threadIDs[i] = watermark.ThreadID
	}

	threadsRes, err := m.threading.Threads(ctx, &threading.ThreadsRequest{
		ThreadIDs: threadIDs,
	})
	if err != nil {
		return nil, err
	}
	threads := threadsRes.Threads

	// ensure that all threads belong to one of the orgs the caller is in
	for _, thread := range threads {
		orgFound := false
		for _, entity := range entities {
			for _, membership := range entity.Memberships {
				if membership.ID == thread.OrganizationID {
					orgFound = true
					break
				}
			}
		}
		if !orgFound {
			return nil, errors.ErrNotAuthorized(ctx, thread.ID)
		}
	}

	return m.threading.MarkThreadsAsRead(ctx, req)
}

func (m *resourceAccessor) OnboardingThreadEvent(ctx context.Context, req *threading.OnboardingThreadEventRequest) (*threading.OnboardingThreadEventResponse, error) {
	if err := m.canAccessResource(ctx, req.GetEntityID(), m.orgsForOrganization); err != nil {
		return nil, err
	}
	return m.threading.OnboardingThreadEvent(ctx, req)
}

func (m *resourceAccessor) CanPostMessage(ctx context.Context, threadID string) error {
	return m.canAccessResource(ctx, threadID, m.orgsForThread)
}

func (m *resourceAccessor) PostMessage(ctx context.Context, req *threading.PostMessageRequest) (*threading.PostMessageResponse, error) {
	if err := m.CanPostMessage(ctx, req.ThreadID); err != nil {
		return nil, err
	}

	res, err := m.postMessage(ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) Profile(ctx context.Context, req *directory.ProfileRequest) (*directory.Profile, error) {
	res, err := m.profile(ctx, req)
	if grpc.Code(err) == codes.NotFound {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	// Leverage the Entity call for cache management
	owningEnt, err := Entity(ctx, m, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: res.Profile.EntityID,
		},
	})
	if grpc.Code(err) == codes.NotFound {
		return nil, fmt.Errorf("Unable to locate entity %s mapped to profile", res.Profile.EntityID)
	} else if err != nil {
		return nil, err
	}

	if owningEnt.Type == directory.EntityType_ORGANIZATION {
		if err := m.canAccessResource(ctx, owningEnt.ID, m.orgsForOrganization); err != nil {
			return nil, err
		}
	} else {
		if err := m.canAccessResource(ctx, owningEnt.ID, m.orgsForEntity); err != nil {
			return nil, err
		}
	}
	return res.Profile, nil
}

func (m *resourceAccessor) ProvisionEmailAddress(ctx context.Context, req *excomms.ProvisionEmailAddressRequest) (*excomms.ProvisionEmailAddressResponse, error) {
	if err := m.canAccessResource(ctx, req.ProvisionFor, m.orgsForEntity); err != nil {
		return nil, err
	}
	resp, err := m.provisionEmailAddress(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) ProvisionPhoneNumber(ctx context.Context, req *excomms.ProvisionPhoneNumberRequest) (*excomms.ProvisionPhoneNumberResponse, error) {
	if err := m.canAccessResource(ctx, req.ProvisionFor, m.orgsForEntity); err != nil {
		return nil, err
	}
	resp, err := m.provisionPhoneNumber(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) SavedQueries(ctx context.Context, entityID string) ([]*threading.SavedQuery, error) {
	if err := m.canAccessResource(ctx, entityID, m.orgsForEntity); err != nil {
		return nil, err
	}
	res, err := m.savedQueries(ctx, entityID)
	if err != nil {
		return nil, err
	}
	return res.SavedQueries, nil
}

func (m *resourceAccessor) SavedQueryTemplates(ctx context.Context, entityID string) ([]*threading.SavedQuery, error) {
	if err := m.canAccessResource(ctx, entityID, m.orgsForEntity); err != nil {
		return nil, err
	}
	res, err := m.savedQueryTemplates(ctx, entityID)
	if err != nil {
		return nil, err
	}
	return res.SavedQueries, nil
}

func (m *resourceAccessor) SavedQuery(ctx context.Context, savedQueryID string) (*threading.SavedQuery, error) {
	res, err := m.savedQuery(ctx, savedQueryID)
	if err != nil {
		return nil, err
	}
	if _, err := m.AssertIsEntity(ctx, res.SavedQuery.EntityID); err != nil {
		return nil, err
	}
	return res.SavedQuery, nil
}

func (m *resourceAccessor) SearchAllergyMedications(ctx context.Context, req *care.SearchAllergyMedicationsRequest) (*care.SearchAllergyMedicationsResponse, error) {
	return m.care.SearchAllergyMedications(ctx, req)
}

func (m *resourceAccessor) SearchSelfReportedMedications(ctx context.Context, req *care.SearchSelfReportedMedicationsRequest) (*care.SearchSelfReportedMedicationsResponse, error) {
	return m.care.SearchSelfReportedMedications(ctx, req)
}

func (m *resourceAccessor) SearchMedications(ctx context.Context, req *care.SearchMedicationsRequest) (*care.SearchMedicationsResponse, error) {
	// No auth required for this.
	return m.care.SearchMedications(ctx, req)
}

func (m *resourceAccessor) SendMessage(ctx context.Context, req *excomms.SendMessageRequest) error {
	// Note: There is currently no authorization required for this operation.
	// TODO: Should there be?
	_, err := m.excomms.SendMessage(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func (m *resourceAccessor) SubmitCarePlan(ctx context.Context, cp *care.CarePlan, parentID string) error {
	if err := m.canAccessCarePlan(ctx, cp); err != nil {
		return err
	}
	_, err := m.care.SubmitCarePlan(ctx, &care.SubmitCarePlanRequest{ID: cp.ID, ParentID: parentID})
	return err
}

func (m *resourceAccessor) UpdateCarePlan(ctx context.Context, cp *care.CarePlan, req *care.UpdateCarePlanRequest) (*care.CarePlan, error) {

	if cp == nil {
		cpRes, err := m.care.CarePlan(ctx, &care.CarePlanRequest{
			ID: req.ID,
		})
		if err != nil {
			return nil, err
		}
		cp = cpRes.CarePlan
	}

	if err := m.canAccessCarePlan(ctx, cp); err != nil {
		return nil, err
	}
	res, err := m.care.UpdateCarePlan(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.CarePlan, err
}

func (m *resourceAccessor) SerializedEntityContact(ctx context.Context, entityID string, platform directory.Platform) (*directory.SerializedClientEntityContact, error) {
	if err := m.canAccessResource(ctx, entityID, m.orgsForEntity); err != nil {
		return nil, err
	}
	res, err := m.serializedEntityContact(ctx, entityID, platform)
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return nil, errors.ErrNotFound(ctx, fmt.Sprintf("serialized contact info for entity %s on platform %s", entityID, platform.String()))
		}
		return nil, err
	}
	return res.SerializedEntityContact, nil
}

func (m *resourceAccessor) Thread(ctx context.Context, threadID, viewerEntityID string) (*threading.Thread, error) {
	if err := m.canAccessResource(ctx, threadID, m.orgsForThread); err != nil {
		return nil, err
	}
	res, err := m.thread(ctx, threadID, viewerEntityID)
	if err != nil {
		return nil, err
	}
	return res.Thread, nil
}

func (m *resourceAccessor) Threads(ctx context.Context, req *threading.ThreadsRequest) (*threading.ThreadsResponse, error) {

	// ensure that one of the entities that the account maps to is the viewer entity id
	acc := gqlctx.Account(ctx)
	if acc == nil {
		return nil, errors.ErrNotAuthenticated(ctx)
	}

	entities, err := m.Entities(ctx, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_AccountID{
			AccountID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	})
	if err != nil {
		return nil, err
	}

	if req.ViewerEntityID != "" {
		var entityFound bool
		for _, entity := range entities {
			if entity.ID == req.ViewerEntityID {
				entityFound = true
				break
			}
		}

		if !entityFound {
			return nil, errors.ErrNotAuthorized(ctx, req.ViewerEntityID)
		}
	}

	res, err := m.threading.Threads(ctx, req)
	if err != nil {
		return nil, err
	}

	// ensure that each of the threads queried for belongs to an organization that one of the entities that the account maps to
	// belongs to as well
	for _, thread := range res.Threads {
		var orgFound bool
		for _, entity := range entities {
			for _, membership := range entity.Memberships {
				if membership.ID == thread.OrganizationID {
					orgFound = true
					break
				}
			}
		}

		if !orgFound {
			return nil, errors.ErrNotAuthorized(ctx, thread.ID)
		}
	}

	return res, nil
}

func (m *resourceAccessor) ThreadItem(ctx context.Context, threadItemID string) (*threading.ThreadItem, error) {
	if err := m.canAccessResource(ctx, threadItemID, m.orgsForThreadItem); err != nil {
		return nil, err
	}
	res, err := m.threadItem(ctx, threadItemID)
	if err != nil {
		return nil, err
	}
	return res.Item, nil
}

func (m *resourceAccessor) ThreadItems(ctx context.Context, req *threading.ThreadItemsRequest) (*threading.ThreadItemsResponse, error) {
	if err := m.canAccessResource(ctx, req.ThreadID, m.orgsForThread); err != nil {
		return nil, err
	}
	res, err := m.threadItems(ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) ThreadItemViewDetails(ctx context.Context, threadItemID string) ([]*threading.ThreadItemViewDetails, error) {
	if err := m.canAccessResource(ctx, threadItemID, m.orgsForThreadItem); err != nil {
		return nil, err
	}
	res, err := m.threadItemViewDetails(ctx, threadItemID)
	if err != nil {
		return nil, err
	}
	return res.ItemViewDetails, nil
}

func (m *resourceAccessor) ThreadFollowers(ctx context.Context, orgID string, req *threading.ThreadMembersRequest) ([]*directory.Entity, error) {
	// Being a member of the thread provides access so no need to check out criteria
	res, err := m.threading.ThreadMembers(ctx, req)
	if err != nil {
		return nil, err
	}
	// Make sure viewer is a member of the thread
	acc := gqlctx.Account(ctx)
	if acc == nil {
		return nil, errors.ErrNotAuthorized(ctx, req.ThreadID)
	}
	ent, err := EntityInOrgForAccountID(ctx, m, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		MemberOfEntity: orgID,
		Statuses:       []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:      []directory.EntityType{directory.EntityType_INTERNAL},
	}, orgID)
	if err == ErrNotFound {
		return nil, errors.ErrNotAuthorized(ctx, req.ThreadID)
	} else if err != nil {
		return nil, err
	}

	var found bool
	for _, mem := range res.Members {
		if mem.EntityID == orgID {
			found = true
			break
		}
		if mem.EntityID == ent.ID {
			found = true
			break
		}
	}
	if !found {
		return nil, errors.ErrNotAuthorized(ctx, req.ThreadID)
	}

	if len(res.FollowerEntityIDs) == 0 {
		return []*directory.Entity{}, nil
	}

	leres, err := m.directory.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_BatchEntityID{
			BatchEntityID: &directory.IDList{
				IDs: res.FollowerEntityIDs,
			},
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	})
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	return leres.Entities, nil
}

func (m *resourceAccessor) ThreadMembers(ctx context.Context, orgID string, req *threading.ThreadMembersRequest) ([]*directory.Entity, error) {
	// Being a member of the thread provides access so no need to check out criteria
	res, err := m.threading.ThreadMembers(ctx, req)
	if err != nil {
		return nil, err
	}
	// Make sure viewer is a member of the thread
	acc := gqlctx.Account(ctx)
	if acc == nil {
		return nil, errors.ErrNotAuthorized(ctx, req.ThreadID)
	}
	ent, err := EntityInOrgForAccountID(ctx, m, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
	}, orgID)
	if err != nil {
		return nil, err
	}

	var found bool
	for _, mem := range res.Members {
		if mem.EntityID == ent.ID {
			found = true
			break
		}
	}
	if !found {
		return nil, errors.ErrNotAuthorized(ctx, req.ThreadID)
	}

	if len(res.Members) == 0 {
		return []*directory.Entity{}, nil
	}

	entIDs := make([]string, len(res.Members))
	for i, m := range res.Members {
		entIDs[i] = m.EntityID
	}

	leres, err := m.directory.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_BatchEntityID{
			BatchEntityID: &directory.IDList{
				IDs: entIDs,
			},
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	})
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	return leres.Entities, nil
}

func (m *resourceAccessor) ThreadsForMember(ctx context.Context, entityID string, primaryOnly bool) ([]*threading.Thread, error) {
	if err := m.canAccessResource(ctx, entityID, m.orgsForEntity); err != nil {
		return nil, err
	}
	res, err := m.threadsForMember(ctx, entityID, primaryOnly)
	if err != nil {
		return nil, err
	}
	return res.Threads, nil
}

func (m *resourceAccessor) QueryThreads(ctx context.Context, req *threading.QueryThreadsRequest) (*threading.QueryThreadsResponse, error) {
	// Note: There is no authorization required as the threading service will only return threads for the viewing entity ID.
	res, err := m.queryThreads(ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) VerifiedValue(ctx context.Context, token string) (string, error) {
	// Note: There is no authorization required for this operation.
	res, err := m.verifiedValue(ctx, token)
	if err != nil {
		return "", err
	}
	return res.Value, nil
}

func (m *resourceAccessor) Unauthenticate(ctx context.Context, token string) error {
	// Note: There is no authorization required for this operation.
	if _, err := m.unauthenticate(ctx, token); err != nil {
		return err
	}
	return nil
}

func (m *resourceAccessor) UnauthorizedCreateExternalIDs(ctx context.Context, req *directory.CreateExternalIDsRequest) error {
	return m.createExternalIDs(ctx, req)
}

func (m *resourceAccessor) UpdateAuthToken(ctx context.Context, req *auth.UpdateAuthTokenRequest) (*auth.AuthToken, error) {
	res, err := m.auth.UpdateAuthToken(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Token, nil
}

func (m *resourceAccessor) UpdateContacts(ctx context.Context, req *directory.UpdateContactsRequest) (*directory.Entity, error) {
	if err := m.canAccessResource(ctx, req.EntityID, m.orgsForEntity); err != nil {
		return nil, err
	}
	res, err := m.updateContacts(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Entity, nil
}

func (m *resourceAccessor) UpdateEntity(ctx context.Context, req *directory.UpdateEntityRequest) (*directory.Entity, error) {
	if err := m.canAccessResource(ctx, req.EntityID, m.orgsForEntity); err != nil {
		return nil, err
	}
	res, err := m.updateEntity(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Entity, nil
}

func (m *resourceAccessor) UpdatePassword(ctx context.Context, token, code, newPassword string) error {
	// Note: There is no authorization required for this operation. It is done remotely in the auth service
	if _, err := m.updatePassword(ctx, token, code, newPassword); err != nil {
		return err
	}
	return nil
}

// ProfileAllowEdit returns if the caller is allowed to edit a profiled owned by the provided ID
func ProfileAllowEdit(ctx context.Context, ram ResourceAccessor, profileEntityID string) bool {
	acc := gqlctx.Account(ctx)
	if acc == nil {
		golog.Errorf("Encountered error while determining editibility of Profile %s: No account set in context", profileEntityID)
		return false
	}
	// Only providers can edit profiles
	if acc.Type != auth.AccountType_PROVIDER {
		return false
	}
	callerEnts, err := ram.Entities(ctx, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_AccountID{
			AccountID: acc.ID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_INTERNAL},
	})
	// Any error means no edit
	if err != nil {
		golog.Errorf("Encountered error while determining editibility of Profile %s by account %s: %s", profileEntityID, acc.ID, err)
		return false
	}
	// If we own the profile we can edit it
	for _, cEnt := range callerEnts {
		if profileEntityID == cEnt.ID {
			return true
		}
	}

	// If we aren't the owner we need to see if the entity is an org and then check membership
	maybeOrgEnt, err := Entity(ctx, ram, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: profileEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS},
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	})
	// Any error means no edit
	if err != nil {
		golog.Warningf("Encountered error while determining editibility of Profile %s by account %s: %s", profileEntityID, acc.ID, err)
		return false
	}
	// This filter check is redudent with the call filter, but let's be safe
	if maybeOrgEnt.Type == directory.EntityType_ORGANIZATION {
		for _, m := range maybeOrgEnt.Members {
			for _, cEnt := range callerEnts {
				if m.ID == cEnt.ID {
					return true
				}
			}
		}
	}
	return false
}

// UpdateProfile handles create and update requests
func (m *resourceAccessor) UpdateProfile(ctx context.Context, req *directory.UpdateProfileRequest) (*directory.UpdateProfileResponse, error) {
	owningEntityID := req.Profile.EntityID
	// If no entity ID is provided then lookup the profile so we can authorize the edit
	if owningEntityID == "" {
		res, err := m.Profile(ctx, &directory.ProfileRequest{
			Key: &directory.ProfileRequest_ProfileID{
				ProfileID: req.ProfileID,
			},
		})
		if err != nil {
			return nil, err
		}
		owningEntityID = res.EntityID
	}
	if !ProfileAllowEdit(ctx, m, owningEntityID) {
		return nil, errors.ErrNotAuthorized(ctx, fmt.Sprintf("Profile for %s", req.Profile.EntityID))
	}
	res, err := m.directory.UpdateProfile(ctx, req)
	if grpc.Code(err) == codes.NotFound {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) UpdateThread(ctx context.Context, req *threading.UpdateThreadRequest) (*threading.UpdateThreadResponse, error) {
	// For authorization, the threading services validtes the actor entity ID against the members of the thread
	return m.threading.UpdateThread(ctx, req)
}

func (m *resourceAccessor) LookupExternalLinksForEntity(ctx context.Context, req *directory.LookupExternalLinksForEntityRequest) (*directory.LookupExternalLinksforEntityResponse, error) {
	if err := m.canAccessResource(ctx, req.EntityID, m.orgsForEntity); err != nil {
		return nil, err
	}

	res, err := m.directory.LookupExternalLinksForEntity(ctx, req)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return res, nil
}

func (m *resourceAccessor) UpdateSavedQuery(ctx context.Context, req *threading.UpdateSavedQueryRequest) (*threading.UpdateSavedQueryResponse, error) {
	sResp, err := m.SavedQuery(ctx, req.SavedQueryID)
	if err != nil {
		return nil, err
	}

	if _, err := m.AssertIsEntity(ctx, sResp.EntityID); err != nil {
		return nil, err
	}

	res, err := m.threading.UpdateSavedQuery(ctx, req)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return res, nil
}

func (m *resourceAccessor) isAccountType(ctx context.Context, accType auth.AccountType) bool {
	acc := gqlctx.Account(ctx)
	return acc != nil && acc.Type == accType
}

func (m *resourceAccessor) AssertIsEntity(ctx context.Context, entityID string) (*directory.Entity, error) {
	acc := gqlctx.Account(ctx)
	if acc == nil {
		return nil, errors.ErrNotAuthenticated(ctx)
	}
	ent, err := directory.SingleEntity(ctx, m.directory, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entityID,
		},
	})
	if grpc.Code(err) == codes.NotFound {
		return nil, errors.ErrNotFound(ctx, entityID)
	} else if err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	if ent.AccountID != acc.ID {
		return nil, errors.ErrNotAuthorized(ctx, entityID)
	}
	return ent, nil
}

func (m *resourceAccessor) canAccessResource(ctx context.Context, resourceID string, missF func(ctx context.Context, resourceID string) (map[string]struct{}, error)) error {
	var err error

	// Get the information about the caller
	acc := gqlctx.Account(ctx)
	if acc == nil || acc.ID == "" {
		return errors.ErrNotAuthenticated(ctx)
	}

	// Get the orgs for the resource
	resourceOrgs := m.rMap.Get(resourceID)

	// Get the orgs for the account
	accountOrgs := m.rMap.Get(acc.ID)

	// Check for overlap
	for aOrg := range accountOrgs {
		if _, ok := resourceOrgs[aOrg]; ok {
			// If the account belongs to any org that owns the resource
			return nil
		}
	}

	golog.Debugf("Authorization: Miss - Refreshing information for account %s and resource %s", acc.ID, resourceID)

	// If we missed then perhaps we need to update the cache
	// Utilize the miss func to refresh the resource
	resourceOrgs, err = missF(ctx, resourceID)
	if err != nil {
		return err
	}
	m.rMap.Set(resourceID, resourceOrgs)

	// Update the orgs associated with the account
	accountOrgs, err = m.orgsForEntityForExternalID(ctx, acc.ID)
	if err != nil {
		return err
	}
	m.rMap.Set(acc.ID, accountOrgs)

	// check for overlap again
	for aOrg := range accountOrgs {
		if _, ok := resourceOrgs[aOrg]; ok {
			// If the account belongs to any org that owns the resource
			return nil
		}
	}

	// If no overlap is found return an authorization failure
	return errors.ErrNotAuthorized(ctx, resourceID)
}

func (m *resourceAccessor) orgsForEntity(ctx context.Context, entityID string) (map[string]struct{}, error) {
	// Don't do any status checks. Authorization is for all existing resources
	res, err := m.directory.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
			Depth:             0,
		},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	})
	if err != nil {
		return nil, err
	} else if len(res.Entities) != 1 {
		return nil, fmt.Errorf("Expected 1 entity for %s but got %d", entityID, len(res.Entities))
	}

	return orgsForEntity(res.Entities[0]), nil
}

func (m *resourceAccessor) orgsForEntityForExternalID(ctx context.Context, externalID string) (map[string]struct{}, error) {
	// Don't do any status checks. Authorization is for all existing resources

	res, err := m.directory.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: externalID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
			Depth:             0,
		},
	})
	if err != nil {
		return nil, err
	}
	// TODO: In the future it will be possible for this to return more than 1 result as accounts are mapped to multiple entities
	// Need to figure out how to deal with this from a merging/caching perspective
	if len(res.Entities) != 1 {
		return nil, errors.InternalError(ctx, fmt.Errorf("Expected only 1 entity to be returned for external id %s but found %d", externalID, len(res.Entities)))
	}
	return orgsForEntity(res.Entities[0]), nil
}

func (m *resourceAccessor) orgsForThread(ctx context.Context, threadID string) (map[string]struct{}, error) {
	res, err := m.thread(ctx, threadID, "")
	if err != nil {
		return nil, err
	}
	return map[string]struct{}{res.Thread.OrganizationID: {}}, nil
}

func (m *resourceAccessor) orgsForThreadItem(ctx context.Context, threadItemID string) (map[string]struct{}, error) {
	res, err := m.threadItem(ctx, threadItemID)
	if err != nil {
		return nil, err
	}
	return map[string]struct{}{res.Item.OrganizationID: {}}, nil
}

func (m *resourceAccessor) orgsForOrganization(ctx context.Context, organizationID string) (map[string]struct{}, error) {
	// Just map organizatiions as members of themselves
	return map[string]struct{}{organizationID: {}}, nil
}

func orgsForEntity(e *directory.Entity) map[string]struct{} {
	orgs := make(map[string]struct{})
	if e.Type == directory.EntityType_ORGANIZATION {
		// Orgs are member of themselves for current mapping purposes
		orgs[e.ID] = struct{}{}
	} else {
		for _, mem := range e.Memberships {
			if mem.Type == directory.EntityType_ORGANIZATION {
				orgs[mem.ID] = struct{}{}
			}
		}
	}
	return orgs
}

func (m *resourceAccessor) createContact(ctx context.Context, req *directory.CreateContactRequest) (*directory.CreateContactResponse, error) {
	resp, err := m.directory.CreateContact(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) createContacts(ctx context.Context, req *directory.CreateContactsRequest) (*directory.CreateContactsResponse, error) {
	resp, err := m.directory.CreateContacts(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) createEmptyThread(ctx context.Context, req *threading.CreateEmptyThreadRequest) (*threading.CreateEmptyThreadResponse, error) {
	resp, err := m.threading.CreateEmptyThread(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) createEntity(ctx context.Context, req *directory.CreateEntityRequest) (*directory.CreateEntityResponse, error) {
	resp, err := m.directory.CreateEntity(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) createEntityDomain(ctx context.Context, organizationID, subdomain string) error {
	if _, err := m.directory.CreateEntityDomain(ctx, &directory.CreateEntityDomainRequest{
		EntityID: organizationID,
		Domain:   subdomain,
	}); err != nil {
		return err
	}
	return nil
}

func (m *resourceAccessor) createExternalIDs(ctx context.Context, req *directory.CreateExternalIDsRequest) error {
	_, err := m.directory.CreateExternalIDs(ctx, req)
	return err
}

func (m *resourceAccessor) createSavedQuery(ctx context.Context, req *threading.CreateSavedQueryRequest) (*threading.CreateSavedQueryResponse, error) {
	resp, err := m.threading.CreateSavedQuery(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) createPasswordResetToken(ctx context.Context, email string) (*auth.CreatePasswordResetTokenResponse, error) {
	resp, err := m.auth.CreatePasswordResetToken(ctx, &auth.CreatePasswordResetTokenRequest{
		Email: email,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) createVerificationCode(ctx context.Context, codeType auth.VerificationCodeType, valueToVerify string) (*auth.CreateVerificationCodeResponse, error) {
	resp, err := m.auth.CreateVerificationCode(ctx, &auth.CreateVerificationCodeRequest{
		Type:          codeType,
		ValueToVerify: valueToVerify,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) deleteContacts(ctx context.Context, req *directory.DeleteContactsRequest) (*directory.DeleteContactsResponse, error) {
	resp, err := m.directory.DeleteContacts(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) deleteThread(ctx context.Context, threadID, entityID string) error {
	_, err := m.threading.DeleteThread(ctx, &threading.DeleteThreadRequest{
		ThreadID:      threadID,
		ActorEntityID: entityID,
	})
	if err != nil {
		return err
	}
	return nil
}

func (m *resourceAccessor) entityDomain(ctx context.Context, entityID, domain string) (*directory.LookupEntityDomainResponse, error) {
	resp, err := m.directory.LookupEntityDomain(ctx, &directory.LookupEntityDomainRequest{
		EntityID: entityID,
		Domain:   domain,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) initiatePhoneCall(ctx context.Context, req *excomms.InitiatePhoneCallRequest) (*excomms.InitiatePhoneCallResponse, error) {
	res, err := m.excomms.InitiatePhoneCall(ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) postMessage(ctx context.Context, req *threading.PostMessageRequest) (*threading.PostMessageResponse, error) {
	res, err := m.threading.PostMessage(ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) profile(ctx context.Context, req *directory.ProfileRequest) (*directory.ProfileResponse, error) {
	res, err := m.directory.Profile(ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) provisionEmailAddress(ctx context.Context, req *excomms.ProvisionEmailAddressRequest) (*excomms.ProvisionEmailAddressResponse, error) {
	res, err := m.excomms.ProvisionEmailAddress(ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) provisionPhoneNumber(ctx context.Context, req *excomms.ProvisionPhoneNumberRequest) (*excomms.ProvisionPhoneNumberResponse, error) {
	res, err := m.excomms.ProvisionPhoneNumber(ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) queryThreads(ctx context.Context, req *threading.QueryThreadsRequest) (*threading.QueryThreadsResponse, error) {
	res, err := m.threading.QueryThreads(ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) savedQueries(ctx context.Context, entityID string) (*threading.SavedQueriesResponse, error) {
	res, err := m.threading.SavedQueries(ctx, &threading.SavedQueriesRequest{
		EntityID: entityID,
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) savedQueryTemplates(ctx context.Context, entityID string) (*threading.SavedQueryTemplatesResponse, error) {
	res, err := m.threading.SavedQueryTemplates(ctx, &threading.SavedQueryTemplatesRequest{
		EntityID: entityID,
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) savedQuery(ctx context.Context, savedQueryID string) (*threading.SavedQueryResponse, error) {
	res, err := m.threading.SavedQuery(ctx, &threading.SavedQueryRequest{
		SavedQueryID: savedQueryID,
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) serializedEntityContact(ctx context.Context, entityID string, platform directory.Platform) (*directory.SerializedEntityContactResponse, error) {
	res, err := m.directory.SerializedEntityContact(ctx, &directory.SerializedEntityContactRequest{
		EntityID: entityID,
		Platform: platform,
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) thread(ctx context.Context, threadID, viewerEntityID string) (*threading.ThreadResponse, error) {
	res, err := m.threading.Thread(ctx, &threading.ThreadRequest{
		ThreadID:       threadID,
		ViewerEntityID: viewerEntityID,
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) threadItem(ctx context.Context, threadItemID string) (*threading.ThreadItemResponse, error) {
	res, err := m.threading.ThreadItem(ctx, &threading.ThreadItemRequest{
		ItemID: threadItemID,
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) threadItems(ctx context.Context, req *threading.ThreadItemsRequest) (*threading.ThreadItemsResponse, error) {
	res, err := m.threading.ThreadItems(ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) threadItemViewDetails(ctx context.Context, threadItemID string) (*threading.ThreadItemViewDetailsResponse, error) {
	res, err := m.threading.ThreadItemViewDetails(ctx, &threading.ThreadItemViewDetailsRequest{
		ItemID: threadItemID,
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) threadsForMember(ctx context.Context, entityID string, primaryOnly bool) (*threading.ThreadsForMemberResponse, error) {
	res, err := m.threading.ThreadsForMember(ctx, &threading.ThreadsForMemberRequest{
		EntityID:    entityID,
		PrimaryOnly: primaryOnly,
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) unauthenticate(ctx context.Context, token string) (*auth.UnauthenticateResponse, error) {
	res, err := m.auth.Unauthenticate(ctx, &auth.UnauthenticateRequest{
		Token: token,
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) updateContacts(ctx context.Context, req *directory.UpdateContactsRequest) (*directory.UpdateContactsResponse, error) {
	res, err := m.directory.UpdateContacts(ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) updateEntity(ctx context.Context, req *directory.UpdateEntityRequest) (*directory.UpdateEntityResponse, error) {
	res, err := m.directory.UpdateEntity(ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) updatePassword(ctx context.Context, token, code, newPassword string) (*auth.UpdatePasswordResponse, error) {
	res, err := m.auth.UpdatePassword(ctx, &auth.UpdatePasswordRequest{
		Token:       token,
		Code:        code,
		NewPassword: newPassword,
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) verifiedValue(ctx context.Context, token string) (*auth.VerifiedValueResponse, error) {
	res, err := m.auth.VerifiedValue(ctx, &auth.VerifiedValueRequest{
		Token: token,
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// entityInfoWithContacts adds contacts in the requested information if not already included
func entityInfoWithContacts(info []directory.EntityInformation) []directory.EntityInformation {
	for _, ei := range info {
		if ei == directory.EntityInformation_CONTACTS {
			return info
		}
	}
	return append(info, directory.EntityInformation_CONTACTS)
}
