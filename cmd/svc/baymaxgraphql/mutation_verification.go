package main

import (
	"errors"
	"fmt"

	"google.golang.org/grpc"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/auth"
)

// verifyPhoneNumber

type verifyPhoneNumberOutput struct {
	ClientMutationID string `json:"clientMutationId"`
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
			"token":            &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
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
		"input": &graphql.ArgumentConfig{Type: verifyPhoneNumberInputType},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		pn, _ := input["phoneNumber"].(string)

		token, err := svc.createAndSendSMSVerificationCode(ctx, auth.VerificationCodeType_PHONE, pn, pn)
		if err != nil {
			return nil, internalError(err)
		}

		last4Phone := pn
		if len(last4Phone) > 4 {
			last4Phone = last4Phone[len(last4Phone)-4:]
		}
		return &verifyPhoneNumberOutput{
			ClientMutationID: mutationID,
			Token:            token,
			Message:          fmt.Sprintf("A verification code has been sent to %s", pn),
		}, nil
	},
}

// verifyPhoneNumberForAccountCreation

var verifyPhoneNumberForAccountCreationMutation = verifyPhoneNumberMutation

// checkVerificationCode

const (
	checkVerificationCodeResultSuccess      = "SUCCESS"
	checkVerificationCodeResultFailure      = "VERIFICATION_FAILED"
	checkVerificationCodeResultExpired      = "CODE_EXPIRED"
	checkVerificationCodeResultDoesNotMatch = "DOES_NOT_MATCH"
)

type checkVerificationCodeOutput struct {
	ClientMutationID string   `json:"clientMutationId"`
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
			checkVerificationCodeResultDoesNotMatch: &graphql.EnumValueConfig{
				Value:       checkVerificationCodeResultDoesNotMatch,
				Description: "Phone number does not match",
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
		"input": &graphql.ArgumentConfig{Type: checkVerificationCodeInputType},
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
