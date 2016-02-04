package main

import (
	"errors"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// authenticate

const (
	authenticateResultSuccess            = "SUCCESS"
	authenticateResultSuccess2FARequired = "SUCCESS_2FA_REQUIRED"
	authenticateResultInvalidEmail       = "INVALID_EMAIL"
	authenticateResultInvalidPassword    = "INVALID_PASSWORD"
	authenticateResultInvalidCode        = "INVALID_CODE"
)

type authenticateOutput struct {
	ClientMutationID      string   `json:"clientMutationId"`
	Result                string   `json:"result"`
	Token                 string   `json:"token,omitempty"`
	Account               *account `json:"account,omitempty"`
	PhoneNumberLastDigits string   `json:"phoneNumberLastDigits,omitempty"`
	ClientEncryptionKey   string   `json:"clientEncryptionKey,omitempty"`
}

var authenticateResultType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "AuthenticateResult",
		Description: "Result of authenticate mutation",
		Values: graphql.EnumValueConfigMap{
			authenticateResultSuccess: &graphql.EnumValueConfig{
				Value:       authenticateResultSuccess,
				Description: "Success",
			},
			authenticateResultSuccess2FARequired: &graphql.EnumValueConfig{
				Value:       authenticateResultSuccess2FARequired,
				Description: "Success but 2FA is required",
			},
			authenticateResultInvalidEmail: &graphql.EnumValueConfig{
				Value:       authenticateResultInvalidEmail,
				Description: "Email not found",
			},
			authenticateResultInvalidPassword: &graphql.EnumValueConfig{
				Value:       authenticateResultInvalidPassword,
				Description: "Password doesn't match",
			},
			authenticateResultInvalidCode: &graphql.EnumValueConfig{
				Value:       authenticateResultInvalidCode,
				Description: "Code doesn't match",
			},
		},
	},
)

var authenticateInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "AuthenticateInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"email":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"password":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

var authenticateOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "AuthenticatePayload",
		Fields: graphql.Fields{
			"clientMutationId":      newClientmutationIDOutputField(),
			"result":                &graphql.Field{Type: graphql.NewNonNull(authenticateResultType)},
			"token":                 &graphql.Field{Type: graphql.String},
			"account":               &graphql.Field{Type: accountType},
			"phoneNumberLastDigits": &graphql.Field{Type: graphql.String},
			"clientEncryptionKey":   &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*authenticateOutput)
			return ok
		},
	},
)

var authenticateField = &graphql.Field{
	Type: graphql.NewNonNull(authenticateOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(authenticateInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		email := input["email"].(string)
		if !validate.Email(email) {
			return nil, errors.New("invalid email")
		}
		password := input["password"].(string)
		res, err := svc.auth.AuthenticateLogin(ctx, &auth.AuthenticateLoginRequest{
			Email:    email,
			Password: password,
		})
		if err != nil {
			switch grpc.Code(err) {
			case auth.EmailNotFound:
				return &authenticateOutput{
					ClientMutationID: mutationID,
					Result:           authenticateResultInvalidEmail,
				}, nil
			case auth.BadPassword:
				return &authenticateOutput{
					ClientMutationID: mutationID,
					Result:           authenticateResultInvalidPassword,
				}, nil
			default:
				return nil, internalError(err)
			}
		}
		var token string
		var clientEncryptionKey string
		var expires time.Time
		var acc *account
		var phoneNumberLastDigits string
		authResult := authenticateResultSuccess
		if res.TwoFactorRequired {
			authResult = authenticateResultSuccess2FARequired
			token, err = svc.createAndSendSMSVerificationCode(ctx, auth.VerificationCodeType_ACCOUNT_2FA, res.Account.ID, res.TwoFactorPhoneNumber)
			if err != nil {
				return nil, internalError(err)
			}
			if len(res.TwoFactorPhoneNumber) > 2 {
				phoneNumberLastDigits = res.TwoFactorPhoneNumber[len(res.TwoFactorPhoneNumber)-2:]
			}

		} else {
			token = res.Token.Value
			expires = time.Unix(int64(res.Token.ExpirationEpoch), 0)
			acc = &account{
				ID: res.Account.ID,
			}
			clientEncryptionKey = res.Token.ClientEncryptionKey

			// TODO: updating the context this is safe for now because the GraphQL pkg serializes mutations.
			// that likely won't change, but this still isn't a great way to update the context.
			*ctx.Value(ctxAccount).(*account) = *acc
			result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
			result.Set("auth_token", token)
			result.Set("auth_expiration", expires)
		}

		return &authenticateOutput{
			ClientMutationID:      mutationID,
			Result:                authResult,
			Token:                 token,
			Account:               acc,
			PhoneNumberLastDigits: phoneNumberLastDigits,
			ClientEncryptionKey:   clientEncryptionKey,
		}, nil
	},
}

// authenticateWithCode

var authenticateWithCodeInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "AuthenticateWithCodeInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"token":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"code":             &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

var authenticateWithCodeField = &graphql.Field{
	Type: graphql.NewNonNull(authenticateOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(authenticateWithCodeInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		token := input["token"].(string)
		code := input["code"].(string)
		res, err := svc.auth.AuthenticateLoginWithCode(ctx, &auth.AuthenticateLoginWithCodeRequest{
			Token: token,
			Code:  code,
		})
		if err != nil {
			switch grpc.Code(err) {
			case auth.BadVerificationCode, codes.NotFound:
				return &authenticateOutput{
					ClientMutationID: mutationID,
					Result:           authenticateResultInvalidCode,
				}, nil
			default:
				return nil, internalError(err)
			}
		}
		result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
		result.Set("auth_token", res.Token.Value)
		result.Set("auth_expiration", time.Unix(int64(res.Token.ExpirationEpoch), 0))

		// TODO: updating the context this is safe for now because the GraphQL pkg serializes mutations.
		// that likely won't change, but this still isn't a great way to update the context.
		acc := &account{
			ID: res.Account.ID,
		}
		*ctx.Value(ctxAccount).(*account) = *acc
		return &authenticateOutput{
			ClientMutationID:    mutationID,
			Result:              authenticateResultSuccess,
			Token:               res.Token.Value,
			ClientEncryptionKey: res.Token.ClientEncryptionKey,
			Account:             acc,
		}, nil
	},
}

/// unauthenticate

type unauthenticateOutput struct {
	ClientMutationID string `json:"clientMutationId"`
}

var unauthenticateInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "UnauthenticateInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"token":            &graphql.InputObjectFieldConfig{Type: graphql.String},
		},
	},
)

var unauthenticateOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "UnauthenticatePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*unauthenticateOutput)
			return ok
		},
	},
)

var unauthenticateField = &graphql.Field{
	Type: graphql.NewNonNull(unauthenticateOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: unauthenticateInputType},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		// TODO: get token from cookie if not provided in args
		token, ok := input["token"].(string)
		if !ok {
			return nil, internalError(errors.New("TODO: unauthenticate using cookie is not yet implemented"))
		}
		_, err := svc.auth.Unauthenticate(ctx, &auth.UnauthenticateRequest{Token: token})
		if err != nil {
			return nil, internalError(err)
		}
		result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
		result.Set("unauthenticated", true)
		return &unauthenticateOutput{
			ClientMutationID: mutationID,
		}, nil
	},
}
