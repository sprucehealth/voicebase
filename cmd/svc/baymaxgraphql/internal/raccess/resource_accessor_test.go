package raccess

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	amock "github.com/sprucehealth/backend/svc/auth/mock"
	vmock "github.com/sprucehealth/backend/svc/care/mock"
	"github.com/sprucehealth/backend/svc/directory"
	dmock "github.com/sprucehealth/backend/svc/directory/mock"
	emock "github.com/sprucehealth/backend/svc/excomms/mock"
	lmock "github.com/sprucehealth/backend/svc/layout/mock"
	mmock "github.com/sprucehealth/backend/svc/media/mock"
	psmock "github.com/sprucehealth/backend/svc/patientsync/mock"
	pmock "github.com/sprucehealth/backend/svc/payments/mock"
	"github.com/sprucehealth/backend/svc/threading"
	tmock "github.com/sprucehealth/backend/svc/threading/mock"
)

type ratest struct {
	aC  *amock.Client
	dC  *dmock.Client
	tC  *tmock.Client
	eC  *emock.Client
	lC  *lmock.Client
	vC  *vmock.Client
	mC  *mmock.Client
	pC  *pmock.Client
	psC *psmock.Client
	ra  ResourceAccessor
}

func (r *ratest) finish() {
	mock.FinishAll(r.aC, r.dC, r.eC, r.tC, r.lC, r.vC, r.mC)
}

func new(t *testing.T) *ratest {
	var rat ratest
	rat.aC = amock.New(t)
	rat.dC = dmock.New(t)
	rat.tC = tmock.New(t)
	rat.eC = emock.New(t)
	rat.lC = lmock.New(t)
	rat.vC = vmock.New(t)
	rat.mC = mmock.New(t)
	rat.pC = pmock.New(t)
	rat.psC = psmock.New(t)
	rat.ra = New(rat.aC, rat.dC, rat.tC, rat.eC, rat.lC, rat.vC, rat.mC, rat.pC, rat.psC)
	return &rat
}

func TestAccessAccount(t *testing.T) {
	accountID := "account_12345"
	ctx := context.Background()
	acc := &auth.Account{
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
	acc := &auth.Account{
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
	acc := &auth.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)
	ctx = devicectx.WithSpruceHeaders(ctx, &device.SpruceHeaders{
		DeviceID: "deviceID",
		Platform: device.IOS,
	})

	rat := new(t)
	defer rat.finish()
	email := "email"
	password := "password"
	rat.aC.Expect(mock.NewExpectation(rat.aC.AuthenticateLogin, &auth.AuthenticateLoginRequest{
		Email:    email,
		Password: password,
		DeviceID: "deviceID",
		Platform: auth.Platform_IOS,
		Duration: auth.TokenDuration_LONG,
	}).WithReturns(&auth.AuthenticateLoginResponse{Account: &auth.Account{ID: accountID}}, nil))

	resp, err := rat.ra.AuthenticateLogin(ctx, email, password, auth.TokenDuration_LONG)
	test.OK(t, err)
	test.Equals(t, &auth.AuthenticateLoginResponse{Account: &auth.Account{ID: accountID}}, resp)
}

func TestAuthenticateLoginWithCode(t *testing.T) {
	accountID := "account_12345"
	ctx := context.Background()
	acc := &auth.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)
	ctx = devicectx.WithSpruceHeaders(ctx, &device.SpruceHeaders{
		DeviceID: "deviceID",
		Platform: device.IOS,
	})

	rat := new(t)
	defer rat.finish()
	token := "token"
	code := "code"
	rat.aC.Expect(mock.NewExpectation(rat.aC.AuthenticateLoginWithCode, &auth.AuthenticateLoginWithCodeRequest{
		Token:    token,
		Code:     code,
		DeviceID: "deviceID",
		Platform: auth.Platform_IOS,
		Duration: auth.TokenDuration_SHORT,
	}).WithReturns(&auth.AuthenticateLoginWithCodeResponse{Account: &auth.Account{ID: accountID}}, nil))

	resp, err := rat.ra.AuthenticateLoginWithCode(ctx, token, code, auth.TokenDuration_SHORT)
	test.OK(t, err)
	test.Equals(t, &auth.AuthenticateLoginWithCodeResponse{Account: &auth.Account{ID: accountID}}, resp)
}

