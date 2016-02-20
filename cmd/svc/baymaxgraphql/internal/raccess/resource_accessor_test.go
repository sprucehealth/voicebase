package raccess

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	amock "github.com/sprucehealth/backend/svc/auth/mock"
	"github.com/sprucehealth/backend/svc/directory"
	dmock "github.com/sprucehealth/backend/svc/directory/mock"
	emock "github.com/sprucehealth/backend/svc/excomms/mock"
	"github.com/sprucehealth/backend/svc/threading"
	tmock "github.com/sprucehealth/backend/svc/threading/mock"
	"github.com/sprucehealth/backend/test"
)

type ratest struct {
	aC *amock.Client
	dC *dmock.Client
	tC *tmock.Client
	eC *emock.Client
	ra ResourceAccessor
}

func (r *ratest) finish() {
	mock.FinishAll(r.aC, r.dC, r.eC, r.tC)
}

func new(t *testing.T) *ratest {
	var rat ratest
	rat.aC = amock.New(t)
	rat.dC = dmock.New(t)
	rat.tC = tmock.New(t)
	rat.eC = emock.New(t)
	rat.ra = New(rat.aC, rat.dC, rat.tC, rat.eC)
	return &rat
}

func TestAccessAccount(t *testing.T) {
	accountID := "account_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)
	rat := new(t)
	defer rat.finish()

	rat.aC.Expect(mock.NewExpectation(rat.aC.GetAccount, &auth.GetAccountRequest{
		AccountID: accountID,
	}).WithReturns(&auth.GetAccountResponse{Account: &auth.Account{ID: accountID}}, nil))

	rAcc, err := rat.ra.Account(ctx, accountID)
	test.OK(t, err)
	test.Equals(t, &auth.Account{ID: accountID}, rAcc)
}

func TestAccessAccountNotSameAccount(t *testing.T) {
	accountID := "account_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID + "1",
	}
	ctx = gqlctx.WithAccount(ctx, acc)
	rat := new(t)
	defer rat.finish()

	rAcc, err := rat.ra.Account(ctx, accountID)
	test.AssertNil(t, rAcc)
	test.Equals(t, errors.ErrTypeNotAuthorized, errors.Type(err))
}

func TestAuthenticateLogin(t *testing.T) {
	accountID := "account_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()
	email := "email"
	password := "password"
	rat.aC.Expect(mock.NewExpectation(rat.aC.AuthenticateLoginWithCode, &auth.AuthenticateLoginRequest{
		Email:    email,
		Password: password,
	}).WithReturns(&auth.AuthenticateLoginResponse{Account: &auth.Account{ID: accountID}}, nil))

	resp, err := rat.ra.AuthenticateLogin(ctx, email, password)
	test.OK(t, err)
	test.Equals(t, &auth.AuthenticateLoginResponse{Account: &auth.Account{ID: accountID}}, resp)
}

func TestAuthenticateLoginWithCode(t *testing.T) {
	accountID := "account_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()
	token := "token"
	code := "code"
	rat.aC.Expect(mock.NewExpectation(rat.aC.AuthenticateLoginWithCode, &auth.AuthenticateLoginWithCodeRequest{
		Token: token,
		Code:  code,
	}).WithReturns(&auth.AuthenticateLoginWithCodeResponse{Account: &auth.Account{ID: accountID}}, nil))

	resp, err := rat.ra.AuthenticateLoginWithCode(ctx, token, code)
	test.OK(t, err)
	test.Equals(t, &auth.AuthenticateLoginWithCodeResponse{Account: &auth.Account{ID: accountID}}, resp)
}

func TestCheckPasswordResetToken(t *testing.T) {
	accountID := "account_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()
	token := "token"
	rat.aC.Expect(mock.NewExpectation(rat.aC.CheckPasswordResetToken, &auth.CheckPasswordResetTokenRequest{
		Token: token,
	}).WithReturns(&auth.CheckPasswordResetTokenResponse{
		AccountID:          accountID,
		AccountEmail:       "my@email.com",
		AccountPhoneNumber: "+1234567890",
	}, nil))

	resp, err := rat.ra.CheckPasswordResetToken(ctx, token)
	test.OK(t, err)
	test.Equals(t, &auth.CheckPasswordResetTokenResponse{
		AccountID:          accountID,
		AccountEmail:       "my@email.com",
		AccountPhoneNumber: "+1234567890",
	}, resp)
}

