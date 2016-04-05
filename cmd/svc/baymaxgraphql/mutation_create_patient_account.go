package main

import (
	"strings"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
)

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
	Name:        "GenderType",
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
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             newUUIDInputField(),
		"email":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"password":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"firstName":        &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"lastName":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"dob":              &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(dateInputType)},
		"gender":           &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(genderEnumType)},
		// TODO: This will not stay as is. This will be retrieved from the invite code
		"phoneNumber": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
	},
})

var createPatientAccountOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "CreatePatientAccountPayload",
	Fields: graphql.Fields{
		"clientMutationId":    newClientmutationIDOutputField(),
		"success":             &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":           &graphql.Field{Type: createAccountErrorCodeEnum},
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
	ram := raccess.ResourceAccess(p)
	ctx := p.Context
	input := p.Args["input"].(map[string]interface{})
	mutationID, _ := input["clientMutationId"].(string)

	req := &auth.CreateAccountRequest{
		Email:    input["email"].(string),
		Password: input["password"].(string),
		Type:     auth.AccountType_PATIENT,
	}
	req.Email = strings.TrimSpace(req.Email)
	if !validate.Email(req.Email) {
		return &createPatientAccountOutput{
			ClientMutationID: mutationID,
			Success:          false,
			ErrorCode:        createAccountErrorCodeInvalidEmail,
			ErrorMessage:     "Please enter a valid email address.",
		}, nil
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
			ErrorCode:        createAccountErrorCodeInvalidFirstName,
			ErrorMessage:     "Please enter a valid first name.",
		}, nil
	}
	if req.LastName == "" || !isValidPlane0Unicode(req.LastName) {
		return &createPatientAccountOutput{
			ClientMutationID: mutationID,
			Success:          false,
			ErrorCode:        createAccountErrorCodeInvalidLastName,
			ErrorMessage:     "Please enter a valid last name.",
		}, nil
	}
	// TODO: This will come from the token
	pn, err := phone.ParseNumber(input["phoneNumber"].(string))
	if err != nil {
		return &createPatientAccountOutput{
			ClientMutationID: mutationID,
			Success:          false,
			ErrorCode:        createAccountErrorCodeInvalidPhoneNumber,
			ErrorMessage:     "Please enter a valid phone number.",
		}, nil
	}
	req.PhoneNumber = pn.String()
	contacts := []*directory.Contact{
		{
			ContactType: directory.ContactType_PHONE,
			Value:       req.PhoneNumber,
			Provisioned: false,
		},
	}
	res, err := ram.CreateAccount(ctx, req)
	if err != nil {
		switch grpc.Code(err) {
		case auth.DuplicateEmail:
			return &createPatientAccountOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        createAccountErrorCodeAccountExists,
				ErrorMessage:     "An account already exists with the entered email address.",
			}, nil
		case auth.InvalidEmail:
			return &createPatientAccountOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        createAccountErrorCodeInvalidEmail,
				ErrorMessage:     "Please enter a valid email address.",
			}, nil
		case auth.InvalidPhoneNumber:
			return &createPatientAccountOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        createAccountErrorCodeInvalidPhoneNumber,
				ErrorMessage:     "Please enter a valid phone number.",
			}, nil
		}
		return nil, errors.InternalError(ctx, err)
	}
	gqlctx.InPlaceWithAccount(ctx, res.Account)

	// TODO: mraines: Add DOB validation

	// Create entity
	_, err = ram.CreateEntity(ctx, &directory.CreateEntityRequest{
		EntityInfo: entityInfo,
		Type:       directory.EntityType_PATIENT,
		// TODO: Formalize this root identifier somehwere
		ExternalID: res.Account.ID,
		Contacts:   contacts,
	})
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	// TODO: Create saved query
	// TODO: Analytics

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
