package mock

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
)

var _ raccess.ResourceAccessor = &ResourceAccessor{}

type ResourceAccessor struct {
	*mock.Expector
}

func New(t *testing.T) *ResourceAccessor {
	return &ResourceAccessor{
		&mock.Expector{T: t},
	}
}

func (m *ResourceAccessor) Account(ctx context.Context, accountID string) (*auth.Account, error) {
	rets := m.Record(accountID)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*auth.Account), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) AuthenticateLogin(ctx context.Context, email, password string) (*auth.AuthenticateLoginResponse, error) {
	rets := m.Record(email, password)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*auth.AuthenticateLoginResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) AuthenticateLoginWithCode(ctx context.Context, token, code string) (*auth.AuthenticateLoginWithCodeResponse, error) {
	rets := m.Record(token, code)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*auth.AuthenticateLoginWithCodeResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) CheckPasswordResetToken(ctx context.Context, token string) (*auth.CheckPasswordResetTokenResponse, error) {
	rets := m.Record(token)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*auth.CheckPasswordResetTokenResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) CheckVerificationCode(ctx context.Context, token, code string) (*auth.CheckVerificationCodeResponse, error) {
	rets := m.Record(token, code)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*auth.CheckVerificationCodeResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) CarePlan(ctx context.Context, id string) (*care.CarePlan, error) {
	rets := m.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*care.CarePlan), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) CreateAccount(ctx context.Context, req *auth.CreateAccountRequest) (*auth.CreateAccountResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*auth.CreateAccountResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) CreateCarePlan(ctx context.Context, req *care.CreateCarePlanRequest) (*care.CreateCarePlanResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*care.CreateCarePlanResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) CreateContact(ctx context.Context, req *directory.CreateContactRequest) (*directory.CreateContactResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*directory.CreateContactResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) CreateContacts(ctx context.Context, req *directory.CreateContactsRequest) (*directory.CreateContactsResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*directory.CreateContactsResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) CreateEmptyThread(ctx context.Context, req *threading.CreateEmptyThreadRequest) (*threading.Thread, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*threading.Thread), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) CreateEntity(ctx context.Context, req *directory.CreateEntityRequest) (*directory.Entity, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*directory.Entity), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) CreateEntityDomain(ctx context.Context, organizationID, subdomain string) error {
	rets := m.Record(organizationID, subdomain)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (m *ResourceAccessor) CreateExternalIDs(ctx context.Context, req *directory.CreateExternalIDsRequest) error {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (m *ResourceAccessor) CreateLinkedThreads(ctx context.Context, req *threading.CreateLinkedThreadsRequest) (*threading.CreateLinkedThreadsResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.CreateLinkedThreadsResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) CreateOnboardingThread(ctx context.Context, req *threading.CreateOnboardingThreadRequest) (*threading.CreateOnboardingThreadResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.CreateOnboardingThreadResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) CreatePasswordResetToken(ctx context.Context, email string) (*auth.CreatePasswordResetTokenResponse, error) {
	rets := m.Record(email)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*auth.CreatePasswordResetTokenResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) CreateSavedQuery(ctx context.Context, req *threading.CreateSavedQueryRequest) error {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (m *ResourceAccessor) CreateVerificationCode(ctx context.Context, codeType auth.VerificationCodeType, valueToVerify string) (*auth.CreateVerificationCodeResponse, error) {
	rets := m.Record(codeType, valueToVerify)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*auth.CreateVerificationCodeResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) DeleteContacts(ctx context.Context, req *directory.DeleteContactsRequest) (*directory.Entity, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*directory.Entity), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) DeleteThread(ctx context.Context, threadID, entityID string) error {
	rets := m.Record(threadID, entityID)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (m *ResourceAccessor) EntityDomain(ctx context.Context, entityID, domain string) (*directory.LookupEntityDomainResponse, error) {
	rets := m.Record(entityID, domain)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*directory.LookupEntityDomainResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) EntitiesByContact(ctx context.Context, req *directory.LookupEntitiesByContactRequest) ([]*directory.Entity, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].([]*directory.Entity), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) Entities(ctx context.Context, req *directory.LookupEntitiesRequest, opts ...raccess.EntityQueryOption) ([]*directory.Entity, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].([]*directory.Entity), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) InitiatePhoneCall(ctx context.Context, req *excomms.InitiatePhoneCallRequest) (*excomms.InitiatePhoneCallResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*excomms.InitiatePhoneCallResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) MarkThreadAsRead(ctx context.Context, threadID, entityID string) error {
	rets := m.Record(threadID, entityID)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (m *ResourceAccessor) OnboardingThreadEvent(ctx context.Context, req *threading.OnboardingThreadEventRequest) (*threading.OnboardingThreadEventResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.OnboardingThreadEventResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) CanPostMessage(ctx context.Context, threadID string) error {
	rets := m.Record(threadID)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[1])
}

func (m *ResourceAccessor) PostMessage(ctx context.Context, req *threading.PostMessageRequest) (*threading.PostMessageResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*threading.PostMessageResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) ProvisionPhoneNumber(ctx context.Context, req *excomms.ProvisionPhoneNumberRequest) (*excomms.ProvisionPhoneNumberResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*excomms.ProvisionPhoneNumberResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) ProvisionEmailAddress(ctx context.Context, req *excomms.ProvisionEmailAddressRequest) (*excomms.ProvisionEmailAddressResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*excomms.ProvisionEmailAddressResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) QueryThreads(ctx context.Context, req *threading.QueryThreadsRequest) (*threading.QueryThreadsResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*threading.QueryThreadsResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) SavedQuery(ctx context.Context, savedQueryID string) (*threading.SavedQuery, error) {
	rets := m.Record(savedQueryID)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*threading.SavedQuery), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) SavedQueries(ctx context.Context, entityID string) ([]*threading.SavedQuery, error) {
	rets := m.Record(entityID)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].([]*threading.SavedQuery), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) SearchMedications(ctx context.Context, req *care.SearchMedicationsRequest) (*care.SearchMedicationsResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*care.SearchMedicationsResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) SendMessage(ctx context.Context, req *excomms.SendMessageRequest) error {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (m *ResourceAccessor) SerializedEntityContact(ctx context.Context, entityID string, platform directory.Platform) (*directory.SerializedClientEntityContact, error) {
	rets := m.Record(entityID, platform)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*directory.SerializedClientEntityContact), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) SubmitCarePlan(ctx context.Context, cp *care.CarePlan, parentID string) error {
	rets := m.Record(cp, parentID)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (m *ResourceAccessor) Thread(ctx context.Context, threadID, viewerEntityID string) (*threading.Thread, error) {
	rets := m.Record(threadID, viewerEntityID)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*threading.Thread), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) ThreadItem(ctx context.Context, threadItemID string) (*threading.ThreadItem, error) {
	rets := m.Record(threadItemID)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*threading.ThreadItem), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) ThreadItems(ctx context.Context, req *threading.ThreadItemsRequest) (*threading.ThreadItemsResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*threading.ThreadItemsResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) ThreadItemViewDetails(ctx context.Context, threadItemID string) ([]*threading.ThreadItemViewDetails, error) {
	rets := m.Record(threadItemID)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].([]*threading.ThreadItemViewDetails), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) ThreadsForMember(ctx context.Context, entityID string, primaryOnly bool) ([]*threading.Thread, error) {
	rets := m.Record(entityID, primaryOnly)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].([]*threading.Thread), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) ThreadMembers(ctx context.Context, orgID string, req *threading.ThreadMembersRequest) ([]*directory.Entity, error) {
	rets := m.Record(orgID, req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].([]*directory.Entity), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) Unauthenticate(ctx context.Context, token string) error {
	rets := m.Record(token)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (m *ResourceAccessor) UnauthorizedCreateExternalIDs(ctx context.Context, req *directory.CreateExternalIDsRequest) error {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (m *ResourceAccessor) UpdateContacts(ctx context.Context, req *directory.UpdateContactsRequest) (*directory.Entity, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*directory.Entity), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) UpdateEntity(ctx context.Context, req *directory.UpdateEntityRequest) (*directory.Entity, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*directory.Entity), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) UpdatePassword(ctx context.Context, token, code, newPassword string) error {
	rets := m.Record(token, code, newPassword)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (m *ResourceAccessor) UpdateThread(ctx context.Context, req *threading.UpdateThreadRequest) (*threading.UpdateThreadResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*threading.UpdateThreadResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) VisitLayout(ctx context.Context, req *layout.GetVisitLayoutRequest) (*layout.GetVisitLayoutResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*layout.GetVisitLayoutResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) CreateVisit(ctx context.Context, req *care.CreateVisitRequest) (*care.CreateVisitResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*care.CreateVisitResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) Visit(ctx context.Context, req *care.GetVisitRequest) (*care.GetVisitResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*care.GetVisitResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) SubmitVisit(ctx context.Context, req *care.SubmitVisitRequest) (*care.SubmitVisitResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*care.SubmitVisitResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) VisitLayoutVersion(ctx context.Context, req *layout.GetVisitLayoutVersionRequest) (*layout.GetVisitLayoutVersionResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*layout.GetVisitLayoutVersionResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) VerifiedValue(ctx context.Context, token string) (string, error) {
	rets := m.Record(token)
	if len(rets) == 0 {
		return "", nil
	}

	return rets[0].(string), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) CreateVisitAnswers(ctx context.Context, req *care.CreateVisitAnswersRequest) (*care.CreateVisitAnswersResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*care.CreateVisitAnswersResponse), mock.SafeError(rets[1])
}

func (m *ResourceAccessor) GetAnswersForVisit(ctx context.Context, req *care.GetAnswersForVisitRequest) (*care.GetAnswersForVisitResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*care.GetAnswersForVisitResponse), mock.SafeError(rets[1])
}
