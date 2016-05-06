package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	createPatientAccountErrorCodeAccountExists           = "ACCOUNT_EXISTS"
	createPatientAccountErrorCodeInvalidEmail            = "INVALID_EMAIL"
	createPatientAccountErrorCodeInvalidFirstName        = "INVALID_FIRST_NAME"
	createPatientAccountErrorCodeInvalidLastName         = "INVALID_LAST_NAME"
	createPatientAccountErrorCodeInvalidOrganizationName = "INVALID_ORGANIZATION_NAME"
	createPatientAccountErrorCodeInvalidPassword         = "INVALID_PASSWORD"
	createPatientAccountErrorCodeInvalidPhoneNumber      = "INVALID_PHONE_NUMBER"
	createPatientAccountErrorCodeInvalidDOB              = "INVALID_DOB"
	createPatientAccountErrorCodeInviteRequired          = "INVITE_REQUIRED"
	createPatientAccountErrorCodeInviteEmailMismatch     = "INVITE_EMAIL_MISMATCH"
	createPatientAccountErrorCodeInvitePhoneMismatch     = "INVITE_PHONE_MISMATCH"
	createPatientAccountErrorCodeEmailNotVerified        = "EMAIL_NOT_VERIFIED"
)

var createPatientAccountErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "CreatePatientAccountErrorCode",
	Values: graphql.EnumValueConfigMap{
		createPatientAccountErrorCodeInvalidEmail: &graphql.EnumValueConfig{
			Value:       createPatientAccountErrorCodeInvalidEmail,
			Description: "The provided email is invalid",
		},
		createPatientAccountErrorCodeInvalidPassword: &graphql.EnumValueConfig{
			Value:       createPatientAccountErrorCodeInvalidPassword,
			Description: "The provided password is invalid",
		},
		createPatientAccountErrorCodeInvalidPhoneNumber: &graphql.EnumValueConfig{
			Value:       createPatientAccountErrorCodeInvalidPhoneNumber,
			Description: "The provided phone number is invalid",
		},
		createPatientAccountErrorCodeAccountExists: &graphql.EnumValueConfig{
			Value:       createPatientAccountErrorCodeAccountExists,
			Description: "An account exists with the provided email address",
		},
		createPatientAccountErrorCodeInvalidOrganizationName: &graphql.EnumValueConfig{
			Value:       createPatientAccountErrorCodeInvalidOrganizationName,
			Description: "The provided organization name is invalid",
		},
		createPatientAccountErrorCodeInvalidFirstName: &graphql.EnumValueConfig{
			Value:       createPatientAccountErrorCodeInvalidFirstName,
			Description: "The provided first name is invalid",
		},
		createPatientAccountErrorCodeInvalidLastName: &graphql.EnumValueConfig{
			Value:       createPatientAccountErrorCodeInvalidLastName,
			Description: "The provided last name is invalid",
		},
		createAccountErrorCodeInvalidDOB: &graphql.EnumValueConfig{
			Value:       createPatientAccountErrorCodeInvalidDOB,
			Description: "The provided date of birth is invalid",
		},
		createPatientAccountErrorCodeInviteRequired: &graphql.EnumValueConfig{
			Value:       createPatientAccountErrorCodeInviteRequired,
			Description: "An invite is required to create an account with this device",
		},
		createPatientAccountErrorCodeInviteEmailMismatch: &graphql.EnumValueConfig{
			Value:       createPatientAccountErrorCodeInviteEmailMismatch,
			Description: "The provided email doesn't match the invite",
		},
		createPatientAccountErrorCodeInvitePhoneMismatch: &graphql.EnumValueConfig{
			Value:       createPatientAccountErrorCodeInvitePhoneMismatch,
			Description: "The provided phone number doesn't match the invite",
		},
		createPatientAccountErrorCodeEmailNotVerified: &graphql.EnumValueConfig{
			Value:       createPatientAccountErrorCodeEmailNotVerified,
			Description: "The email associated with this account creation has not been verified.",
		},
	},
})

type createPatientAccountOutput struct {
	ClientMutationID    string         `json:"clientMutationId,omitempty"`
	Success             bool           `json:"success"`
	ErrorCode           string         `json:"errorCode,omitempty"`
	ErrorMessage        string         `json:"errorMessage,omitempty"`
	Token               string         `json:"token,omitempty"`
	Account             models.Account `json:"account,omitempty"`
	ClientEncryptionKey string         `json:"clientEncryptionKey,omitempty"`
}

const (
	genderMale    = "MALE"
	genderFemale  = "FEMALE"
	genderOther   = "OTHER"
	genderUnknown = "UNKNOWN"
)

var genderEnumType = graphql.NewEnum(graphql.EnumConfig{
	Name:        "Gender",
	Description: "The gender of a thing",
	Values: graphql.EnumValueConfigMap{
		genderUnknown: &graphql.EnumValueConfig{
			Value: genderUnknown,
		},
		genderMale: &graphql.EnumValueConfig{
			Value: genderMale,
		},
		genderFemale: &graphql.EnumValueConfig{
			Value: genderFemale,
		},
		genderOther: &graphql.EnumValueConfig{
			Value: genderOther,
		},
	},
})