func TestCheckVerificationCode(t *testing.T) {
	accountID := "account_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()
	token := "token"
	code := "code"
	rat.aC.Expect(mock.NewExpectation(rat.aC.CheckVerificationCode, &auth.CheckVerificationCodeRequest{
		Token: token,
		Code:  code,
	}).WithReturns(&auth.CheckVerificationCodeResponse{Value: "hello"}, nil))

	resp, err := rat.ra.CheckVerificationCode(ctx, token, code)
	test.OK(t, err)
	test.Equals(t, &auth.CheckVerificationCodeResponse{Value: "hello"}, resp)
}

func TestCreateAccount(t *testing.T) {
	accountID := "account_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()
	rat.aC.Expect(mock.NewExpectation(rat.aC.CreateAccount, &auth.CreateAccountRequest{
		FirstName: "name",
	}).WithReturns(&auth.CreateAccountResponse{Account: &auth.Account{ID: "Hi"}}, nil))

	resp, err := rat.ra.CreateAccount(ctx, &auth.CreateAccountRequest{
		FirstName: "name",
	})
	test.OK(t, err)
	test.Equals(t, &auth.CreateAccountResponse{Account: &auth.Account{ID: "Hi"}}, resp)
}

func expectOrgsForEntity(rat *ratest, entityID, orgID string) {
	rat.dC.Expect(mock.NewExpectation(rat.dC.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   entityID,
				Type: directory.EntityType_INTERNAL,
				Memberships: []*directory.Entity{
					{
						ID:   orgID,
						Type: directory.EntityType_ORGANIZATION,
					},
				},
			},
		},
	}, nil))
}

func expectOrgsForEntityForExternalID(rat *ratest, externalID, orgID string) {
	rat.dC.Expect(mock.NewExpectation(rat.dC.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: externalID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   externalID,
				Type: directory.EntityType_INTERNAL,
				Memberships: []*directory.Entity{
					{
						ID:   orgID,
						Type: directory.EntityType_ORGANIZATION,
					},
				},
			},
		},
	}, nil))
}

func expectOrgsForThread(rat *ratest, threadID, orgID string) {
	rat.tC.Expect(mock.NewExpectation(rat.tC.Thread, &threading.ThreadRequest{
		ThreadID:       threadID,
		ViewerEntityID: "",
	}).WithReturns(&threading.ThreadResponse{
		Thread: &threading.Thread{
			OrganizationID: orgID,
		},
	}, nil))
}

func TestCreateContact(t *testing.T) {
	accountID := "account_12345"
	entityID := "entity_12345"
	orgID := "org_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForEntity(rat, entityID, orgID)
	expectOrgsForEntityForExternalID(rat, accountID, orgID)
	rat.dC.Expect(mock.NewExpectation(rat.dC.CreateContact, &directory.CreateContactRequest{
		EntityID: entityID,
	}).WithReturns(&directory.CreateContactResponse{Entity: &directory.Entity{ID: entityID}}, nil))

	resp, err := rat.ra.CreateContact(ctx, &directory.CreateContactRequest{
		EntityID: entityID,
	})
	test.OK(t, err)
	test.Equals(t, &directory.CreateContactResponse{Entity: &directory.Entity{ID: entityID}}, resp)
}

func TestCreateContactNotAuthorized(t *testing.T) {
	accountID := "account_12345"
	entityID := "entity_12345"
	orgID1 := "org_12345"
	orgID2 := "org_67890"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForEntity(rat, entityID, orgID1)
	expectOrgsForEntityForExternalID(rat, accountID, orgID2)

	resp, err := rat.ra.CreateContact(ctx, &directory.CreateContactRequest{
		EntityID: entityID,
	})
	test.AssertNil(t, resp)
	test.Equals(t, errors.ErrTypeNotAuthorized, errors.Type(err))
}

