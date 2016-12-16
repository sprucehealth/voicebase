package main

import (
	"fmt"
	"runtime/debug"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
	"google.golang.org/grpc"
)

// verifyPhoneNumber

const (
	verifyPhoneNumberErrorCodeInvitePhoneMismatch = "INVITE_PHONE_MISMATCH"
	verifyPhoneNumberErrorCodeInvalidPhone        = "INVALID_PHONE_NUMBER"
)

var verifyPhoneNumberErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name:        "VerifyPhoneNumberErrorCode",
	Description: "Result of verifyPhoneNumber mutation",
	Values: graphql.EnumValueConfigMap{
		verifyPhoneNumberErrorCodeInvitePhoneMismatch: &graphql.EnumValueConfig{
			Value:       verifyPhoneNumberErrorCodeInvitePhoneMismatch,
			Description: "Phone number from invite does not match",
		},
		verifyPhoneNumberErrorCodeInvalidPhone: &graphql.EnumValueConfig{
			Value:       verifyPhoneNumberErrorCodeInvalidPhone,
			Description: "Invalid phone number",
		},
	},
})

type verifyPhoneNumberOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
	Token            string `json:"token"`
	Message          string `json:"message"`
}

type verifyPhoneNumberInput struct {
	ClientMutationID      string `gql:"clientMutationId"`
	UUID                  string `gql:"uuid"`
	PhoneNumber           string `gql:"phoneNumber"`
	AccountInviteClientID string `gql:"accountInviteClientID"`
}

var verifyPhoneNumberInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "VerifyPhoneNumberInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             newUUIDInputField(),
		"phoneNumber": &graphql.InputObjectFieldConfig{
			Type:        graphql.NewNonNull(graphql.String),
			Description: "Specify the phone number to send a verification code to.",
		},
		"accountInviteClientID": accountInviteClientInputType,
	},
})

var verifyPhoneNumberOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "VerifyPhoneNumberPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientMutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: verifyPhoneNumberErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
		"token":            &graphql.Field{Type: graphql.String},
		"message":          &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*verifyPhoneNumberOutput)
		return ok
	},
})

var verifyPhoneNumberMutation = &graphql.Field{
	Type: graphql.NewNonNull(verifyPhoneNumberOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(verifyPhoneNumberInputType)},
	},
	Resolve: makeVerifyPhoneNumberResolve(false),
}

var verifyPhoneNumberForAccountCreationMutation = &graphql.Field{
	Type: graphql.NewNonNull(verifyPhoneNumberOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(verifyPhoneNumberInputType)},
	},
	Resolve: makeVerifyPhoneNumberResolve(true),
}

