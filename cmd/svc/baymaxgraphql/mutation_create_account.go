package main

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
)

type createAccountOutput struct {
	ClientMutationID    string   `json:"clientMutationId,omitempty"`
	Success             bool     `json:"success"`
	ErrorCode           string   `json:"errorCode,omitempty"`
	ErrorMessage        string   `json:"errorMessage,omitempty"`
	Token               string   `json:"token,omitempty"`
	Account             *account `json:"account,omitempty"`
	ClientEncryptionKey string   `json:"clientEncryptionKey,omitempty"`
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

const (
	createAccountErrorCodeAccountExists           = "ACCOUNT_EXISTS"
	createAccountErrorCodeInvalidEmail            = "INVALID_EMAIL"
	createAccountErrorCodeInvalidFirstName        = "INVALID_FIRST_NAME"
	createAccountErrorCodeInvalidLastName         = "INVALID_LAST_NAME"
	createAccountErrorCodeInvalidOrganizationName = "INVALID_ORGANIZATION_NAME"
	createAccountErrorCodeInvalidPassword         = "INVALID_PASSWORD"
	createAccountErrorCodeInvalidPhoneNumber      = "INVALID_PHONE_NUMBER"
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
	},
})

var createAccountOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "CreateAccountPayload",
	Fields: graphql.Fields{
		"clientMutationId":    newClientmutationIDOutputField(),
		"success":             &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":           &graphql.Field{Type: createAccountErrorCodeEnum},
		"errorMessage":        &graphql.Field{Type: graphql.String},
		"token":               &graphql.Field{Type: graphql.String},
		"account":             &graphql.Field{Type: accountType},
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
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)

		inv, err := svc.inviteInfo(ctx)
		if err != nil {
			return nil, internalError(ctx, err)
		}
		// Sanity check to make sure we fail early in case we forgot to handle all new invite types
		if inv != nil && inv.Type != invite.LookupInviteResponse_COLLEAGUE {
			return nil, internalError(ctx, fmt.Errorf("unknown invite type %s", inv.Type.String()))
		}

		req := &auth.CreateAccountRequest{
			Email:    input["email"].(string),
			Password: input["password"].(string),
		}
		if !validate.Email(req.Email) {
			return &createAccountOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        createAccountErrorCodeInvalidEmail,
				ErrorMessage:     "Please enter a valid email address.",
			}, nil
		}
		entityInfo, err := entityInfoFromInput(input)
		if err != nil {
			return nil, internalError(ctx, err)
		}

		req.FirstName = entityInfo.FirstName
		req.LastName = entityInfo.LastName
		if req.FirstName == "" || !isValidPlane0Unicode(req.FirstName) {
			return &createAccountOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        createAccountErrorCodeInvalidFirstName,
				ErrorMessage:     "Please enter a valid first name.",
			}, nil
		}
		if req.LastName == "" || !isValidPlane0Unicode(req.LastName) {
			return &createAccountOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        createAccountErrorCodeInvalidLastName,
				ErrorMessage:     "Please enter a valid last name.",
			}, nil
		}

		var organizationName string
		if inv == nil {
			organizationName, _ = input["organizationName"].(string)
			if organizationName == "" || !isValidPlane0Unicode(req.LastName) {
				return &createAccountOutput{
					ClientMutationID: mutationID,
					Success:          false,
					ErrorCode:        createAccountErrorCodeInvalidOrganizationName,
					ErrorMessage:     "Please enter a valid organization name.",
				}, nil
			}
		}
		respVerifiedValue, err := svc.auth.VerifiedValue(ctx, &auth.VerifiedValueRequest{
			Token: input["phoneVerificationToken"].(string),
		})
		if grpc.Code(err) == auth.ValueNotYetVerified {
			return nil, errors.New("The phone number for the provided token has not yet been verified.")
		} else if err != nil {
			return nil, internalError(ctx, err)
		}
		vpn, err := phone.ParseNumber(respVerifiedValue.Value)
		if err != nil {
			return nil, internalError(ctx, err)
		}
		pn, err := phone.ParseNumber(input["phoneNumber"].(string))
		if err != nil {
			return &createAccountOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        createAccountErrorCodeInvalidPhoneNumber,
				ErrorMessage:     "Please enter a valid phone number.",
			}, nil
		}
		req.PhoneNumber = pn.String()
		if vpn.String() != pn.String() {
			golog.Debugf("The provided phone number %q does not match the number validated by the provided token %s", pn.String(), vpn.String())
			return nil, fmt.Errorf("The provided phone number %q does not match the number validated by the provided token", req.PhoneNumber)
		}
		res, err := svc.auth.CreateAccount(ctx, req)
		if err != nil {
			switch grpc.Code(err) {
			case auth.DuplicateEmail:
				return &createAccountOutput{
					ClientMutationID: mutationID,
					Success:          false,
					ErrorCode:        createAccountErrorCodeAccountExists,
					ErrorMessage:     "An account already exists with the entered email address.",
				}, nil
			case auth.InvalidEmail:
				return &createAccountOutput{
					ClientMutationID: mutationID,
					Success:          false,
					ErrorCode:        createAccountErrorCodeInvalidEmail,
					ErrorMessage:     "Please enter a valid email address.",
				}, nil
			case auth.InvalidPhoneNumber:
				return &createAccountOutput{
					ClientMutationID: mutationID,
					Success:          false,
					ErrorCode:        createAccountErrorCodeInvalidPhoneNumber,
					ErrorMessage:     "Please enter a valid phone number.",
				}, nil
			}
			return nil, internalError(ctx, err)
		}
		accountID := res.Account.ID

		var orgEntityID string
		var accEntityID string
		{
			if inv == nil {
				// Create organization
				res, err := svc.directory.CreateEntity(ctx, &directory.CreateEntityRequest{
					EntityInfo: &directory.EntityInfo{
						GroupName:   organizationName,
						DisplayName: organizationName,
					},
					Type: directory.EntityType_ORGANIZATION,
				})
				if err != nil {
					return nil, internalError(ctx, err)
				}
				orgEntityID = res.Entity.ID
			} else {
				orgEntityID = inv.GetColleague().OrganizationEntityID
			}

			contacts := []*directory.Contact{
				{
					ContactType: directory.ContactType_PHONE,
					Value:       req.PhoneNumber,
					Provisioned: false,
				},
			}
			entityInfo.DisplayName, err = buildDisplayName(entityInfo, contacts)
			if err != nil {
				return nil, internalError(ctx, err)
			}
			// Create entity
			res, err := svc.directory.CreateEntity(ctx, &directory.CreateEntityRequest{
				EntityInfo:                entityInfo,
				Type:                      directory.EntityType_INTERNAL,
				ExternalID:                accountID,
				InitialMembershipEntityID: orgEntityID,
				Contacts:                  contacts,
			})
			if err != nil {
				return nil, internalError(ctx, err)
			}
			accEntityID = res.Entity.ID
		}

		// Create a default saved query
		_, err = svc.threading.CreateSavedQuery(ctx, &threading.CreateSavedQueryRequest{
			OrganizationID: orgEntityID,
			EntityID:       accEntityID,
			// TODO: query
		})
		if err != nil {
			return nil, internalError(ctx, err)
		}

		result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
		result.Set("auth_token", res.Token.Value)
		result.Set("auth_expiration", time.Unix(int64(res.Token.ExpirationEpoch), 0))

		acc, err := transformAccountToResponse(res.Account)
		if err != nil {
			return nil, internalError(ctx, err)
		}
		// TODO: updating the context this is safe for now because the GraphQL pkg serializes mutations.
		// that likely won't change, but this still isn't a great way to update the context.
		*ctx.Value(ctxAccount).(*account) = *acc
		return &createAccountOutput{
			ClientMutationID:    mutationID,
			Success:             true,
			Token:               res.Token.Value,
			Account:             acc,
			ClientEncryptionKey: res.Token.ClientEncryptionKey,
		}, nil
	},
}