func TestCheckPasswordResetToken(t *testing.T) {
	accountID := "account_12345"
	ctx := context.Background()
	acc := &auth.Account{
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
	acc := &auth.Account{
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
	acc := &auth.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)
	ctx = devicectx.WithSpruceHeaders(ctx, &device.SpruceHeaders{
		DeviceID: "deviceID",
		Platform: device.Android,
	})

	rat := new(t)
	defer rat.finish()
	rat.aC.Expect(mock.NewExpectation(rat.aC.CreateAccount, &auth.CreateAccountRequest{
		FirstName: "name",
		DeviceID:  "deviceID",
		Platform:  auth.Platform_ANDROID,
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
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
			Depth:             0,
		},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
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
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
			Depth:             0,
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
	acc := &auth.Account{
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
	acc := &auth.Account{
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
	acc := &auth.Account{
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
	acc := &auth.Account{
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
	acc := &auth.Account{
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
	acc := &auth.Account{
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
	acc := &auth.Account{
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
	acc := &auth.Account{
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
	acc := &auth.Account{
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
	entityID := "entity_12345"
	ctx := context.Background()
	acc := &auth.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	rat.tC.Expect(mock.NewExpectation(rat.tC.CreateSavedQuery, &threading.CreateSavedQueryRequest{
		EntityID: entityID,
	}).WithReturns(&threading.CreateSavedQueryResponse{}, nil))

	err := rat.ra.CreateSavedQuery(ctx, &threading.CreateSavedQueryRequest{
		EntityID: entityID,
	})
	test.OK(t, err)
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
	acc := &auth.Account{
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
	acc := &auth.Account{
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
	acc := &auth.Account{
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
	acc := &auth.Account{
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
	acc := &auth.Account{
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
	acc := &auth.Account{
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

	resp, err := Entity(ctx, rat.ra, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: entityInfo,
			Depth:             depth,
		},
	})
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
	acc := &auth.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	expectOrgsForEntity(rat, entityID, orgID)
	expectOrgsForEntityForExternalID(rat, accountID, orgID2)

	resp, err := Entity(ctx, rat.ra, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: entityInfo,
			Depth:             depth,
		},
	})
	test.AssertNil(t, resp)
	test.Equals(t, errors.ErrTypeNotAuthorized, errors.Type(err))
}

func TestMarkThreadAsRead(t *testing.T) {
	accountID := "account_12345"
	orgID := "org_12345"
	orgID2 := "org_67890"
	entityID := "entity_12345"
	threadID1 := "t_1"
	threadID2 := "t_2"
	ctx := context.Background()
	acc := &auth.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	rat.dC.Expect(mock.NewExpectation(rat.dC.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ACCOUNT_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_AccountID{
			AccountID: accountID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(&directory.LookupEntitiesResponse{Entities: []*directory.Entity{
		{
			ID: entityID,
			Memberships: []*directory.Entity{
				{
					ID: orgID,
				},
				{
					ID: orgID2,
				},
			},
		},
	}}, nil))

	rat.tC.Expect(mock.NewExpectation(rat.tC.Threads, &threading.ThreadsRequest{
		ThreadIDs:      []string{threadID1, threadID2},
		ViewerEntityID: "",
	}).WithReturns(&threading.ThreadsResponse{
		Threads: []*threading.Thread{
			{
				OrganizationID: orgID,
				ID:             threadID1,
			},
			{
				OrganizationID: orgID2,
				ID:             threadID2,
			},
		},
	}, nil))

	req := &threading.MarkThreadsAsReadRequest{
		ThreadWatermarks: []*threading.MarkThreadsAsReadRequest_ThreadWatermark{
			{
				ThreadID: threadID1,
			},
			{
				ThreadID: threadID2,
			},
		},
		EntityID: entityID,
	}

	rat.tC.Expect(mock.NewExpectation(rat.tC.MarkThreadsAsRead, req))

	_, err := rat.ra.MarkThreadsAsRead(ctx, req)
	test.OK(t, err)
}

func TestMarkThreadAsRead_NotAuthorized(t *testing.T) {
	accountID := "account_12345"
	orgID := "org_12345"
	orgID2 := "org_67890"
	entityID := "entity_12345"
	threadID1 := "t_1"
	threadID2 := "t_2"
	ctx := context.Background()
	acc := &auth.Account{
		ID: accountID,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	rat := new(t)
	defer rat.finish()

	rat.dC.Expect(mock.NewExpectation(rat.dC.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ACCOUNT_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_AccountID{
			AccountID: accountID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(&directory.LookupEntitiesResponse{Entities: []*directory.Entity{
		{
			ID: entityID,
			Memberships: []*directory.Entity{
				{
					ID: orgID,
				},
			},
		},
	}}, nil))

	// second thread part of a different organization
	rat.tC.Expect(mock.NewExpectation(rat.tC.Threads, &threading.ThreadsRequest{
		ThreadIDs:      []string{threadID1, threadID2},
		ViewerEntityID: "",
	}).WithReturns(&threading.ThreadsResponse{
		Threads: []*threading.Thread{
			{
				OrganizationID: orgID,
				ID:             threadID1,
			},
			{
				OrganizationID: orgID2,
				ID:             threadID2,
			},
		},
	}, nil))

	req := &threading.MarkThreadsAsReadRequest{
		ThreadWatermarks: []*threading.MarkThreadsAsReadRequest_ThreadWatermark{
			{
				ThreadID: threadID1,
			},
			{
				ThreadID: threadID2,
			},
		},
		EntityID: entityID,
	}

	_, err := rat.ra.MarkThreadsAsRead(ctx, req)
	test.Equals(t, errors.ErrTypeNotAuthorized, errors.Type(err))
}
