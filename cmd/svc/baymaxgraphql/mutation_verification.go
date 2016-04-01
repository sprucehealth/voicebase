package main

import (
	"fmt"
	"runtime/debug"

	"github.com/sprucehealth/backend/svc/directory"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/invite"

	"github.com/sprucehealth/graphql"
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

var verifyPhoneNumberInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "VerifyPhoneNumberInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             newUUIDInputField(),
		"phoneNumber": &graphql.InputObjectFieldConfig{
			Type:        graphql.NewNonNull(graphql.String),
			Description: "Specify the phone number to send a verification code to.",
		},
	},
})

var verifyPhoneNumberOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "VerifyPhoneNumberPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientmutationIDOutputField(),
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
		mutationID, _ := input["clientMutationId"].(string)
		pn, err := phone.ParseNumber(input["phoneNumber"].(string))
		if err != nil {
			return &verifyPhoneNumberOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        verifyPhoneNumberErrorCodeInvalidPhone,
				ErrorMessage:     "Please use a valid U.S. phone number",
			}, nil
		}

		// ensure that the phone number is not a provisioned phone number
		entities, err := ram.EntitiesByContact(ctx, pn.String(), []directory.EntityInformation{
			directory.EntityInformation_CONTACTS,
		}, 0, []directory.EntityStatus{
			directory.EntityStatus_ACTIVE,
		})
		if err != nil {
			golog.Errorf("Unable to lookup entity by contact: %s", err.Error())
		}

		for _, ent := range entities {
			for _, c := range ent.Contacts {
				if c.Provisioned && c.Value == pn.String() {
					return &verifyPhoneNumberOutput{
						ClientMutationID: mutationID,
						Success:          false,
						ErrorCode:        verifyPhoneNumberErrorCodeInvalidPhone,
						ErrorMessage:     "Please use a non-Spruce number to create an account with.",
					}, nil
				}
			}
		}

		// Provided phone number must match what was provided during invite if here through invite
		if forAccountCreation {
			inv, err := svc.inviteInfo(ctx)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			if inv != nil {
				switch inv.Type {
				case invite.LookupInviteResponse_COLLEAGUE:
					col := inv.GetColleague().Colleague
					if !ignorePhoneNumberCheckForInvite(inv) && col.PhoneNumber != pn.String() {
						return &verifyPhoneNumberOutput{
							ClientMutationID: mutationID,
							Success:          false,
							ErrorCode:        verifyPhoneNumberErrorCodeInvitePhoneMismatch,
							ErrorMessage:     "The phone number must match the one that was in your invite.",
						}, nil
					}
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
			ClientMutationID: mutationID,
			Success:          true,
			Token:            token,
			Message:          fmt.Sprintf("A verification code has been sent to %s", nicePhone),
		}, nil
	}
}

// ignorePhoneNumberCheckForOrgs contains a list of orgIDs scoped by
// environment for which to ignore the phone number match during the invite process.
var ignorePhoneNumberCheckForOrgs = []string{
	"staging:entity_0CGGMPGPG1800",
	"prod:entity_0D6LQNE983O00",
}

// ignorePhoneNumberCheckForInvite returns true if the phone number check is to be
// ignored for a particular org such that anyone with a valid link to join an org
// can do so.
// TODO: This is a hack to get one of our providers setup to explore the use-case of network-wide chat with a single
// invite link. Remove this once we have built a better way for invite links to work.
func ignorePhoneNumberCheckForInvite(inv *invite.LookupInviteResponse) bool {
	if inv == nil {
		return true
	}

	orgKey := environment.GetCurrent() + ":" + inv.GetColleague().OrganizationEntityID
	for _, o := range ignorePhoneNumberCheckForOrgs {
		if orgKey == o {
			return true
		}
	}
	return false
}

// checkVerificationCode

const (
	checkVerificationCodeErrorCodeFailure = "VERIFICATION_FAILED"
	checkVerificationCodeErrorCodeExpired = "CODE_EXPIRED"
)

type checkVerificationCodeOutput struct {
	ClientMutationID string         `json:"clientMutationId,omitempty"`
	Success          bool           `json:"success"`
	ErrorCode        string         `json:"errorCode,omitempty"`
	ErrorMessage     string         `json:"errorMessage,omitempty"`
	Account          models.Account `json:"account"`
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
			"clientMutationId": newClientMutationIDInputField(),
			"uuid":             newUUIDInputField(),
			"token":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"code":             &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

var checkVerificationCodeOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CheckVerificationCodePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: checkVerificationCodeErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"account":          &graphql.Field{Type: accountInterfaceType},
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
		ram := raccess.ResourceAccess(p)
		ctx := p.Context

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		token := input["token"].(string)
		code := input["code"].(string)

		golog.Debugf("Checking token %s against code %s", token, code)
		resp, err := ram.CheckVerificationCode(ctx, token, code)
		if grpc.Code(err) == auth.BadVerificationCode {
			return &checkVerificationCodeOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        checkVerificationCodeErrorCodeFailure,
				ErrorMessage:     "The entered code is incorrect.",
			}, nil
		} else if grpc.Code(err) == auth.VerificationCodeExpired {
			return &checkVerificationCodeOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        checkVerificationCodeErrorCodeExpired,
				ErrorMessage:     "The entered code has expired. Please request a new code.",
			}, nil
		} else if err != nil {
			golog.Errorf(err.Error())
			return nil, errors.New("Failed to check verification code")
		}

		return &checkVerificationCodeOutput{
			ClientMutationID: mutationID,
			Success:          true,
			Account:          transformAccountToResponse(resp.Account),
		}, nil
	},
}
