package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	segment "github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/analytics"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/textutil"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
	"google.golang.org/grpc"
)

const (
	createPatientAccountErrorCodeAccountExists                        = "ACCOUNT_EXISTS"
	createPatientAccountErrorCodeInvalidEmail                         = "INVALID_EMAIL"
	createPatientAccountErrorCodeInvalidFirstName                     = "INVALID_FIRST_NAME"
	createPatientAccountErrorCodeInvalidLastName                      = "INVALID_LAST_NAME"
	createPatientAccountErrorCodeInvalidOrganizationName              = "INVALID_ORGANIZATION_NAME"
	createPatientAccountErrorCodeInvalidPassword                      = "INVALID_PASSWORD"
	createPatientAccountErrorCodeInvalidPhoneNumber                   = "INVALID_PHONE_NUMBER"
	createPatientAccountErrorCodeInvalidDOB                           = "INVALID_DOB"
	createPatientAccountErrorCodeInviteRequired                       = "INVITE_REQUIRED"
	createPatientAccountErrorCodeInviteEmailMismatch                  = "INVITE_EMAIL_MISMATCH"
	createPatientAccountErrorCodeInvitePhoneMismatch                  = "INVITE_PHONE_MISMATCH"
	createPatientAccountErrorCodeEmailNotVerified                     = "EMAIL_NOT_VERIFIED"
	createPatientAccountErrorCodePhoneNumberNotVerified               = "PHONE_NUMBER_NOT_VERIFIED"
	createPatientAccountErrorCodeEmailVerificationTokenRequired       = "EMAIL_VERIFICATION_TOKEN_REQUIRED"
	createPatientAccountErrorCodePhoneNumberVerificationTokenRequired = "PHONE_NUMBER_VERIFICATION_TOKEN_REQUIRED"
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
		createPatientAccountErrorCodePhoneNumberNotVerified: &graphql.EnumValueConfig{
			Value:       createPatientAccountErrorCodePhoneNumberNotVerified,
			Description: "The phone number associated with this account creation has not been verified.",
		},
		createPatientAccountErrorCodeEmailVerificationTokenRequired: &graphql.EnumValueConfig{
			Value:       createPatientAccountErrorCodeEmailVerificationTokenRequired,
			Description: "The email verification token is required for this type of invite.",
		},
		createPatientAccountErrorCodePhoneNumberVerificationTokenRequired: &graphql.EnumValueConfig{
			Value:       createPatientAccountErrorCodePhoneNumberVerificationTokenRequired,
			Description: "The phone number verification token is required for this type of invite.",
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

type dateInput struct {
	Month int `gql:"month"`
	Day   int `gql:"day"`
	Year  int `gql:"year"`
}

// dateInputType represents a Date of Birth input pattern
var dateInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "DateInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"month": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Int)},
		"day":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Int)},
		"year":  &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Int)},
	},
})

type createPatientAccountInput struct {
	ClientMutationID       string     `gql:"clientMutationId"`
	UUID                   string     `gql:"uuid"`
	Email                  string     `gql:"email,nonempty"`
	PhoneNumber            string     `gql:"phoneNumber"`
	Password               string     `gql:"password,nonempty"`
	FirstName              string     `gql:"firstName,nonempty"`
	LastName               string     `gql:"lastName,nonempty"`
	DOB                    *dateInput `gql:"dob,nonempty"`
	Gender                 string     `gql:"gender,nonempty"`
	EmailVerificationToken string     `gql:"emailVerificationToken"`
	PhoneVerificationToken string     `gql:"phoneVerificationToken"`
	Duration               string     `gql:"duration"`
	AccountInviteClientID  string     `gql:"accountInviteClientID"`
}

var createPatientAccountInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "CreatePatientAccountInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId":       newClientMutationIDInputField(),
		"uuid":                   newUUIDInputField(),
		"email":                  &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"phoneNumber":            &graphql.InputObjectFieldConfig{Type: graphql.String},
		"password":               &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"firstName":              &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"lastName":               &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"dob":                    &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(dateInputType)},
		"gender":                 &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(genderEnumType)},
		"emailVerificationToken": &graphql.InputObjectFieldConfig{Type: graphql.String},
		"phoneVerificationToken": &graphql.InputObjectFieldConfig{Type: graphql.String},
		"duration":               &graphql.InputObjectFieldConfig{Type: tokenDurationEnum},
		"accountInviteClientID":  accountInviteClientInputType,
	},
})

var createPatientAccountOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "CreatePatientAccountPayload",
	Fields: graphql.Fields{
		"clientMutationId":    newClientMutationIDOutputField(),
		"success":             &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":           &graphql.Field{Type: createPatientAccountErrorCodeEnum},
		"errorMessage":        &graphql.Field{Type: graphql.String},
		"token":               &graphql.Field{Type: graphql.String},
		"account":             &graphql.Field{Type: accountInterfaceType},
		"clientEncryptionKey": &graphql.Field{Type: graphql.String},
		"intercomToken":       intercomTokenField,
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
	sh := devicectx.SpruceHeaders(ctx)

	var in createPatientAccountInput
	if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
		}
		return nil, errors.InternalError(ctx, err)
	}

	inv, atts, err := svc.inviteAndAttributionInfo(ctx, in.AccountInviteClientID)
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	inviteValid := inv != nil

	if inv != nil {
		switch inv.Invite.(type) {
		case *invite.LookupInviteResponse_Patient, *invite.LookupInviteResponse_Organization:
			inviteValid = true
		default:
			inviteValid = false
		}
	}

	if !inviteValid {
		return &createPatientAccountOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          false,
			ErrorCode:        createPatientAccountErrorCodeInviteRequired,
			ErrorMessage:     "An invite from a provider is required to create an account with this device.",
		}, nil

	}

	req := &auth.CreateAccountRequest{
		Email:    in.Email,
		Password: in.Password,
		Type:     auth.AccountType_PATIENT,
		DeviceID: sh.DeviceID,
		Platform: authPlatform(sh.Platform),
	}
	if in.Duration == "" {
		in.Duration = auth.TokenDuration_SHORT.String()
	}
	req.Duration = auth.TokenDuration(auth.TokenDuration_value[in.Duration])
	if req.Password == "" {
		return &createPatientAccountOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          false,
			ErrorCode:        createPatientAccountErrorCodeInvalidPassword,
			ErrorMessage:     "Password cannot be empty",
		}, nil
	}
	var phoneNumber string
	if _, ok := inv.Invite.(*invite.LookupInviteResponse_Patient); ok {
		phoneNumber, err = contactForParkedEntity(ctx, ram, inv.GetPatient().Patient.ParkedEntityID, directory.ContactType_PHONE)
		if err != nil {
			return nil, errors.InternalError(ctx, fmt.Errorf("Encountered error whil getting parked phone number for account creation: %s", err))
		}
	} else {
		phoneNumber = in.PhoneNumber
	}

	pn, err := phone.ParseNumber(phoneNumber)
	if err != nil {
		return &createPatientAccountOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          false,
			ErrorCode:        createPatientAccountErrorCodeInvalidPhoneNumber,
			ErrorMessage:     "Please enter a valid phone number.",
		}, nil
	}
	req.PhoneNumber = pn.String()

	if _, ok := inv.Invite.(*invite.LookupInviteResponse_Organization); ok {
		// Assert that the phone number was verified
		if in.PhoneVerificationToken == "" {
			return &createPatientAccountOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        createPatientAccountErrorCodePhoneNumberVerificationTokenRequired,
				ErrorMessage:     "Phone number verification token required.",
			}, nil
		}
		if _, err := ram.VerifiedValue(ctx, in.PhoneVerificationToken); err != nil {
			if grpc.Code(err) == auth.ValueNotYetVerified {
				return &createPatientAccountOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          false,
					ErrorCode:        createPatientAccountErrorCodePhoneNumberNotVerified,
					ErrorMessage:     "The phone number associated with this account creation has not been verified.",
				}, nil
			}
			return nil, errors.InternalError(ctx, fmt.Errorf("Encountered error while checking if phone number has been verified: %s", err))
		}
	}

	req.Email = strings.TrimSpace(req.Email)
	if !validate.Email(req.Email) {
		return &createPatientAccountOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          false,
			ErrorCode:        createPatientAccountErrorCodeInvalidEmail,
			ErrorMessage:     "Please enter a valid email address.",
		}, nil
	}

	entityInfo, err := entityInfoFromInput(input)
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	req.FirstName = strings.TrimSpace(entityInfo.FirstName)
	req.LastName = strings.TrimSpace(entityInfo.LastName)
	if req.FirstName == "" || !textutil.IsValidPlane0Unicode(req.FirstName) {
		return &createPatientAccountOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          false,
			ErrorCode:        createPatientAccountErrorCodeInvalidFirstName,
			ErrorMessage:     "Please enter a valid first name.",
		}, nil
	}
	if req.LastName == "" || !textutil.IsValidPlane0Unicode(req.LastName) {
		return &createPatientAccountOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          false,
			ErrorCode:        createPatientAccountErrorCodeInvalidLastName,
			ErrorMessage:     "Please enter a valid last name.",
		}, nil
	}

	// Only do email verification if it was a direct patient invite
	var accountEntityID string
	var orgID string
	switch inviteData := inv.Invite.(type) {
	case *invite.LookupInviteResponse_Patient:
		patientInvite := inviteData.Patient

		// Assert that the email was verified (if verification requirement is unknown then treat as though
		// email required for backwards compatibility)
		if patientInvite.InviteVerificationRequirement == invite.VERIFICATION_REQUIREMENT_EMAIL ||
			patientInvite.InviteVerificationRequirement == invite.VERIFICATION_REQUIREMENT_UNKNOWN {
			if in.EmailVerificationToken == "" {
				return &createPatientAccountOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          false,
					ErrorCode:        createPatientAccountErrorCodeEmailVerificationTokenRequired,
					ErrorMessage:     "Email verification token required.",
				}, nil
			}

			if _, err := ram.VerifiedValue(ctx, in.EmailVerificationToken); err != nil {
				if grpc.Code(err) == auth.ValueNotYetVerified {
					return &createPatientAccountOutput{
						ClientMutationID: in.ClientMutationID,
						Success:          false,
						ErrorCode:        createPatientAccountErrorCodeEmailNotVerified,
						ErrorMessage:     "The email associated with this account creation has not been verified.",
					}, nil
				}
				return nil, errors.InternalError(ctx, fmt.Errorf("Encountered error while checking if email has been verified: %s", err))
			}
		}

		if patientInvite.InviteVerificationRequirement == invite.VERIFICATION_REQUIREMENT_PHONE_MATCH ||
			patientInvite.InviteVerificationRequirement == invite.VERIFICATION_REQUIREMENT_PHONE {

			if in.PhoneVerificationToken == "" {
				return &createPatientAccountOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          false,
					ErrorCode:        createPatientAccountErrorCodePhoneNumberVerificationTokenRequired,
					ErrorMessage:     "Phone verification token required.",
				}, nil
			}

			if _, err := ram.VerifiedValue(ctx, in.PhoneVerificationToken); err != nil {
				if grpc.Code(err) == auth.ValueNotYetVerified {
					return &createPatientAccountOutput{
						ClientMutationID: in.ClientMutationID,
						Success:          false,
						ErrorCode:        createPatientAccountErrorCodePhoneNumberNotVerified,
						ErrorMessage:     "The phone number associated with this account creation has not been verified.",
					}, nil
				}
				return nil, errors.InternalError(ctx, fmt.Errorf("Encountered error while checking if email has been verified: %s", err))
			}
		}

		if patientInvite.Patient.ParkedEntityID == "" {
			return nil, errors.InternalError(ctx, fmt.Errorf("Unable to find parked entity account associated with invite %+v", inv))
		}
		accountEntityID = patientInvite.Patient.ParkedEntityID
		orgID = patientInvite.OrganizationEntityID
	case *invite.LookupInviteResponse_Organization:
	default:
		return nil, errors.InternalError(ctx, fmt.Errorf("Unsupported invite type %s", inv.Type))
	}

	res, err := ram.CreateAccount(ctx, req)
	if err != nil {
		switch grpc.Code(err) {
		case auth.DuplicateEmail:
			return &createPatientAccountOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        createPatientAccountErrorCodeAccountExists,
				ErrorMessage:     "An account already exists with the entered email address.",
			}, nil
		case auth.InvalidEmail:
			return &createPatientAccountOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        createPatientAccountErrorCodeInvalidEmail,
				ErrorMessage:     "Please enter a valid email address.",
			}, nil
		case auth.InvalidPhoneNumber:
			return &createPatientAccountOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        createPatientAccountErrorCodeInvalidPhoneNumber,
				ErrorMessage:     "Please enter a valid phone number.",
			}, nil
		}
		return nil, errors.InternalError(ctx, err)
	}
	gqlctx.InPlaceWithAccount(ctx, res.Account)

	var autoTags []string
	var patientEntity *directory.Entity
	if oinv, ok := inv.Invite.(*invite.LookupInviteResponse_Organization); ok {
		// If this is an org code then there is no parked entity and we need to create the entity and thread
		patientEntity, err = ram.CreateEntity(ctx, &directory.CreateEntityRequest{
			Type: directory.EntityType_PATIENT,
			InitialMembershipEntityID: inv.GetOrganization().OrganizationEntityID,
			Contacts: []*directory.Contact{
				{
					ContactType: directory.ContactType_EMAIL,
					Value:       req.Email,
				},
				{
					ContactType: directory.ContactType_PHONE,
					Value:       req.PhoneNumber,
				},
			},
			EntityInfo: &directory.EntityInfo{
				FirstName: entityInfo.FirstName,
				LastName:  entityInfo.LastName,
			},
			Source: &directory.EntitySource{
				Type: directory.EntitySource_PRACTICE_CODE,
				Data: oinv.Organization.Token,
			},
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		accountEntityID = patientEntity.ID
		orgID = inv.GetOrganization().OrganizationEntityID
		autoTags = inv.GetOrganization().Tags
	}

	// Associate the parked entity with the account
	if err = ram.UnauthorizedCreateExternalIDs(ctx, &directory.CreateExternalIDsRequest{
		EntityID:    accountEntityID,
		ExternalIDs: []string{res.Account.ID},
	}); err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	if _, ok := inv.Invite.(*invite.LookupInviteResponse_Organization); ok {
		// Create a thread with the parked patient in the org
		_, err := ram.CreateEmptyThread(ctx, &threading.CreateEmptyThreadRequest{
			OrganizationID:  inv.GetOrganization().OrganizationEntityID,
			PrimaryEntityID: patientEntity.ID,
			MemberEntityIDs: []string{inv.GetOrganization().OrganizationEntityID, accountEntityID},
			Type:            threading.THREAD_TYPE_SECURE_EXTERNAL,
			Summary:         patientEntity.Info.DisplayName,
			SystemTitle:     patientEntity.Info.DisplayName,
			Origin:          threading.THREAD_ORIGIN_ORGANIZATION_CODE,
			Tags:            autoTags,
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
	}

	patientEntity, err = ram.UpdateEntity(ctx, &directory.UpdateEntityRequest{
		EntityID:         accountEntityID,
		UpdateAccountID:  true,
		EntityInfo:       entityInfo,
		UpdateEntityInfo: true,
		AccountID:        res.Account.ID,
		UpdateContacts:   true,
		Contacts: []*directory.Contact{
			{ContactType: directory.ContactType_EMAIL, Value: req.Email, Verified: false},
			{ContactType: directory.ContactType_PHONE, Value: req.PhoneNumber, Verified: true},
		},
	})
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	// Since the patient may have changed their name, asynchronously update the thread
	// Async contexts are cloned to preserve the permissions of the caller
	asyncCtx := gqlctx.Clone(ctx)
	conc.Go(func() {
		threads, err := ram.ThreadsForMember(asyncCtx, accountEntityID, true)
		if err != nil {
			golog.Errorf("Encountered error when attempting to get threads for parked entity %q to update title: %s", accountEntityID, err)
			return
		}
		for _, th := range threads {
			if _, err := ram.UpdateThread(asyncCtx, &threading.UpdateThreadRequest{
				ActorEntityID: th.OrganizationID,
				ThreadID:      th.ID,
				SystemTitle:   patientEntity.Info.DisplayName,
			}); err != nil {
				golog.Errorf("Encountered error when attempting to update thread title for new patient account: %s", err)
				return
			}
		}
	})

	// Mark the invite as consumed if it's not an org code
	if inv != nil {
		if _, ok := inv.Invite.(*invite.LookupInviteResponse_Organization); !ok {
			conc.GoCtx(gqlctx.Clone(ctx), func(ctx context.Context) {
				if _, err := svc.invite.MarkInviteConsumed(ctx, &invite.MarkInviteConsumedRequest{Token: atts[inviteTokenAttributionKey]}); err != nil {
					golog.Errorf("Error while marking invite with code %q as consumed: %s", atts[inviteTokenAttributionKey], err)
				}
			})
		}
	}

	// Record Analytics
	recordCreatePatientAccountAnalytics(ctx, ram, svc, p, inv, res.Account, orgID, accountEntityID)
	result := p.Info.RootValue.(map[string]interface{})["result"].(*conc.Map)
	result.Set("auth_token", res.Token.Value)
	result.Set("auth_expiration", time.Unix(int64(res.Token.ExpirationEpoch), 0))

	return &createPatientAccountOutput{
		ClientMutationID:    in.ClientMutationID,
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
	headers := devicectx.SpruceHeaders(ctx)
	var platform string
	if headers != nil {
		platform = headers.Platform.String()
		golog.Debugf("Patient Account created. ID = %s Device = %s", account.ID, headers.DeviceID)
	}

	analytics.SegmentIdentify(&segment.Identify{
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
	analytics.SegmentTrack(&segment.Track{
		Event:      "signedup",
		UserId:     account.ID,
		Properties: props,
	})
}
