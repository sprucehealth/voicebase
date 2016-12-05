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
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/graphql"
)

const (
	contactEnumPhone = "PHONE"
	contactEnumEmail = "EMAIL"
)

// contactArgumentsConfig represents the config for arguments referencing a contact
var contactArgumentsConfig = graphql.FieldConfigArgument{
	"id": &graphql.ArgumentConfig{Type: graphql.String},
}

// contactArguments represents arguments for referencing a contact
type contactArguments struct {
	ID string `json:"id"`
}

// parsecontactArguments parses the contact arguments out of requests params
func parseContactArguments(args map[string]interface{}) *contactArguments {
	cArgs := &contactArguments{}
	if args != nil {
		if iid, ok := args["id"]; ok {
			if id, ok := iid.(string); ok {
				cArgs.ID = id
			}
		}
	}
	return cArgs
}

// contactField returns is a graphql field for Querying an contact object
var contactField = &graphql.Field{
	Type:    contactType,
	Args:    contactArgumentsConfig,
	Resolve: contactResolve,
}

func contactResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	args := parseContactArguments(p.Args)
	golog.ContextLogger(ctx).Debugf("Resolving contact with args %+v", args)
	if args.ID == "" {
		return nil, nil
	}
	return getContact(ctx, client.Directory(p), args.ID)
}

// contactEnumType represents the possible enum values mapped to contact types
var contactEnumType = graphql.NewEnum(
	graphql.EnumConfig{
		Name: "ContactType",
		Values: graphql.EnumValueConfigMap{
			contactEnumPhone: &graphql.EnumValueConfig{
				Value: contactEnumPhone,
			},
			contactEnumEmail: &graphql.EnumValueConfig{
				Value: contactEnumEmail,
			},
		},
	},
)

// newContactType returns a type object representing an contact contact
var contactType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ContactInfo",
		Fields: graphql.Fields{
			"id":          &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"type":        &graphql.Field{Type: graphql.NewNonNull(contactEnumType)},
			"value":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"provisioned": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"label":       &graphql.Field{Type: graphql.String},
		},
	})

func getContact(ctx context.Context, dirCli directory.DirectoryClient, id string) (*models.Contact, error) {
	resp, err := dirCli.Contact(ctx, &directory.ContactRequest{
		ContactID: id,
	})
	if err != nil {
		golog.ContextLogger(ctx).Warningf("Error while fetching contact %s", err)
		return nil, errors.Trace(err)
	}
	return models.TransformContactToModel(resp.Contact), nil
}

type provisionNumberInput struct {
	UUID           string `gql:"uuid"`
	OrganizationID string `gql:"organizationID"`
	AreaCode       string `gql:"areaCode"`
	PhoneNumber    string `gql:"phoneNumber"`
}

var provisionNumberInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "ProvisionNumberInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"uuid":           &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"organizationID": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"areaCode":       &graphql.InputObjectFieldConfig{Type: graphql.String},
			"phoneNumber":    &graphql.InputObjectFieldConfig{Type: graphql.String},
		},
	},
)

type provisionNumberOutput struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	PhoneNumber  string `json:"phoneNumber"`
}

var provisionNumberOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ProvisionNumberPayload",
		Fields: graphql.Fields{
			"success":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorMessage": &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*provisionNumberOutput)
			return ok
		},
	},
)

var provisionNumberField = &graphql.Field{
	Type: graphql.NewNonNull(provisionNumberOutputType),
	Args: graphql.FieldConfigArgument{
		common.InputFieldName: &graphql.ArgumentConfig{Type: graphql.NewNonNull(provisionNumberInputType)},
	},
	Resolve: provisionNumberResolve,
}

func provisionNumberResolve(p graphql.ResolveParams) (interface{}, error) {
	var in provisionNumberInput
	if err := gqldecode.Decode(p.Args[common.InputFieldName].(map[string]interface{}), &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, errors.Errorf("%s is invalid: %s", err.Field, err.Reason)
		}
		return nil, errors.Trace(err)
	}
	return provisionNumber(p.Context, client.Directory(p), client.ExComms(p), &in)
}

func provisionNumber(ctx context.Context, dirCli directory.DirectoryClient, excommsCli excomms.ExCommsClient, in *provisionNumberInput) (*provisionNumberOutput, error) {
	if (in.AreaCode == "") == (in.PhoneNumber == "") {
		return &provisionNumberOutput{
			Success:      false,
			ErrorMessage: "One of area code or phone number must be set but not both.",
		}, nil
	}

	in.UUID = strings.TrimSpace(in.UUID)
	if in.UUID != "" {
		// Prefix the UUID to be safe
		in.UUID = in.OrganizationID + ":" + in.UUID
	}

	// Verify that organization exists
	org, err := directory.SingleEntity(ctx, dirCli, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: in.OrganizationID,
		},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	})
	if errors.Cause(err) == directory.ErrEntityNotFound {
		return &provisionNumberOutput{
			Success:      false,
			ErrorMessage: "Organization does not exist.",
		}, nil
	}
	if err != nil {
		return nil, errors.Trace(err)
	}

	var req *excomms.ProvisionPhoneNumberRequest
	if in.AreaCode != "" {
		req = &excomms.ProvisionPhoneNumberRequest{
			UUID:         in.UUID,
			ProvisionFor: org.ID,
			Number: &excomms.ProvisionPhoneNumberRequest_AreaCode{
				AreaCode: in.AreaCode,
			},
		}
	} else {
		req = &excomms.ProvisionPhoneNumberRequest{
			UUID:         in.UUID,
			ProvisionFor: org.ID,
			Number: &excomms.ProvisionPhoneNumberRequest_PhoneNumber{
				PhoneNumber: in.PhoneNumber,
			},
		}
	}
	res, err := excommsCli.ProvisionPhoneNumber(ctx, req)
	if err != nil {
		return nil, errors.Trace(err)
	}

	_, err = dirCli.CreateContact(ctx, &directory.CreateContactRequest{
		EntityID: org.ID,
		Contact: &directory.Contact{
			Provisioned: true,
			ContactType: directory.ContactType_PHONE,
			Value:       res.PhoneNumber,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &provisionNumberOutput{
		Success:     true,
		PhoneNumber: res.PhoneNumber,
	}, nil
}
