package raccess

import (
	"fmt"
	"sync"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// ParamKey is the name of the parameter index
const ParamKey = "ram"

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

// ResourceAccessor defines an interface for the retreival and authorization of resources
type ResourceAccessor interface {
	Account(ctx context.Context, accountID string) (*auth.Account, error)
	AuthenticateLogin(ctx context.Context, email, password string) (*auth.AuthenticateLoginResponse, error)
	AuthenticateLoginWithCode(ctx context.Context, token, code string) (*auth.AuthenticateLoginWithCodeResponse, error)
	CanPostMessage(ctx context.Context, threadID string) error
	CarePlan(ctx context.Context, id string) (*care.CarePlan, error)
	CheckPasswordResetToken(ctx context.Context, token string) (*auth.CheckPasswordResetTokenResponse, error)
	CheckVerificationCode(ctx context.Context, token, code string) (*auth.CheckVerificationCodeResponse, error)
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
	CreateSavedQuery(ctx context.Context, req *threading.CreateSavedQueryRequest) error
	CreateVerificationCode(ctx context.Context, codeType auth.VerificationCodeType, valueToVerify string) (*auth.CreateVerificationCodeResponse, error)
	DeleteContacts(ctx context.Context, req *directory.DeleteContactsRequest) (*directory.Entity, error)
	DeleteThread(ctx context.Context, threadID, entityID string) error
	Entity(ctx context.Context, entityID string, entityInfo []directory.EntityInformation, depth int64) (*directory.Entity, error)
	Entities(ctx context.Context, orgID string, entityIDs []string, entityInfo []directory.EntityInformation) ([]*directory.Entity, error)
	EntityDomain(ctx context.Context, entityID, domain string) (*directory.LookupEntityDomainResponse, error)
	// TODO: Rename this EntityForOrgAndAccountID
	EntityForAccountID(ctx context.Context, orgID, accountID string) (*directory.Entity, error)
	EntitiesByContact(ctx context.Context, contactValue string, entityInfo []directory.EntityInformation, depth int64, statuses []directory.EntityStatus) ([]*directory.Entity, error)
	EntitiesForExternalID(ctx context.Context, externalID string, entityInfo []directory.EntityInformation, depth int64, statuses []directory.EntityStatus) ([]*directory.Entity, error)
	InitiatePhoneCall(ctx context.Context, req *excomms.InitiatePhoneCallRequest) (*excomms.InitiatePhoneCallResponse, error)
	MarkThreadAsRead(ctx context.Context, threadID, entityID string) error
	OnboardingThreadEvent(ctx context.Context, req *threading.OnboardingThreadEventRequest) (*threading.OnboardingThreadEventResponse, error)
	PatientEntity(ctx context.Context, a *models.PatientAccount) (*directory.Entity, error)
	PostMessage(ctx context.Context, req *threading.PostMessageRequest) (*threading.PostMessageResponse, error)
	ProvisionPhoneNumber(ctx context.Context, req *excomms.ProvisionPhoneNumberRequest) (*excomms.ProvisionPhoneNumberResponse, error)
	ProvisionEmailAddress(ctx context.Context, req *excomms.ProvisionEmailAddressRequest) (*excomms.ProvisionEmailAddressResponse, error)
	QueryThreads(ctx context.Context, req *threading.QueryThreadsRequest) (*threading.QueryThreadsResponse, error)
	SavedQuery(ctx context.Context, savedQueryID string) (*threading.SavedQuery, error)
	SavedQueries(ctx context.Context, entityID string) ([]*threading.SavedQuery, error)
	SendMessage(ctx context.Context, req *excomms.SendMessageRequest) error
	SerializedEntityContact(ctx context.Context, entityID string, platform directory.Platform) (*directory.SerializedClientEntityContact, error)
	SubmitCarePlan(ctx context.Context, cp *care.CarePlan, parentID string) error
	Thread(ctx context.Context, threadID, viewerEntityID string) (*threading.Thread, error)
	ThreadItem(ctx context.Context, threadItemID string) (*threading.ThreadItem, error)
	ThreadItems(ctx context.Context, req *threading.ThreadItemsRequest) (*threading.ThreadItemsResponse, error)
	ThreadItemViewDetails(ctx context.Context, threadItemID string) ([]*threading.ThreadItemViewDetails, error)
	ThreadMembers(ctx context.Context, orgID string, req *threading.ThreadMembersRequest) ([]*directory.Entity, error)
	ThreadsForMember(ctx context.Context, entityID string, primaryOnly bool) ([]*threading.Thread, error)
	Unauthenticate(ctx context.Context, token string) error
	UnauthorizedCreateExternalIDs(ctx context.Context, req *directory.CreateExternalIDsRequest) error
	UnauthorizedEntity(ctx context.Context, entityID string, entityInfo []directory.EntityInformation, depth int64) (*directory.Entity, error)
	UpdateContacts(ctx context.Context, req *directory.UpdateContactsRequest) (*directory.Entity, error)
	UpdateEntity(ctx context.Context, req *directory.UpdateEntityRequest) (*directory.Entity, error)
	UpdatePassword(ctx context.Context, token, code, newPassword string) error
	UpdateThread(ctx context.Context, req *threading.UpdateThreadRequest) (*threading.UpdateThreadResponse, error)
	VerifiedValue(ctx context.Context, token string) (string, error)
	VisitLayout(ctx context.Context, req *layout.GetVisitLayoutRequest) (*layout.GetVisitLayoutResponse, error)
	CreateVisit(ctx context.Context, req *care.CreateVisitRequest) (*care.CreateVisitResponse, error)
	Visit(ctx context.Context, req *care.GetVisitRequest) (*care.GetVisitResponse, error)
	SubmitVisit(ctx context.Context, req *care.SubmitVisitRequest) (*care.SubmitVisitResponse, error)
	VisitLayoutVersion(ctx context.Context, req *layout.GetVisitLayoutVersionRequest) (*layout.GetVisitLayoutVersionResponse, error)
	CreateVisitAnswers(ctx context.Context, req *care.CreateVisitAnswersRequest) (*care.CreateVisitAnswersResponse, error)
	GetAnswersForVisit(ctx context.Context, req *care.GetAnswersForVisitRequest) (*care.GetAnswersForVisitResponse, error)
}

type resourceAccessor struct {
	rMap      *resourceMap
	auth      auth.AuthClient
	directory directory.DirectoryClient
	threading threading.ThreadsClient
	excomms   excomms.ExCommsClient
	layout    layout.LayoutClient
	care      care.CareClient
}

// New returns an initialized instance of resourceAccessor
func New(
	auth auth.AuthClient,
	directory directory.DirectoryClient,
	threading threading.ThreadsClient,
	excomms excomms.ExCommsClient,
	layout layout.LayoutClient,
	care care.CareClient,
) ResourceAccessor {
	return &resourceAccessor{
		rMap:      newResourceMap(),
		auth:      auth,
		directory: directory,
		threading: threading,
		excomms:   excomms,
		layout:    layout,
		care:      care,
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
		AccountID: accountID,
	})
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return nil, errors.ErrNotFound(ctx, accountID)
		}
		return nil, err
	}
	return resp.Account, nil
}