func TestCreateContacts(t *testing.T) {
	accountID := "account_12345"
	entityID := "entity_12345"
	orgID := "org_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForEntity(rat, entityID, orgID)
	expectOrgsForEntityForExternalID(rat, accountID, orgID)
	rat.dC.Expect(mock.NewExpectation(rat.dC.CreateContact, &directory.CreateContactsRequest{
		EntityID: entityID,
	}).WithReturns(&directory.CreateContactsResponse{Entity: &directory.Entity{ID: entityID}}, nil))

	resp, err := rat.ra.CreateContacts(ctx, &directory.CreateContactsRequest{
		EntityID: entityID,
	})
	test.OK(t, err)
	test.Equals(t, &directory.CreateContactsResponse{Entity: &directory.Entity{ID: entityID}}, resp)
}

func TestCreateContactsNotAuthorized(t *testing.T) {
	accountID := "account_12345"
	entityID := "entity_12345"
	orgID1 := "org_12345"
	orgID2 := "org_67890"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForEntity(rat, entityID, orgID1)
	expectOrgsForEntityForExternalID(rat, accountID, orgID2)

	resp, err := rat.ra.CreateContacts(ctx, &directory.CreateContactsRequest{
		EntityID: entityID,
	})
	test.AssertNil(t, resp)
	test.Equals(t, errors.ErrTypeNotAuthorized, errors.Type(err))
}

func TestCreateEmptyThread(t *testing.T) {
	accountID := "account_12345"
	orgID := "org_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForEntityForExternalID(rat, accountID, orgID)
	rat.tC.Expect(mock.NewExpectation(rat.tC.CreateEmptyThread, &threading.CreateEmptyThreadRequest{
		OrganizationID: orgID,
	}).WithReturns(&threading.CreateEmptyThreadResponse{Thread: &threading.Thread{ID: "id"}}, nil))

	resp, err := rat.ra.CreateEmptyThread(ctx, &threading.CreateEmptyThreadRequest{
		OrganizationID: orgID,
	})
	test.OK(t, err)
	test.Equals(t, &threading.Thread{ID: "id"}, resp)
}

func TestCreateEmptyThreadNotAuthorized(t *testing.T) {
	accountID := "account_12345"
	orgID1 := "org_12345"
	orgID2 := "org_67890"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForEntityForExternalID(rat, accountID, orgID2)

	resp, err := rat.ra.CreateEmptyThread(ctx, &threading.CreateEmptyThreadRequest{
		OrganizationID: orgID1,
	})
	test.AssertNil(t, resp)
	test.Equals(t, errors.ErrTypeNotAuthorized, errors.Type(err))
}

func TestCreateEntity(t *testing.T) {
	accountID := "account_12345"
	entityID := "entity_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	rat.dC.Expect(mock.NewExpectation(rat.dC.CreateEntity, &directory.CreateEntityRequest{
		ExternalID: accountID,
	}).WithReturns(&directory.CreateEntityResponse{Entity: &directory.Entity{ID: entityID}}, nil))

	resp, err := rat.ra.CreateEntity(ctx, &directory.CreateEntityRequest{
		ExternalID: accountID,
	})
	test.OK(t, err)
	test.Equals(t, &directory.Entity{ID: entityID}, resp)
}

func TestCreateEntityDomain(t *testing.T) {
	accountID := "account_12345"
	orgID := "org_12345"
	subdomain := "subdomain"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForEntityForExternalID(rat, accountID, orgID)
	rat.dC.Expect(mock.NewExpectation(rat.dC.CreateEntityDomain, &directory.CreateEntityDomainRequest{
		EntityID: orgID,
		Domain:   subdomain,
	}).WithReturns(&directory.CreateEntityDomainResponse{}, nil))

	err := rat.ra.CreateEntityDomain(ctx, orgID, subdomain)
	test.OK(t, err)
}

func TestCreateEntityDomainNotAuthorized(t *testing.T) {
	accountID := "account_12345"
	orgID := "org_12345"
	subdomain := "subdomain"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForEntityForExternalID(rat, accountID, orgID+"1")

	err := rat.ra.CreateEntityDomain(ctx, orgID, subdomain)
	test.Equals(t, errors.ErrTypeNotAuthorized, errors.Type(err))
}