func makeVerifyPhoneNumberResolve(forAccountCreation bool) func(p graphql.ResolveParams) (interface{}, error) {
	return func(p graphql.ResolveParams) (interface{}, error) {
		defer func() {
			if recover() != nil {
				debug.PrintStack()
			}
		}()

		svc := serviceFromParams(p)
		ram := raccess.ResourceAccess(p)
		ctx := p.Context

		input := p.Args["input"].(map[string]interface{})
		var in verifyPhoneNumberInput
		if err := gqldecode.Decode(input, &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(ctx, err)
		}

		pn, err := phone.ParseNumber(in.PhoneNumber)
		if err != nil {
			return &verifyPhoneNumberOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        verifyPhoneNumberErrorCodeInvalidPhone,
				ErrorMessage:     "Please use a valid U.S. phone number",
			}, nil
		}

		// ensure that the phone number is not a provisioned phone number
		entities, err := ram.EntitiesByContact(ctx, &directory.LookupEntitiesByContactRequest{
			ContactValue: pn.String(),
			Statuses:     []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RequestedInformation: &directory.RequestedInformation{
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
			},
		})
		if err != nil {
			golog.Errorf("Unable to lookup entity by contact: %s", err.Error())
		}

		for _, ent := range entities {
			for _, c := range ent.Contacts {
				if c.Provisioned && c.Value == pn.String() {
					return &verifyPhoneNumberOutput{
						ClientMutationID: in.ClientMutationID,
						Success:          false,
						ErrorCode:        verifyPhoneNumberErrorCodeInvalidPhone,
						ErrorMessage:     "Please use a non-Spruce number to create an account with.",
					}, nil
				}
			}
		}

		// Provided phone number must match what was provided during invite if here through invite
		if forAccountCreation {
			inv, _, err := svc.inviteAndAttributionInfo(ctx, in.AccountInviteClientID)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			if inv != nil {
				switch inv.Invite.(type) {
				case *invite.LookupInviteResponse_Colleague:
					col := inv.GetColleague().Colleague
					if col.PhoneNumber != pn.String() {
						return &verifyPhoneNumberOutput{
							ClientMutationID: in.ClientMutationID,
							Success:          false,
							ErrorCode:        verifyPhoneNumberErrorCodeInvitePhoneMismatch,
							ErrorMessage:     "The phone number must match the one that was in your invite.",
						}, nil
					}
				case *invite.LookupInviteResponse_Patient:

					ent, err := raccess.UnauthorizedEntity(ctx, ram, &directory.LookupEntitiesRequest{
						Key: &directory.LookupEntitiesRequest_EntityID{
							EntityID: inv.GetPatient().Patient.ParkedEntityID,
						},
						RequestedInformation: &directory.RequestedInformation{
							EntityInformation: []directory.EntityInformation{
								directory.EntityInformation_CONTACTS,
							},
						},
					})
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}

					contacts := directory.FilterContacts(ent, directory.ContactType_PHONE)

					if inv.GetPatient().InviteVerificationRequirement == invite.VERIFICATION_REQUIREMENT_PHONE_MATCH {
						if len(contacts) == 0 {
							return nil, errors.InternalError(ctx, errors.Errorf("no phone number for patient %s", ent.ID))
						}

						if contacts[0].Value != pn.String() {
							return &verifyPhoneNumberOutput{
								ClientMutationID: in.ClientMutationID,
								Success:          false,
								ErrorCode:        verifyPhoneNumberErrorCodeInvitePhoneMismatch,
								ErrorMessage:     "The phone number must match the one that was in your invite.",
							}, nil
						}
					}
				case *invite.LookupInviteResponse_Organization:
					// do nothing
				default:
					golog.Errorf("Unknown invite type %s", inv.Type.String())
				}
			}
		}

		token, err := createAndSendSMSVerificationCode(ctx, ram, svc.serviceNumber, auth.VerificationCodeType_PHONE, pn.String(), pn)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		nicePhone, err := pn.Format(phone.Pretty)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		return &verifyPhoneNumberOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          true,
			Token:            token,
			Message:          fmt.Sprintf("A verification code has been sent to %s", nicePhone),
		}, nil
	}
}

// checkVerificationCode

const (
	checkVerificationCodeErrorCodeFailure = "VERIFICATION_FAILED"
	checkVerificationCodeErrorCodeExpired = "CODE_EXPIRED"
)

type checkVerificationCodeOutput struct {
	ClientMutationID   string                     `json:"clientMutationId,omitempty"`
	Success            bool                       `json:"success"`
	ErrorCode          string                     `json:"errorCode,omitempty"`
	ErrorMessage       string                     `json:"errorMessage,omitempty"`
	Account            models.Account             `json:"account"`
	VerifiedEntityInfo *models.VerifiedEntityInfo `json:"verifiedEntityInfo"`
}

var checkVerificationCodeErrorCodeEnum = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "CheckVerificationCodeErrorCode",
		Description: "Result of checkVerificationCode mutation",
		Values: graphql.EnumValueConfigMap{
			checkVerificationCodeErrorCodeFailure: &graphql.EnumValueConfig{
				Value:       checkVerificationCodeErrorCodeFailure,
				Description: "Code expired",
			},
			checkVerificationCodeErrorCodeExpired: &graphql.EnumValueConfig{
				Value:       checkVerificationCodeErrorCodeExpired,
				Description: "Code verification failed",
			},
		},
	},
)

var checkVerificationCodeInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "CheckVerificationCodeInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId":      newClientMutationIDInputField(),
			"uuid":                  newUUIDInputField(),
			"token":                 &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"code":                  &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"accountInviteClientID": accountInviteClientInputType,
		},
	},
)

type CheckVerificationCodeInput struct {
	ClientMutationID      string `gql:"clientMutationId"`
	UUID                  string `gql:"uuid"`
	Token                 string `gql:"token"`
	Code                  string `gql:"code"`
	AccountInviteClientID string `gql:"accountInviteClientID"`
}

var verifiedEntityInfo = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "VerifiedEntityInfo",
		Fields: graphql.Fields{
			"firstName": &graphql.Field{Type: graphql.String},
			"lastName":  &graphql.Field{Type: graphql.String},
			"email":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

var checkVerificationCodeOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CheckVerificationCodePayload",
		Fields: graphql.Fields{
			"clientMutationId":   newClientMutationIDOutputField(),
			"success":            &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":          &graphql.Field{Type: checkVerificationCodeErrorCodeEnum},
			"errorMessage":       &graphql.Field{Type: graphql.String},
			"account":            &graphql.Field{Type: accountInterfaceType},
			"verifiedEntityInfo": &graphql.Field{Type: verifiedEntityInfo},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*checkVerificationCodeOutput)
			return ok
		},
	},
)

var checkVerificationCodeMutation = &graphql.Field{
	Type: graphql.NewNonNull(checkVerificationCodeOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(checkVerificationCodeInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ram := raccess.ResourceAccess(p)
		ctx := p.Context

		input := p.Args["input"].(map[string]interface{})
		var in CheckVerificationCodeInput
		if err := gqldecode.Decode(input, &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(ctx, err)
		}

		golog.Debugf("Checking token %s against code %s", in.Token, in.Code)
		resp, err := ram.CheckVerificationCode(ctx, in.Token, in.Code)
		if grpc.Code(err) == auth.BadVerificationCode {
			return &checkVerificationCodeOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        checkVerificationCodeErrorCodeFailure,
				ErrorMessage:     "The entered code is incorrect.",
			}, nil
		} else if grpc.Code(err) == auth.VerificationCodeExpired {
			return &checkVerificationCodeOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        checkVerificationCodeErrorCodeExpired,
				ErrorMessage:     "The entered code has expired. Please request a new code.",
			}, nil
		} else if err != nil {
			golog.Errorf(err.Error())
			return nil, errors.New("Failed to check verification code")
		}

		// If the value we just verified was an email address and this device is associated with a patient invite
		// then send back the parked entity info
		var verifiedEntityInfo *models.VerifiedEntityInfo
		if validate.Email(resp.Value) {
			inv, _, err := svc.inviteAndAttributionInfo(ctx, in.AccountInviteClientID)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			if inv != nil {
				if _, ok := inv.Invite.(*invite.LookupInviteResponse_Patient); ok {
					entities, err := ram.Entities(ctx, &directory.LookupEntitiesRequest{
						Key: &directory.LookupEntitiesRequest_EntityID{
							EntityID: inv.GetPatient().Patient.ParkedEntityID,
						},
						RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
					}, raccess.EntityQueryOptionUnathorized)
					if err != nil {
						return nil, errors.InternalError(ctx, fmt.Errorf("Encountered an error while looking up parked entity %q: %s", inv.GetPatient().Patient.ParkedEntityID, err))
					} else if len(entities) > 1 {
						return "", errors.InternalError(ctx, fmt.Errorf("Expected 1 entity to be returned for %s but got back %d", inv.GetPatient().Patient.ParkedEntityID, len(entities)))
					}
					parkedEntity := entities[0]

					verifiedEntityInfo = &models.VerifiedEntityInfo{
						FirstName: parkedEntity.Info.FirstName,
						LastName:  parkedEntity.Info.LastName,
						Email:     resp.Value,
					}
				}
			}
		}

		return &checkVerificationCodeOutput{
			ClientMutationID:   in.ClientMutationID,
			Success:            true,
			Account:            transformAccountToResponse(resp.Account),
			VerifiedEntityInfo: verifiedEntityInfo,
		}, nil
	},
}