func (m *resourceAccessor) AuthenticateLogin(ctx context.Context, email, password string) (*auth.AuthenticateLoginResponse, error) {
	headers := gqlctx.SpruceHeaders(ctx)
	// Note: There is no authorization required for this operation.
	resp, err := m.auth.AuthenticateLogin(ctx, &auth.AuthenticateLoginRequest{
		Email:    email,
		Password: password,
		DeviceID: headers.DeviceID,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) AuthenticateLoginWithCode(ctx context.Context, token, code string) (*auth.AuthenticateLoginWithCodeResponse, error) {
	headers := gqlctx.SpruceHeaders(ctx)
	// Note: There is no authorization required for this operation.
	resp, err := m.auth.AuthenticateLoginWithCode(ctx, &auth.AuthenticateLoginWithCodeRequest{
		Token:    token,
		Code:     code,
		DeviceID: headers.DeviceID,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *resourceAccessor) CarePlan(ctx context.Context, id string) (*care.CarePlan, error) {
	// Authorization: if the care plan has no parent (not yet submitted), then only permit the owner (match account ID)
	// access to it. If it has been submitted, then fetch the object the parent ID references (e.g. thread item) and validate
	// the account has access to the thread.
	golog.Errorf("TODO: implement authorization for CarePlan")
	res, err := m.care.CarePlan(ctx, &care.CarePlanRequest{ID: id})
	if grpc.Code(err) == codes.NotFound {
		return nil, errors.ErrNotFound(ctx, fmt.Sprintf("care plan %s not found", id))
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return res.CarePlan, nil
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
	headers := gqlctx.SpruceHeaders(ctx)
	req.DeviceID = headers.DeviceID
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
	// TODO: This auth pattern isn't quite right. This asserts that the caller is in the same org as the entity
	// It does not assert that the caller is the entity
	if err := m.canAccessResource(ctx, req.EntityID, m.orgsForEntity); err != nil {
		return err
	}
	if err := m.canAccessResource(ctx, req.OrganizationID, m.orgsForOrganization); err != nil {
		return err
	}
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

func cachedEntities(ctx context.Context, entityIDs []string, wantedInfo []directory.EntityInformation, depth int64) ([]*directory.Entity, []string) {
	// Currently we can only cache entities with a search depth of 0
	if depth != 0 {
		return nil, entityIDs
	}
	notFoundEntIDs := make([]string, 0, len(entityIDs))
	cachedEnts := make([]*directory.Entity, 0, len(entityIDs))
	for _, eID := range entityIDs {
		cE := cachedEntity(ctx, eID, wantedInfo, depth)
		if cE != nil {
			cachedEnts = append(cachedEnts, cE)
		} else {
			notFoundEntIDs = append(notFoundEntIDs, eID)
		}
	}
	return cachedEnts, notFoundEntIDs
}

func cachedEntity(ctx context.Context, entityID string, wantedInfo []directory.EntityInformation, depth int64) *directory.Entity {
	if depth != 0 {
		return nil
	}

	ec := gqlctx.Entities(ctx)
	if ec == nil {
		return nil
	}
	ent := ec.Get(entityID)
	if ent == nil {
		return nil
	}
	// Determine if our cached value has enough information to meet the request
	// TODO: We could hash what we have to make the lookups faster, but that would require an alloc for every cache check. For now just iterate till something else is figured
	for _, wei := range wantedInfo {
		var found bool
		for _, ei := range ent.IncludedInformation {
			if ei == wei {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	return ent
}

func cacheEntities(ctx context.Context, ents []*directory.Entity) {
	ec := gqlctx.Entities(ctx)
	if ec == nil {
		return
	}
	for _, ent := range ents {
		// Note: Perhaps read then write? Depends on if we only call this after a remote call
		ec.Set(ent.ID, ent)
		for _, mem := range append(ent.Members, ent.Memberships...) {
			ec.Set(mem.ID, mem)
		}
	}
}

func (m *resourceAccessor) Entity(ctx context.Context, entityID string, entityInfo []directory.EntityInformation, depth int64) (*directory.Entity, error) {
	ent := cachedEntity(ctx, entityID, entityInfo, depth)
	if ent != nil {
		return ent, nil
	}
	if err := m.canAccessResource(ctx, entityID, m.orgsForEntity); err != nil {
		return nil, err
	}
	res, err := m.entity(ctx, entityID, entityInfo, depth, nil)
	if err != nil {
		return nil, err
	}
	cacheEntities(ctx, res.Entities)
	return res.Entities[0], nil
}

func (m *resourceAccessor) Entities(ctx context.Context, orgID string, entityIDs []string, entityInfo []directory.EntityInformation) ([]*directory.Entity, error) {
	// Check our cache for the entities and filter anything we already have
	// A depth of 0 will return everything but members of members
	var depth int64
	cachedEnts, notFoundEntIDs := cachedEntities(ctx, entityIDs, entityInfo, depth)
	if len(notFoundEntIDs) == 0 {
		return cachedEnts, nil
	}

	if err := m.canAccessResource(ctx, orgID, m.orgsForOrganization); err != nil {
		return nil, err
	}
	// TODO: verify access to entities based on their org memberships. that's expensive so avoiding for now
	res, err := m.directory.LookupEntities(ctx,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
				BatchEntityID: &directory.IDList{IDs: notFoundEntIDs},
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth:             depth,
				EntityInformation: entityInfo,
			},
			Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		})
	if err != nil {
		return nil, err
	}
	cacheEntities(ctx, res.Entities)
	return append(res.Entities, cachedEnts...), nil
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

func (m *resourceAccessor) EntityForAccountID(ctx context.Context, orgID, accountID string) (*directory.Entity, error) {
	// Check our cached account entities first
	acc := gqlctx.Account(ctx)
	if acc != nil && acc.ID == accountID {
		ents := gqlctx.AccountEntities(ctx)
		if ents != nil {
			ent := ents.Get(orgID)
			if ent != nil {
				return ent, nil
			}
		}
	}
	// Note: Authorization is done at the next level down
	entities, err := m.EntitiesForExternalID(ctx, accountID, []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS}, 0, nil)
	if err != nil {
		return nil, err
	}
	for _, e := range entities {
		for _, e2 := range e.GetMemberships() {
			if e2.Type == directory.EntityType_ORGANIZATION && e2.ID == orgID {
				return e, nil
			}
		}
	}
	return nil, errors.ErrNotFound(ctx, fmt.Sprintf("(entity for account %s and org %s)", accountID, orgID))
}

func (m *resourceAccessor) EntitiesByContact(ctx context.Context, contactValue string, entityInfo []directory.EntityInformation, depth int64, statuses []directory.EntityStatus) ([]*directory.Entity, error) {
	// Note: There is no authorization required for this operation.
	res, err := m.entitiesForContact(ctx, contactValue, entityInfo, depth, statuses)
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	cacheEntities(ctx, res.Entities)
	return res.Entities, nil
}

func (m *resourceAccessor) EntitiesForExternalID(ctx context.Context, externalID string, entityInfo []directory.EntityInformation, depth int64, statuses []directory.EntityStatus) ([]*directory.Entity, error) {
	// TODO: externalID at the moment at least is always an account. this should change if that ever changes
	acc := gqlctx.Account(ctx)
	if acc == nil || acc.ID != externalID {
		return nil, errors.ErrNotAuthenticated(ctx)
	}
	res, err := m.entitiesForExternalID(ctx, externalID, entityInfo, depth, statuses)
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	cacheEntities(ctx, res.Entities)
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

func (m *resourceAccessor) MarkThreadAsRead(ctx context.Context, threadID, entityID string) error {
	// TODO: This auth pattern isn't quite right. This asserts that the caller is in the same org as the thread and the entity
	// It does not assert that the caller is the entity
	if err := m.canAccessResource(ctx, threadID, m.orgsForThread); err != nil {
		return err
	}
	if err := m.canAccessResource(ctx, entityID, m.orgsForEntity); err != nil {
		return err
	}
	if err := m.markThreadAsRead(ctx, threadID, entityID); err != nil {
		return err
	}
	return nil
}

func (m *resourceAccessor) OnboardingThreadEvent(ctx context.Context, req *threading.OnboardingThreadEventRequest) (*threading.OnboardingThreadEventResponse, error) {
	if err := m.canAccessResource(ctx, req.GetEntityID(), m.orgsForOrganization); err != nil {
		return nil, err
	}
	return m.threading.OnboardingThreadEvent(ctx, req)
}

func (m *resourceAccessor) PatientEntity(ctx context.Context, acc *models.PatientAccount) (*directory.Entity, error) {
	entities, err := m.EntitiesForExternalID(ctx, acc.GetID(), []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS}, 0, nil)
	if err != nil {
		return nil, err
	}
	if len(entities) != 1 {
		return nil, fmt.Errorf("AccountEntity: Expected to find 1 entity for external id %s but found %d", acc.GetID(), len(entities))
	}
	cacheEntities(ctx, entities)
	return entities[0], nil
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

func (m *resourceAccessor) SavedQuery(ctx context.Context, savedQueryID string) (*threading.SavedQuery, error) {
	if err := m.canAccessResource(ctx, savedQueryID, m.orgsForSavedQuery); err != nil {
		return nil, err
	}
	res, err := m.savedQuery(ctx, savedQueryID)
	if err != nil {
		return nil, err
	}
	return res.SavedQuery, nil
}

func (m *resourceAccessor) SendMessage(ctx context.Context, req *excomms.SendMessageRequest) error {
	// Note: There is currentl no authorization required for this operation.
	// TODO: Should there be?
	_, err := m.excomms.SendMessage(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func (m *resourceAccessor) SubmitCarePlan(ctx context.Context, cp *care.CarePlan, parentID string) error {
	// Authorization is the same as for CarePlan
	golog.Errorf("TODO: implement authorization for SubmitCarePlan")
	_, err := m.care.SubmitCarePlan(ctx, &care.SubmitCarePlanRequest{ID: cp.ID, ParentID: parentID})
	return err
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

func (m *resourceAccessor) ThreadMembers(ctx context.Context, orgID string, req *threading.ThreadMembersRequest) ([]*directory.Entity, error) {
	// Being a member of the thread provides access so no need to check out criteria
	res, err := m.threading.ThreadMembers(ctx, req)
	if err != nil {
		return nil, err
	}
	// Make sure viewer is a member of the thread
	acc := gqlctx.Account(ctx)
	ent, err := m.EntityForAccountID(ctx, orgID, acc.ID)
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

	entIDs := make([]string, len(res.Members))
	for i, m := range res.Members {
		entIDs[i] = m.EntityID
	}

	// Check our cache for the entities and filter anything we already have
	var depth int64
	entInfo := []directory.EntityInformation{directory.EntityInformation_CONTACTS}
	cachedEnts, notFoundEntIDs := cachedEntities(ctx, entIDs, entInfo, depth)
	if len(notFoundEntIDs) == 0 {
		return cachedEnts, nil
	}
	leres, err := m.directory.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
			BatchEntityID: &directory.IDList{
				IDs: notFoundEntIDs,
			},
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             depth,
			EntityInformation: entInfo,
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	})
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	return append(leres.Entities, cachedEnts...), nil
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
	// TODO: Add auth check that the calling account owns the viewing entity
	if req.OrganizationID != "" {
		if err := m.canAccessResource(ctx, req.OrganizationID, m.orgsForOrganization); err != nil {
			return nil, err
		}
	}
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

func (m *resourceAccessor) UnauthorizedEntity(ctx context.Context, entityID string, entityInfo []directory.EntityInformation, depth int64) (*directory.Entity, error) {
	res, err := m.entity(ctx, entityID, entityInfo, depth, nil)
	if err != nil {
		return nil, err
	}
	return res.Entities[0], nil
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

func (m *resourceAccessor) UpdateThread(ctx context.Context, req *threading.UpdateThreadRequest) (*threading.UpdateThreadResponse, error) {
	if err := m.canAccessResource(ctx, req.ThreadID, m.orgsForThread); err != nil {
		return nil, err
	}
	return m.threading.UpdateThread(ctx, req)
}

func (m *resourceAccessor) VisitLayout(ctx context.Context, req *layout.GetVisitLayoutRequest) (*layout.GetVisitLayoutResponse, error) {
	if !m.isAccountType(ctx, auth.AccountType_PROVIDER) {
		return nil, errors.ErrNotAuthorized(ctx, req.ID)
	}

	res, err := m.layout.GetVisitLayout(ctx, req)
	if grpc.Code(err) == codes.NotFound {
		return nil, errors.ErrNotFound(ctx, req.ID)
	} else if err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	return res, nil
}

func (m *resourceAccessor) CreateVisit(ctx context.Context, req *care.CreateVisitRequest) (*care.CreateVisitResponse, error) {
	if !m.isAccountType(ctx, auth.AccountType_PROVIDER) {
		return nil, errors.ErrNotAuthorized(ctx, req.LayoutVersionID)
	}

	if err := m.canAccessResource(ctx, req.EntityID, m.orgsForEntity); err != nil {
		return nil, err
	}

	res, err := m.care.CreateVisit(ctx, req)
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	return res, nil
}

func (m *resourceAccessor) Visit(ctx context.Context, req *care.GetVisitRequest) (*care.GetVisitResponse, error) {
	// first get the visit then check whether or not caller can access resource
	res, err := m.care.GetVisit(ctx, req)
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return nil, errors.ErrNotFound(ctx, req.ID)
		}

		return nil, errors.InternalError(ctx, err)
	}

	if err := m.canAccessResource(ctx, res.Visit.EntityID, m.orgsForEntity); err != nil {
		return nil, err
	}

	return res, nil
}

func (m *resourceAccessor) SubmitVisit(ctx context.Context, req *care.SubmitVisitRequest) (*care.SubmitVisitResponse, error) {
	_, err := m.Visit(ctx, &care.GetVisitRequest{
		ID: req.VisitID,
	})
	if err != nil {
		return nil, err
	}

	if !m.isAccountType(ctx, auth.AccountType_PATIENT) {
		return nil, errors.ErrNotAuthorized(ctx, req.VisitID)
	}

	return m.care.SubmitVisit(ctx, req)
}

func (m *resourceAccessor) VisitLayoutVersion(ctx context.Context, req *layout.GetVisitLayoutVersionRequest) (*layout.GetVisitLayoutVersionResponse, error) {

	res, err := m.layout.GetVisitLayoutVersion(ctx, req)
	if grpc.Code(err) == codes.NotFound {
		return nil, errors.ErrNotFound(ctx, req.ID)
	} else if err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	return res, nil
}

func (m *resourceAccessor) CreateVisitAnswers(ctx context.Context, req *care.CreateVisitAnswersRequest) (*care.CreateVisitAnswersResponse, error) {
	// only the patient can submit answers
	if !m.isAccountType(ctx, auth.AccountType_PATIENT) {
		return nil, errors.ErrNotAuthorized(ctx, req.VisitID)
	}

	if err := m.canAccessResource(ctx, req.ActorEntityID, m.orgsForEntity); err != nil {
		return nil, err
	}

	res, err := m.care.CreateVisitAnswers(ctx, req)
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return nil, errors.ErrNotFound(ctx, req.VisitID)
		}
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) GetAnswersForVisit(ctx context.Context, req *care.GetAnswersForVisitRequest) (*care.GetAnswersForVisitResponse, error) {
	_, err := m.Visit(ctx, &care.GetVisitRequest{
		ID: req.VisitID,
	})
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	res, err := m.care.GetAnswersForVisit(ctx, req)
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	return res, nil
}

func (m *resourceAccessor) isAccountType(ctx context.Context, accType auth.AccountType) bool {
	acc := gqlctx.Account(ctx)
	return acc != nil && acc.Type == accType
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
	res, err := m.entity(ctx, entityID, []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS}, 0, nil)
	if err != nil {
		return nil, err
	}
	return orgsForEntity(res.Entities[0]), nil
}

func (m *resourceAccessor) orgsForEntityForExternalID(ctx context.Context, externalID string) (map[string]struct{}, error) {
	// Don't do any status checks. Authorization is for all existing resources
	res, err := m.entitiesForExternalID(ctx, externalID, []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS}, 0, nil)
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

func (m *resourceAccessor) orgsForSavedQuery(ctx context.Context, savedQueryID string) (map[string]struct{}, error) {
	res, err := m.savedQuery(ctx, savedQueryID)
	if err != nil {
		return nil, err
	}
	return map[string]struct{}{res.SavedQuery.OrganizationID: {}}, nil
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

func (m *resourceAccessor) entity(ctx context.Context, entityID string, entityInfo []directory.EntityInformation, depth int64, statuses []directory.EntityStatus) (*directory.LookupEntitiesResponse, error) {
	if len(entityInfo) == 0 {
		entityInfo = []directory.EntityInformation{
			directory.EntityInformation_MEMBERSHIPS,
			directory.EntityInformation_CONTACTS,
		}
	}
	res, err := m.directory.LookupEntities(ctx,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: entityID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth:             depth,
				EntityInformation: entityInfo,
			},
			Statuses: statuses,
		})
	if err != nil {
		return nil, err
	}
	if len(res.Entities) != 1 {
		return nil, errors.InternalError(ctx, fmt.Errorf("Expected only 1 entity to be returned for id %s but found %d", entityID, len(res.Entities)))
	}
	return res, nil
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

func (m *resourceAccessor) entitiesForContact(ctx context.Context, contactValue string, entityInfo []directory.EntityInformation, depth int64, statuses []directory.EntityStatus) (*directory.LookupEntitiesByContactResponse, error) {
	if len(entityInfo) == 0 {
		entityInfo = []directory.EntityInformation{
			directory.EntityInformation_MEMBERSHIPS,
			directory.EntityInformation_CONTACTS,
		}
	}
	res, err := m.directory.LookupEntitiesByContact(ctx,
		&directory.LookupEntitiesByContactRequest{
			ContactValue: contactValue,
			RequestedInformation: &directory.RequestedInformation{
				Depth:             depth,
				EntityInformation: entityInfo,
			},
			Statuses: statuses,
		})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) entitiesForExternalID(ctx context.Context, externalID string, entityInfo []directory.EntityInformation, depth int64, statuses []directory.EntityStatus) (*directory.LookupEntitiesResponse, error) {
	if len(entityInfo) == 0 {
		entityInfo = []directory.EntityInformation{
			directory.EntityInformation_MEMBERSHIPS,
			directory.EntityInformation_CONTACTS,
		}
	}
	res, err := m.directory.LookupEntities(ctx,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: externalID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth:             depth,
				EntityInformation: entityInfo,
			},
			Statuses: statuses,
		})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *resourceAccessor) markThreadAsRead(ctx context.Context, threadID, entityID string) error {
	_, err := m.threading.MarkThreadAsRead(ctx, &threading.MarkThreadAsReadRequest{
		ThreadID: threadID,
		EntityID: entityID,
	})
	if err != nil {
		return err
	}
	return nil
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