func TestCreatePasswordResetToken(t *testing.T) {
	email := "email"
	ctx := context.Background()
	rat := new(t)
	defer rat.finish()

	rat.aC.Expect(mock.NewExpectation(rat.aC.CreatePasswordResetToken, &auth.CreatePasswordResetTokenRequest{
		Email: email,
	}).WithReturns(&auth.CreatePasswordResetTokenResponse{Token: "token"}, nil))

	resp, err := rat.ra.CreatePasswordResetToken(ctx, email)
	test.OK(t, err)
	test.Equals(t, &auth.CreatePasswordResetTokenResponse{Token: "token"}, resp)
}

func TestCreateSavedQuery(t *testing.T) {
	accountID := "account_12345"
	orgID := "org_12345"
	entityID := "entity_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForEntity(rat, entityID, orgID)
	expectOrgsForEntityForExternalID(rat, accountID, orgID)
	expectOrgsForEntityForExternalID(rat, accountID, orgID)
	rat.tC.Expect(mock.NewExpectation(rat.tC.CreateSavedQuery, &threading.CreateSavedQueryRequest{
		OrganizationID: orgID,
		EntityID:       entityID,
	}).WithReturns(&threading.CreateSavedQueryResponse{}, nil))

	err := rat.ra.CreateSavedQuery(ctx, &threading.CreateSavedQueryRequest{
		OrganizationID: orgID,
		EntityID:       entityID,
	})
	test.OK(t, err)
}

func TestCreateSavedQueryNotAuthorizedEntity(t *testing.T) {
	accountID := "account_12345"
	orgID := "org_12345"
	orgID2 := "org_67890"
	entityID := "entity_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForEntity(rat, entityID, orgID2)
	expectOrgsForEntityForExternalID(rat, accountID, orgID)

	err := rat.ra.CreateSavedQuery(ctx, &threading.CreateSavedQueryRequest{
		OrganizationID: orgID,
		EntityID:       entityID,
	})
	test.Equals(t, errors.ErrTypeNotAuthorized, errors.Type(err))
}

func TestCreateSavedQueryNotAuthorizedOrganization(t *testing.T) {
	accountID := "account_12345"
	orgID := "org_12345"
	orgID2 := "org_67890"
	entityID := "entity_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForEntity(rat, entityID, orgID)
	expectOrgsForEntityForExternalID(rat, accountID, orgID)
	expectOrgsForEntityForExternalID(rat, accountID, orgID)
	err := rat.ra.CreateSavedQuery(ctx, &threading.CreateSavedQueryRequest{
		OrganizationID: orgID2,
		EntityID:       entityID,
	})
	test.Equals(t, errors.ErrTypeNotAuthorized, errors.Type(err))
}

func TestCreateVerificationToken(t *testing.T) {
	accountID := "account_12345"
	token := "token"
	ctx := context.Background()
	rat := new(t)
	defer rat.finish()

	rat.aC.Expect(mock.NewExpectation(rat.aC.CreateVerificationCode, &auth.CreateVerificationCodeRequest{
		Type:          auth.VerificationCodeType_ACCOUNT_2FA,
		ValueToVerify: accountID,
	}).WithReturns(&auth.CreateVerificationCodeResponse{VerificationCode: &auth.VerificationCode{
		Token: token,
	}}, nil))

	resp, err := rat.ra.CreateVerificationCode(ctx, auth.VerificationCodeType_ACCOUNT_2FA, accountID)
	test.OK(t, err)
	test.Equals(t, &auth.CreateVerificationCodeResponse{VerificationCode: &auth.VerificationCode{
		Token: token,
	}}, resp)
}

func TestDeleteContacts(t *testing.T) {
	accountID := "account_12345"
	orgID := "org_12345"
	entityID := "entity_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForEntity(rat, entityID, orgID)
	expectOrgsForEntityForExternalID(rat, accountID, orgID)
	rat.dC.Expect(mock.NewExpectation(rat.dC.DeleteContacts, &directory.DeleteContactsRequest{
		EntityID: entityID,
	}).WithReturns(&directory.DeleteContactsResponse{Entity: &directory.Entity{ID: entityID}}, nil))

	resp, err := rat.ra.DeleteContacts(ctx, &directory.DeleteContactsRequest{
		EntityID: entityID,
	})
	test.OK(t, err)
	test.Equals(t, &directory.Entity{ID: entityID}, resp)
}

