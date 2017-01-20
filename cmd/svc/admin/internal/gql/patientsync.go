package gql

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/common"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/client"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/patientsync"
	"github.com/sprucehealth/graphql"
)

var tagMappingType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "TagMappingItem",
		Fields: graphql.Fields{
			"tag":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"providerID": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	})

var patientSyncConfigurationType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "PatientSyncConfiguration",
		Fields: graphql.Fields{
			"source":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"connected":   &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"threadType":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"tagMappings": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(tagMappingType))},
		},
	})

func getEntityPatientSyncConfigurations(
	ctx context.Context,
	patientSyncClient patientsync.PatientSyncClient,
	source patientsync.Source,
	entityID string) (*models.PatientSyncConfiguration, error) {
	resp, err := patientSyncClient.LookupSyncConfiguration(ctx, &patientsync.LookupSyncConfigurationRequest{
		Source:               source,
		OrganizationEntityID: entityID,
	})
	if grpc.Code(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return models.TransformPatientSyncConfigurationToModel(resp.Config), nil
}

type tagMappingItem struct {
	Tag        string `gql:"tag"`
	ProviderID string `gql:"providerID"`
}

type updateSyncConfigurationInput struct {
	EntityID    string            `gql:"entityID"`
	TagMappings []*tagMappingItem `gql:"tagMappings"`
}

var tagMappingInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "TagMappingInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"providerID": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"tag":        &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

var updateSyncConfigurationInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "UpdateSyncConfigurationInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"entityID":    &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"tagMappings": &graphql.InputObjectFieldConfig{Type: graphql.NewList(tagMappingInputType)},
		},
	},
)

type updateSyncConfigurationOutput struct {
	Success       bool                             `json:"success"`
	ErrorMessage  string                           `json:"errorMessage,omitempty"`
	Configuration *models.PatientSyncConfiguration `json:"configuration"`
}

var updateSyncConfigurationOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "UpdateSyncConfigurationPayload",
		Fields: graphql.Fields{
			"success":       &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorMessage":  &graphql.Field{Type: graphql.String},
			"configuration": &graphql.Field{Type: patientSyncConfigurationType},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*updateSyncConfigurationOutput)
			return ok
		},
	},
)

var updateSyncConfigurationField = &graphql.Field{
	Type: graphql.NewNonNull(updateSyncConfigurationOutputType),
	Args: graphql.FieldConfigArgument{
		common.InputFieldName: &graphql.ArgumentConfig{Type: graphql.NewNonNull(updateSyncConfigurationInputType)},
	},
	Resolve: updateSyncConfigurationResolve,
}

func updateSyncConfigurationResolve(p graphql.ResolveParams) (interface{}, error) {
	var in updateSyncConfigurationInput
	if err := gqldecode.Decode(p.Args[common.InputFieldName].(map[string]interface{}), &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, errors.Errorf("%s is invalid: %s", err.Field, err.Reason)
		}
		return nil, errors.Trace(err)
	}

	mappings := make([]*patientsync.TagMappingItem, len(in.TagMappings))
	for i, item := range in.TagMappings {
		mappings[i] = &patientsync.TagMappingItem{
			Tag: item.Tag,
			Key: &patientsync.TagMappingItem_ProviderID{ // TODO: Make this generic when we support more mapping key types
				ProviderID: item.ProviderID,
			},
		}
	}

	golog.ContextLogger(p.Context).Debugf("Updating Sync Configuration Account - %s", in.EntityID)

	res, err := client.PatientSync(p).UpdateSyncConfiguration(p.Context, &patientsync.UpdateSyncConfigurationRequest{
		OrganizationEntityID: in.EntityID,
		Source:               patientsync.SOURCE_HINT, // TODO: make this configurable
		TagMappings:          mappings,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &updateSyncConfigurationOutput{
		Success:       true,
		Configuration: models.TransformPatientSyncConfigurationToModel(res.Config),
	}, nil
}
