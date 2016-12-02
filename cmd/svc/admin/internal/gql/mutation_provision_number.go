package gql

import (
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/common"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/client"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/graphql"
)

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
		Name: "ModifySettingPayload",
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

	ctx := p.Context
	exc := client.ExComms(p)
	dirc := client.Directory(p)

	// Verify that organization exists
	org, err := directory.SingleEntity(ctx, dirc, &directory.LookupEntitiesRequest{
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
	res, err := exc.ProvisionPhoneNumber(ctx, req)
	if err != nil {
		return nil, errors.Trace(err)
	}

	_, err = dirc.CreateContact(ctx, &directory.CreateContactRequest{
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