// dateInputType represents a Date of Birth input pattern
var dateInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "DateInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"month": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Int)},
		"day":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Int)},
		"year":  &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Int)},
	},
})

var createPatientAccountInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "CreatePatientAccountInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId":       newClientMutationIDInputField(),
		"uuid":                   newUUIDInputField(),
		"email":                  &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"password":               &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"firstName":              &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"lastName":               &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"dob":                    &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(dateInputType)},
		"gender":                 &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(genderEnumType)},
		"emailVerificationToken": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
	},
})

var createPatientAccountOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "CreatePatientAccountPayload",
	Fields: graphql.Fields{
		"clientMutationId":    newClientmutationIDOutputField(),
		"success":             &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":           &graphql.Field{Type: createPatientAccountErrorCodeEnum},
		"errorMessage":        &graphql.Field{Type: graphql.String},
		"token":               &graphql.Field{Type: graphql.String},
		"account":             &graphql.Field{Type: accountInterfaceType},
		"clientEncryptionKey": &graphql.Field{Type: graphql.String},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*createPatientAccountOutput)
		return ok
	},
})

var createPatientAccountMutation = &graphql.Field{
	Type: graphql.NewNonNull(createPatientAccountOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(createPatientAccountInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		return createPatientAccount(p)
	},
}

func createPatientAccount(p graphql.ResolveParams) (*createPatientAccountOutput, error) {
	svc := serviceFromParams(p)
	ram := raccess.ResourceAccess(p)
	ctx := p.Context
	input := p.Args["input"].(map[string]interface{})
	mutationID, _ := input["clientMutationId"].(string)

	inv, atts, err := svc.inviteAndAttributionInfo(ctx)
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	if inv == nil || inv.Type != invite.LookupInviteResponse_PATIENT {
		return &createPatientAccountOutput{
			ClientMutationID: mutationID,
			Success:          false,
			ErrorCode:        createPatientAccountErrorCodeInviteRequired,
			ErrorMessage:     "An invite from a provider is required to create an account with this device.",
		}, nil
	}

	req := &auth.CreateAccountRequest{
		Email:    input["email"].(string),
		Password: input["password"].(string),
		Type:     auth.AccountType_PATIENT,
	}
	if req.Password == "" {
		return &createPatientAccountOutput{
			ClientMutationID: mutationID,
			Success:          false,
			ErrorCode:        createPatientAccountErrorCodeInvalidPassword,
			ErrorMessage:     "Password cannot be empty",
		}, nil
	}
	invPhone, err := contactForParkedEntity(ctx, ram, inv.GetPatient().Patient.ParkedEntityID, directory.ContactType_PHONE)
	if err != nil {
		return nil, errors.InternalError(ctx, fmt.Errorf("Encountered error whil getting parked phone number for account creation: %s", err))
	}
	pn, err := phone.ParseNumber(invPhone)
	if err != nil {
		return &createPatientAccountOutput{
			ClientMutationID: mutationID,
			Success:          false,
			ErrorCode:        createPatientAccountErrorCodeInvalidPhoneNumber,
			ErrorMessage:     "Please enter a valid phone number.",
		}, nil
	}
	req.PhoneNumber = pn.String()
	req.Email = strings.TrimSpace(req.Email)
	if !validate.Email(req.Email) {
		return &createPatientAccountOutput{
			ClientMutationID: mutationID,
			Success:          false,
			ErrorCode:        createPatientAccountErrorCodeInvalidEmail,
			ErrorMessage:     "Please enter a valid email address.",
		}, nil
	}
	// Assert that the email was verified
	if _, err := ram.VerifiedValue(ctx, input["emailVerificationToken"].(string)); err != nil {
		if grpc.Code(err) == auth.ValueNotYetVerified {
			return &createPatientAccountOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        createPatientAccountErrorCodeEmailNotVerified,
				ErrorMessage:     "The email associated with this account creation has not been verified.",
			}, nil
		}
		return nil, errors.InternalError(ctx, fmt.Errorf("Encountered error while checking if email has been verified: %s", err))
	}
	if inv.GetPatient().Patient.ParkedEntityID == "" {
		return nil, errors.InternalError(ctx, fmt.Errorf("Unable to find parked entity account associated with invite %+v", inv))
	}

	entityInfo, err := entityInfoFromInput(input)
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	req.FirstName = strings.TrimSpace(entityInfo.FirstName)
	req.LastName = strings.TrimSpace(entityInfo.LastName)
	if req.FirstName == "" || !isValidPlane0Unicode(req.FirstName) {
		return &createPatientAccountOutput{
			ClientMutationID: mutationID,
			Success:          false,
			ErrorCode:        createPatientAccountErrorCodeInvalidFirstName,
			ErrorMessage:     "Please enter a valid first name.",
		}, nil
	}
	if req.LastName == "" || !isValidPlane0Unicode(req.LastName) {
		return &createPatientAccountOutput{
			ClientMutationID: mutationID,
			Success:          false,
			ErrorCode:        createPatientAccountErrorCodeInvalidLastName,
			ErrorMessage:     "Please enter a valid last name.",
		}, nil
	}
	res, err := ram.CreateAccount(ctx, req)
	if err != nil {
		switch grpc.Code(err) {
		case auth.DuplicateEmail:
			return &createPatientAccountOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        createPatientAccountErrorCodeAccountExists,
				ErrorMessage:     "An account already exists with the entered email address.",
			}, nil
		case auth.InvalidEmail:
			return &createPatientAccountOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        createPatientAccountErrorCodeInvalidEmail,
				ErrorMessage:     "Please enter a valid email address.",
			}, nil
		case auth.InvalidPhoneNumber:
			return &createPatientAccountOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        createPatientAccountErrorCodeInvalidPhoneNumber,
				ErrorMessage:     "Please enter a valid phone number.",
			}, nil
		}
		return nil, errors.InternalError(ctx, err)
	}
	gqlctx.InPlaceWithAccount(ctx, res.Account)

	// Associate the parked entity with the account
	if err = ram.UnauthorizedCreateExternalIDs(ctx, &directory.CreateExternalIDsRequest{
		EntityID:    inv.GetPatient().Patient.ParkedEntityID,
		ExternalIDs: []string{res.Account.ID},
	}); err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	// Update our parked entity
	patientEntity, err := ram.UpdateEntity(ctx, &directory.UpdateEntityRequest{
		EntityID:         inv.GetPatient().Patient.ParkedEntityID,
		UpdateEntityInfo: true,
		EntityInfo:       entityInfo,
		UpdateAccountID:  true,
		AccountID:        res.Account.ID,
		UpdateContacts:   true,
		Contacts: []*directory.Contact{
			{ContactType: directory.ContactType_EMAIL, Value: req.Email},
		},
	})
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	// Since the patient may have changed their name, asynchronously update the thread
	// Async contexts are cloned to preserve the permissions of the caller
	asyncCtx := gqlctx.Clone(ctx)
	conc.Go(func() {
		threads, err := ram.ThreadsForMember(asyncCtx, inv.GetPatient().Patient.ParkedEntityID, true)
		if err != nil {
			golog.Errorf("Encountered error when attempting to get threads for parked entity %q to update title: %s", inv.GetPatient().Patient.ParkedEntityID, err)
			return
		}
		for _, th := range threads {
			if _, err := ram.UpdateThread(asyncCtx, &threading.UpdateThreadRequest{
				ThreadID:    th.ID,
				SystemTitle: patientEntity.Info.DisplayName,
			}); err != nil {
				golog.Errorf("Encountered error when attempting to update thread title for new patient account: %s", err)
				return
			}
		}
	})
	// Mark the invite as consumed
	if inv != nil {
		conc.Go(func() {
			if _, err := svc.invite.MarkInviteConsumed(ctx, &invite.MarkInviteConsumedRequest{Token: atts[inviteTokenAttributionKey]}); err != nil {
				golog.Errorf("Error while marking invite with code %q as consumed: %s", atts[inviteTokenAttributionKey], err)
			}
		})
	}

	// Record Analytics
	recordCreatePatientAccountAnalytics(ctx, ram, svc, p, inv, res.Account, inv.GetPatient().OrganizationEntityID, inv.GetPatient().Patient.ParkedEntityID)
	result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
	result.Set("auth_token", res.Token.Value)
	result.Set("auth_expiration", time.Unix(int64(res.Token.ExpirationEpoch), 0))

	return &createPatientAccountOutput{
		ClientMutationID:    mutationID,
		Success:             true,
		Token:               res.Token.Value,
		Account:             transformAccountToResponse(res.Account),
		ClientEncryptionKey: res.Token.ClientEncryptionKey,
	}, nil
}

func recordCreatePatientAccountAnalytics(
	ctx context.Context,
	ram raccess.ResourceAccessor,
	svc *service,
	p graphql.ResolveParams,
	inv *invite.LookupInviteResponse,
	account *auth.Account,
	orgID, accEntityID string) {
	// Record analytics
	headers := gqlctx.SpruceHeaders(ctx)
	var platform string
	if headers != nil {
		platform = headers.Platform.String()
		golog.Debugf("Patient Account created. ID = %s Device = %s", account.ID, headers.DeviceID)
	}
	conc.Go(func() {
		svc.segmentio.Identify(&analytics.Identify{
			UserId: account.ID,
			Traits: map[string]interface{}{
				"platform":  platform,
				"createdAt": time.Now().Unix(),
				"type":      "patient",
			},
			Context: map[string]interface{}{
				"ip":        remoteAddrFromParams(p),
				"userAgent": userAgentFromParams(p),
			},
		})
		props := map[string]interface{}{
			"entity_id":       accEntityID,
			"organization_id": orgID,
		}
		if inv != nil {
			props["invite"] = inv.Type.String()
		}
		svc.segmentio.Track(&analytics.Track{
			Event:      "signedup",
			UserId:     account.ID,
			Properties: props,
		})
	})
}
