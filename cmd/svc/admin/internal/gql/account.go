package gql

import (
	"context"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/common"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/client"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
)

// accountArgumentsConfig represents the config for arguments referencing an account
var accountArgumentsConfig = graphql.FieldConfigArgument{
	"accountID": &graphql.ArgumentConfig{Type: graphql.String},
	"email":     &graphql.ArgumentConfig{Type: graphql.String},
}

// accountArguments represents arguments for referencing an entity
type accountArguments struct {
	AccountID string `json:"accountID"`
	Email     string `json:"email"`
}

// parseAccountArguments parses the account arguments out of requests params
func parseAccountArguments(args map[string]interface{}) *accountArguments {
	accArgs := &accountArguments{}
	if args != nil {
		if iid, ok := args["accountID"]; ok {
			if id, ok := iid.(string); ok {
				accArgs.AccountID = id
			}
		}
		if iemail, ok := args["email"]; ok {
			if email, ok := iemail.(string); ok {
				accArgs.Email = email
			}
		}
	}
	return accArgs
}

// accountType returns is a type object representing a baymax account
var accountType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Account",
		Fields: graphql.Fields{
			"id":          &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"type":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"status":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"firstName":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"lastName":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"email":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"phoneNumber": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"entities":    &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(entityType)), Resolve: accountEntitiesResolve},
			"settings":    &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(settingType)), Resolve: accountSettingsResolve},
		},
	})

// accountField represents a graphql field for Querying an Account object
var accountField = &graphql.Field{
	Type:    accountType,
	Args:    accountArgumentsConfig,
	Resolve: accountResolve,
}

func accountResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	args := parseAccountArguments(p.Args)
	golog.ContextLogger(ctx).Debugf("Resolving Account with args %+v", args)
	if args.AccountID != "" {
		return getAccountByID(ctx, client.Auth(p), args.AccountID)
	} else if args.Email != "" {
		return getAccountByEmail(ctx, client.Auth(p), args.Email)
	}
	return nil, errors.Errorf("Email or AccountID required")
}

func getAccountByID(ctx context.Context, authClient auth.AuthClient, accountID string) (*models.Account, error) {
	resp, err := authClient.GetAccount(ctx, &auth.GetAccountRequest{
		AccountID: accountID,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	contacts, err := getAccountContacts(ctx, authClient, accountID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return models.TransformAccountToModel(resp.Account, contacts.Email, contacts.PhoneNumber), nil
}

func getAccountByEmail(ctx context.Context, authClient auth.AuthClient, email string) (*models.Account, error) {
	resp, err := authClient.GetAccount(ctx, &auth.GetAccountRequest{
		AccountEmail: email,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	contacts, err := getAccountContacts(ctx, authClient, resp.Account.ID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return models.TransformAccountToModel(resp.Account, contacts.Email, contacts.PhoneNumber), nil
}

func getAccountContacts(ctx context.Context, authClient auth.AuthClient, accountID string) (*auth.GetAccountContactsResponse, error) {
	return authClient.GetAccountContacts(ctx, &auth.GetAccountContactsRequest{
		AccountID: accountID,
	})
}

func accountEntitiesResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	account := p.Source.(*models.Account)
	golog.ContextLogger(ctx).Debugf("Looking up account entities for %s", account.ID)
	return entitiesForAccount(ctx, client.Directory(p), account.ID)
}

func entitiesForAccount(ctx context.Context, directoryClient directory.DirectoryClient, accountID string) ([]*models.Entity, error) {
	// Collect all the entities mapped to the account
	resp, err := directoryClient.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ACCOUNT_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_AccountID{
			AccountID: accountID,
		},
		RootTypes: []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return models.TransformEntitiesToModels(resp.Entities), nil
}

func accountSettingsResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	account := p.Source.(*models.Account)
	golog.ContextLogger(ctx).Debugf("Looking up account settings for %s", account.ID)
	return getNodeSettings(ctx, client.Settings(p), account.ID)
}

// modifyAccountContact

type modifyAccountContactInput struct {
	AccountID   string `gql:"accountID,nonempty"`
	Email       string `gql:"email"`
	PhoneNumber string `gql:"phoneNumber"`
}

var modifyAccountContactInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "ModifyAccountContactInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"accountID":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"email":       &graphql.InputObjectFieldConfig{Type: graphql.String},
			"phoneNumber": &graphql.InputObjectFieldConfig{Type: graphql.String},
		},
	},
)

type modifyAccountContactOutput struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

var modifyAccountContactOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ModifyAccountContactPayload",
		Fields: graphql.Fields{
			"success":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorMessage": &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*modifyAccountContactOutput)
			return ok
		},
	},
)

