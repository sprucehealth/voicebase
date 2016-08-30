package main

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/patientsync"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
)

type integrateAccountInput struct {
	ClientMutationID string `gql:"clientMutationId"`
	EntityID         string `gql:"entityID,nonempty"`
	Code             string `gql:"code,nonempty"`
	Type             string `gql:"type,omitempty"`
}

var integrateAccountInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "IntegratePartnerAccountInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"entityID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"code":             &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"type":             &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

const (
	integrateAccountErrorCodeExpiredCode             = "EXPIRED_CODE"
	integrateAccountErrorCodeIntegrationNotSupported = "INTEGRATION_NOT_SUPPORTED"
	integrateAccountErrorCodeEntityNotFound          = "ENTITY_NOT_FOUND"
	integrateAccountErrorCodeEntityNotSupported      = "ENTITY_NOT_SUPPORTED"
	integrateAccountTypeStripe                       = "stripe"
	integrateAccountTypeHint                         = "hint"
	integrateAccountTypeDrChrono                     = "drchrono"
)

var integrateAccountErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "IntegratePartnerAccountErrorCode",
	Values: graphql.EnumValueConfigMap{
		integrateAccountErrorCodeExpiredCode: &graphql.EnumValueConfig{
			Value:       integrateAccountErrorCodeExpiredCode,
			Description: "The provided code is expired.",
		},
		integrateAccountErrorCodeIntegrationNotSupported: &graphql.EnumValueConfig{
			Value:       integrateAccountErrorCodeIntegrationNotSupported,
			Description: "Integration not yet supported.",
		},
		integrateAccountErrorCodeEntityNotFound: &graphql.EnumValueConfig{
			Value:       integrateAccountErrorCodeEntityNotFound,
			Description: "The referenced entity was not found.",
		},
		integrateAccountErrorCodeEntityNotSupported: &graphql.EnumValueConfig{
			Value:       integrateAccountErrorCodeEntityNotSupported,
			Description: "The referenced entity is not supported for this integration.",
		},
	},
})

type integrateAccountOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var integrateAccountOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "IntegratePartnerAccountPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: integrateAccountErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*integrateAccountOutput)
			return ok
		},
	},
)

var integrateAccountMutation = &graphql.Field{
	Type:        graphql.NewNonNull(integrateAccountOutputType),
	Description: "Integrate partner account enables connection to a third party account via an access code.",
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(integrateAccountInputType)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		var in integrateAccountInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(p.Context, err)
		}

		switch in.Type {
		case integrateAccountTypeStripe:
			return connectVendorStripeAccount(p, in)
		case integrateAccountTypeHint:
			return configureSync(p, in)
		default:
		}
		return &integrateAccountOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          false,
			ErrorCode:        integrateAccountErrorCodeIntegrationNotSupported,
			ErrorMessage:     "This integration is not supported yet",
		}, nil
	}),
}

func connectVendorStripeAccount(p graphql.ResolveParams, in integrateAccountInput) (interface{}, error) {
	ram := raccess.ResourceAccess(p)
	ctx := p.Context

	ent, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: in.EntityID,
		},
	})
	if grpc.Code(err) == codes.NotFound {
		return &integrateAccountOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          false,
			ErrorCode:        integrateAccountErrorCodeEntityNotFound,
			ErrorMessage:     fmt.Sprintf("Entity %s Not Found", in.EntityID),
		}, nil
	} else if err != nil {
		return nil, errors.InternalError(p.Context, err)
	} else if ent.Type != directory.EntityType_ORGANIZATION {
		return &integrateAccountOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          false,
			ErrorCode:        integrateAccountErrorCodeEntityNotSupported,
			ErrorMessage:     fmt.Sprintf("Entity %s is not supported for this integration. Expect an Organization", in.EntityID),
		}, nil
	}

	_, err = ram.ConnectVendorAccount(ctx, &payments.ConnectVendorAccountRequest{
		EntityID: in.EntityID,
		Type:     payments.VENDOR_ACCOUNT_TYPE_STRIPE,
		ConnectVendorAccountOneof: &payments.ConnectVendorAccountRequest_StripeRequest{
			StripeRequest: &payments.StripeAccountConnectRequest{Code: in.Code},
		},
	})
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	return &integrateAccountOutput{
		ClientMutationID: in.ClientMutationID,
		Success:          true,
	}, nil
}

func configureSync(p graphql.ResolveParams, in integrateAccountInput) (interface{}, error) {
	ram := raccess.ResourceAccess(p)
	ctx := p.Context

	_, err := ram.ConfigurePatientSync(ctx, &patientsync.ConfigureSyncRequest{
		OrganizationEntityID: in.EntityID,
		Token:                in.Code,
		Source:               patientsync.SOURCE_HINT,
	})
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	return &integrateAccountOutput{
		ClientMutationID: in.ClientMutationID,
		Success:          true,
	}, nil
}