func TestDeleteContactsNotAuthorized(t *testing.T) {
	accountID := "account_12345"
	orgID := "org_12345"
	orgID2 := "org_67890"
	entityID := "entity_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForEntity(rat, entityID, orgID2)
	expectOrgsForEntityForExternalID(rat, accountID, orgID)

	resp, err := rat.ra.DeleteContacts(ctx, &directory.DeleteContactsRequest{
		EntityID: entityID,
	})
	test.AssertNil(t, resp)
	test.Equals(t, errors.ErrTypeNotAuthorized, errors.Type(err))
}

func TestDeleteThread(t *testing.T) {
	accountID := "account_12345"
	orgID := "org_12345"
	entityID := "entity_12345"
	threadID := "t_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForThread(rat, threadID, orgID)
	expectOrgsForEntityForExternalID(rat, accountID, orgID)
	expectOrgsForEntity(rat, entityID, orgID)
	expectOrgsForEntityForExternalID(rat, accountID, orgID)
	rat.tC.Expect(mock.NewExpectation(rat.tC.DeleteThread, &threading.DeleteThreadRequest{
		ActorEntityID: entityID,
		ThreadID:      threadID,
	}).WithReturns(&threading.DeleteThreadResponse{}, nil))

	err := rat.ra.DeleteThread(ctx, threadID, entityID)
	test.OK(t, err)
}

func TestDeleteThreadNotAuthorizedThread(t *testing.T) {
	accountID := "account_12345"
	orgID := "org_12345"
	orgID2 := "org_67890"
	entityID := "entity_12345"
	threadID := "t_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForThread(rat, threadID, orgID2)
	expectOrgsForEntityForExternalID(rat, accountID, orgID)

	err := rat.ra.DeleteThread(ctx, threadID, entityID)
	test.Equals(t, errors.ErrTypeNotAuthorized, errors.Type(err))
}

func TestDeleteThreadNotAuthorizedEntity(t *testing.T) {
	accountID := "account_12345"
	orgID := "org_12345"
	orgID2 := "org_67890"
	entityID := "entity_12345"
	threadID := "t_12345"
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForThread(rat, threadID, orgID)
	expectOrgsForEntityForExternalID(rat, accountID, orgID)
	expectOrgsForEntity(rat, entityID, orgID2)
	expectOrgsForEntityForExternalID(rat, accountID, orgID)

	err := rat.ra.DeleteThread(ctx, threadID, entityID)
	test.Equals(t, errors.ErrTypeNotAuthorized, errors.Type(err))
}

func TestEntity(t *testing.T) {
	accountID := "account_12345"
	orgID := "org_12345"
	entityID := "entity_12345"
	depth := int64(0)
	entityInfo := []directory.EntityInformation{
		directory.EntityInformation_MEMBERSHIPS,
		directory.EntityInformation_CONTACTS,
	}
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForEntity(rat, entityID, orgID)
	expectOrgsForEntityForExternalID(rat, accountID, orgID)
	rat.dC.Expect(mock.NewExpectation(rat.dC.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             depth,
			EntityInformation: entityInfo,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{Entities: []*directory.Entity{
		{ID: entityID},
	}}, nil))

	resp, err := rat.ra.Entity(ctx, entityID, entityInfo, depth)
	test.OK(t, err)
	test.Equals(t, &directory.Entity{ID: entityID}, resp)
}

func TestEntityNotAuthorized(t *testing.T) {
	accountID := "account_12345"
	orgID := "org_12345"
	orgID2 := "org_67890"
	entityID := "entity_12345"
	depth := int64(0)
	entityInfo := []directory.EntityInformation{
		directory.EntityInformation_MEMBERSHIPS,
		directory.EntityInformation_CONTACTS,
	}
	ctx := context.Background()
	acc := &models.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForEntity(rat, entityID, orgID)
	expectOrgsForEntityForExternalID(rat, accountID, orgID2)

	resp, err := rat.ra.Entity(ctx, entityID, entityInfo, depth)
	test.AssertNil(t, resp)
	test.Equals(t, errors.ErrTypeNotAuthorized, errors.Type(err))
}