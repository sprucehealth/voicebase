package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/graphql"
)

const (
	createAccountErrorCodeAccountExists           = "ACCOUNT_EXISTS"
	createAccountErrorCodeInvalidEmail            = "INVALID_EMAIL"
	createAccountErrorCodeInvalidFirstName        = "INVALID_FIRST_NAME"
	createAccountErrorCodeInvalidLastName         = "INVALID_LAST_NAME"
	createAccountErrorCodeInvalidOrganizationName = "INVALID_ORGANIZATION_NAME"
	createAccountErrorCodeInvalidPassword         = "INVALID_PASSWORD"
	createAccountErrorCodeInvalidPhoneNumber      = "INVALID_PHONE_NUMBER"
	createAccountErrorCodeInvalidDOB              = "INVALID_DOB"
	createAccountErrorCodeInviteRequired          = "INVITE_REQUIRED"
	createAccountErrorCodeInviteEmailMismatch     = "INVITE_EMAIL_MISMATCH"
	createAccountErrorCodeInvitePhoneMismatch     = "INVITE_PHONE_MISMATCH"
)

var createAccountErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "CreateAccountErrorCode",
	Values: graphql.EnumValueConfigMap{
		createAccountErrorCodeInvalidEmail: &graphql.EnumValueConfig{
			Value:       createAccountErrorCodeInvalidEmail,
			Description: "The provided email is invalid",
		},
		createAccountErrorCodeInvalidPassword: &graphql.EnumValueConfig{
			Value:       createAccountErrorCodeInvalidPassword,
			Description: "The provided password is invalid",
		},
		createAccountErrorCodeInvalidPhoneNumber: &graphql.EnumValueConfig{
			Value:       createAccountErrorCodeInvalidPhoneNumber,
			Description: "The provided phone number is invalid",
		},
		createAccountErrorCodeAccountExists: &graphql.EnumValueConfig{
			Value:       createAccountErrorCodeAccountExists,
			Description: "An account exists with the provided email address",
		},
		createAccountErrorCodeInvalidOrganizationName: &graphql.EnumValueConfig{
			Value:       createAccountErrorCodeInvalidOrganizationName,
			Description: "The provided organization name is invalid",
		},
		createAccountErrorCodeInvalidFirstName: &graphql.EnumValueConfig{
			Value:       createAccountErrorCodeInvalidFirstName,
			Description: "The provided first name is invalid",
		},
		createAccountErrorCodeInvalidLastName: &graphql.EnumValueConfig{
			Value:       createAccountErrorCodeInvalidLastName,
			Description: "The provided last name is invalid",
		},
		createAccountErrorCodeInvalidDOB: &graphql.EnumValueConfig{
			Value:       createAccountErrorCodeInvalidDOB,
			Description: "The provided date of birth is invalid",
		},
		createAccountErrorCodeInviteRequired: &graphql.EnumValueConfig{
			Value:       createAccountErrorCodeInviteRequired,
			Description: "An invite is required to create an account with this device",
		},
		createAccountErrorCodeInviteEmailMismatch: &graphql.EnumValueConfig{
			Value:       createAccountErrorCodeInviteEmailMismatch,
			Description: "The provided email doesn't match the invite",
		},
		createAccountErrorCodeInvitePhoneMismatch: &graphql.EnumValueConfig{
			Value:       createAccountErrorCodeInvitePhoneMismatch,
			Description: "The provided phone number doesn't match the invite",
		},
	},
})

type createAccountOutput struct {
	ClientMutationID    string         `json:"clientMutationId,omitempty"`
	Success             bool           `json:"success"`
	ErrorCode           string         `json:"errorCode,omitempty"`
	ErrorMessage        string         `json:"errorMessage,omitempty"`
	Token               string         `json:"token,omitempty"`
	Account             models.Account `json:"account,omitempty"`
	ClientEncryptionKey string         `json:"clientEncryptionKey,omitempty"`
}

var createAccountInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "CreateAccountInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId":       newClientMutationIDInputField(),
		"uuid":                   newUUIDInputField(),
		"email":                  &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"password":               &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"phoneNumber":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"firstName":              &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"lastName":               &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"shortTitle":             &graphql.InputObjectFieldConfig{Type: graphql.String},
		"longTitle":              &graphql.InputObjectFieldConfig{Type: graphql.String},
		"organizationName":       &graphql.InputObjectFieldConfig{Type: graphql.String},
		"phoneVerificationToken": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
	},
})

var createAccountOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "CreateAccountPayload",
	Fields: graphql.Fields{
		"clientMutationId":    newClientMutationIDOutputField(),
		"success":             &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":           &graphql.Field{Type: createAccountErrorCodeEnum},
		"errorMessage":        &graphql.Field{Type: graphql.String},
		"token":               &graphql.Field{Type: graphql.String},
		"account":             &graphql.Field{Type: accountInterfaceType},
		"clientEncryptionKey": &graphql.Field{Type: graphql.String},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*createAccountOutput)
		return ok
	},
})

var createAccountMutation = &graphql.Field{
	Type: graphql.NewNonNull(createAccountOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(createAccountInputType)},
	},
	DeprecationReason: "Replaced with createProviderAccount",
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		cpaOutput, err := createProviderAccount(p)
		if err != nil {
			return nil, err
		}
		return &createAccountOutput{
			ClientMutationID:    cpaOutput.ClientMutationID,
			Success:             cpaOutput.Success,
			ErrorCode:           cpaOutput.ErrorCode,
			ErrorMessage:        cpaOutput.ErrorMessage,
			Token:               cpaOutput.Token,
			Account:             cpaOutput.Account,
			ClientEncryptionKey: cpaOutput.ClientEncryptionKey,
		}, nil
	},
}