var modifyAccountContactField = &graphql.Field{
	Type: graphql.NewNonNull(modifyAccountContactOutputType),
	Args: graphql.FieldConfigArgument{
		common.InputFieldName: &graphql.ArgumentConfig{Type: graphql.NewNonNull(modifyAccountContactInputType)},
	},
	Resolve: modifyAccountContactResolve,
}

func modifyAccountContactResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	var in modifyAccountContactInput
	if err := gqldecode.Decode(p.Args[common.InputFieldName].(map[string]interface{}), &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, errors.Errorf("%s is invalid: %s", err.Field, err.Reason)
		}
		return nil, errors.Trace(err)
	}

	return modifyAccountContact(ctx, &in, client.Auth(p))
}

func modifyAccountContact(ctx context.Context, in *modifyAccountContactInput, authClient auth.AuthClient) (*modifyAccountContactOutput, error) {
	in.Email = strings.TrimSpace(in.Email)
	if in.Email != "" {
		if !validate.Email(in.Email) {
			return nil, errors.Errorf("Invalid email %q", in.Email)
		}
	}
	var phoneNumber string
	in.PhoneNumber = strings.TrimSpace(in.PhoneNumber)
	if in.PhoneNumber != "" {
		parsedNumber, err := phone.ParseNumber(in.PhoneNumber)
		if err != nil {
			return nil, errors.Errorf("Error parsing phone number %q", in.PhoneNumber)
		}
		phoneNumber = parsedNumber.String()
	}
	if _, err := authClient.UpdateAccountContacts(ctx, &auth.UpdateAccountContactsRequest{
		AccountID:   in.AccountID,
		PhoneNumber: phoneNumber,
		Email:       in.Email,
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &modifyAccountContactOutput{
		Success: true,
	}, nil
}

// disableAccount

type disableAccountInput struct {
	AccountID string `gql:"accountID"`
}

var disableAccountInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "DisableAccountInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"accountID": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

type disableAccountOutput struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

var disableAccountOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "DisableAccountPayload",
		Fields: graphql.Fields{
			"success":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorMessage": &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*disableAccountOutput)
			return ok
		},
	},
)

var disableAccountField = &graphql.Field{
	Type: graphql.NewNonNull(disableAccountOutputType),
	Args: graphql.FieldConfigArgument{
		common.InputFieldName: &graphql.ArgumentConfig{Type: graphql.NewNonNull(disableAccountInputType)},
	},
	Resolve: disableAccountResolve,
}

func disableAccountResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	var in disableAccountInput
	if err := gqldecode.Decode(p.Args[common.InputFieldName].(map[string]interface{}), &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, errors.Errorf("%s is invalid: %s", err.Field, err.Reason)
		}
		return nil, errors.Trace(err)
	}

	return disableAccount(ctx, &in, client.Auth(p), client.Directory(p))
}

func disableAccount(ctx context.Context, in *disableAccountInput, authClient auth.AuthClient, directoryClient directory.DirectoryClient) (*disableAccountOutput, error) {
	// Delete the account in the auth service. This should be idempotent
	if _, err := authClient.DeleteAccount(ctx, &auth.DeleteAccountRequest{
		AccountID: in.AccountID,
	}); err != nil {
		return nil, errors.Trace(err)
	}

	entities, err := entitiesForAccount(ctx, directoryClient, in.AccountID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	for _, ent := range entities {
		// Delete the related entities. This should be idempotent
		if _, err := directoryClient.DeleteEntity(ctx, &directory.DeleteEntityRequest{
			EntityID: ent.ID,
		}); err != nil {
			return nil, errors.Trace(err)
		}
	}

	// TODO: In the future we may need to delete push tokens for the device but for now we can assume they won't get push notifications for deleted entities
	// The notification service currently doesn't expose a gRPC API or hav eeasy mechanisms for clearing all for an account
	return &disableAccountOutput{
		Success: true,
	}, nil
}
