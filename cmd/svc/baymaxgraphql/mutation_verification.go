package main

import (
	"fmt"
	"runtime/debug"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
)

// verifyPhoneNumber

const (
	verifyPhoneNumberResultSuccess             = "SUCCESS"
	verifyPhoneNumberResultInvitePhoneMismatch = "INVITE_PHONE_MISMATCH"
)

var verifyPhoneNumberResultType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "VerifyPhoneNumberResult",
		Description: "Result of verifyPhoneNumber mutation",
		Values: graphql.EnumValueConfigMap{
			verifyPhoneNumberResultSuccess: &graphql.EnumValueConfig{
				Value:       verifyPhoneNumberResultSuccess,
				Description: "Success",
			},
			verifyPhoneNumberResultInvitePhoneMismatch: &graphql.EnumValueConfig{
				Value:       verifyPhoneNumberResultInvitePhoneMismatch,
				Description: "Phone number from invite does not match",
			},
		},
	},
)

type verifyPhoneNumberOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Result           string `json:"result"`
	Token            string `json:"token"`
	Message          string `json:"message"`
}

var verifyPhoneNumberInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "VerifyPhoneNumberInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"uuid":             newUUIDInputField(),
			"phoneNumber": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "Specify the phone number to send a verification code to.",
			},
		},
	},
)

var verifyPhoneNumberOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "VerifyPhoneNumberPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"result":           &graphql.Field{Type: graphql.NewNonNull(verifyPhoneNumberResultType)},
			"token":            &graphql.Field{Type: graphql.String},
			"message":          &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*verifyPhoneNumberOutput)
			return ok
		},
	},
)

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
		ctx := p.Context

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		pn, err := phone.ParseNumber(input["phoneNumber"].(string))
		if err != nil {
			return nil, errors.New("Phone number is invalid")
		}

		// Provided phone number must match what was provided during invite if here through invite
		if forAccountCreation {
			inv, err := svc.inviteInfo(ctx)
			if err != nil {
				return nil, internalError(err)
			}
			if inv != nil {
				switch inv.Type {
				case invite.LookupInviteResponse_COLLEAGUE:
					col := inv.GetColleague().Colleague
					if col.PhoneNumber != pn.String() {
						return &verifyPhoneNumberOutput{
							ClientMutationID: mutationID,
							Result:           verifyPhoneNumberResultInvitePhoneMismatch,
							Message:          "The phone number did not match.",
						}, nil
					}
				default:
					golog.Errorf("Unknown invite type %s", inv.Type.String())
				}
			}
		}

		token, err := svc.createAndSendSMSVerificationCode(ctx, auth.VerificationCodeType_PHONE, pn.String(), pn)
		if err != nil {
			return nil, internalError(err)
		}

		nicePhone, err := pn.Format(phone.Pretty)
		if err != nil {
			return nil, internalError(err)
		}
		return &verifyPhoneNumberOutput{
			ClientMutationID: mutationID,
			Result:           verifyPhoneNumberResultSuccess,
			Token:            token,
			Message:          fmt.Sprintf("A verification code has been sent to %s", nicePhone),
		}, nil
	}
}

// checkVerificationCode

const (
	checkVerificationCodeResultSuccess = "SUCCESS"
	checkVerificationCodeResultFailure = "VERIFICATION_FAILED"
	checkVerificationCodeResultExpired = "CODE_EXPIRED"
)

type checkVerificationCodeOutput struct {
	ClientMutationID string   `json:"clientMutationId,omitempty"`
	Result           string   `json:"result"`
	Account          *account `json:"account"`
}

var checkVerificationCodeResultType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "CheckVerificationCodeResult",
		Description: "Result of checkVerificationCode mutation",
		Values: graphql.EnumValueConfigMap{
			checkVerificationCodeResultSuccess: &graphql.EnumValueConfig{
				Value:       checkVerificationCodeResultSuccess,
				Description: "Success",
			},
			checkVerificationCodeResultExpired: &graphql.EnumValueConfig{
				Value:       checkVerificationCodeResultExpired,
				Description: "Code expired",
			},
			checkVerificationCodeResultFailure: &graphql.EnumValueConfig{
				Value:       checkVerificationCodeResultFailure,
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
			"result":           &graphql.Field{Type: graphql.NewNonNull(checkVerificationCodeResultType)},
			"account":          &graphql.Field{Type: accountType},
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
		ctx := p.Context

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		token := input["token"].(string)
		code := input["code"].(string)

		golog.Debugf("Checking token %s against code %s", token, code)
		resp, err := svc.auth.CheckVerificationCode(ctx, &auth.CheckVerificationCodeRequest{
			Token: token,
			Code:  code,
		})
		if grpc.Code(err) == auth.BadVerificationCode {
			return &checkVerificationCodeOutput{
				ClientMutationID: mutationID,
				Result:           checkVerificationCodeResultFailure,
			}, nil
		} else if grpc.Code(err) == auth.VerificationCodeExpired {
			return &checkVerificationCodeOutput{
				ClientMutationID: mutationID,
				Result:           checkVerificationCodeResultExpired,
			}, nil
		} else if err != nil {
			golog.Errorf(err.Error())
			return nil, errors.New("Failed to check verification code")
		}

		var acc *account
		if resp.Account != nil {
			acc = &account{
				ID: resp.Account.ID,
			}
		}

		return &checkVerificationCodeOutput{
			ClientMutationID: mutationID,
			Result:           checkVerificationCodeResultSuccess,
			Account:          acc,
		}, nil
	},
}
