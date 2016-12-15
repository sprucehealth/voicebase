package gql

import (
	"context"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/common"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/client"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/models"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var savedThreadQueryType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "SavedThreadQuery",
		Fields: graphql.Fields{
			"id":                   &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"query":                &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"shortTitle":           &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"longTitle":            &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"description":          &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"unread":               &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"total":                &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"ordinal":              &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"notificationsEnabled": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"hidden":               &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"template":             &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"defaultTemplate":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		},
	})

// Create

var createSavedThreadQueryField = &graphql.Field{
	Type: graphql.NewNonNull(createSavedThreadQueryOutputType),
	Args: graphql.FieldConfigArgument{
		common.InputFieldName: &graphql.ArgumentConfig{Type: graphql.NewNonNull(createSavedThreadQueryInputType)},
	},
	Resolve: createSavedThreadQueryResolve,
}

type createSavedThreadQueryInput struct {
	EntityID             string `gql:"entityID,required"`
	Query                string `gql:"query"`
	ShortTitle           string `gql:"shortTitle,required"`
	LongTitle            string `gql:"longTitle,required"`
	Description          string `gql:"description,required"`
	Ordinal              int    `gql:"orginal,required"`
	NotificationsEnabled bool   `gql:"notificationsEnabled,required"`
}

var createSavedThreadQueryInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "CreateSavedThreadQueryInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"entityID":             &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"query":                &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"shortTitle":           &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"longTitle":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"description":          &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"ordinal":              &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Int)},
			"notificationsEnabled": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Boolean)},
		},
	},
)

type createSavedThreadQueryOutput struct {
	Success          bool                     `json:"success"`
	ErrorMessage     string                   `json:"errorMessage,omitempty"`
	SavedThreadQuery *models.SavedThreadQuery `json:"savedThreadQuery"`
}

var createSavedThreadQueryOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CreateSavedThreadQueryOutput",
		Fields: graphql.Fields{
			"success":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorMessage": &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*createSavedThreadQueryOutput)
			return ok
		},
	},
)

func createSavedThreadQueryResolve(p graphql.ResolveParams) (interface{}, error) {
	var in createSavedThreadQueryInput
	if err := gqldecode.Decode(p.Args[common.InputFieldName].(map[string]interface{}), &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, errors.Errorf("%s is invalid: %s", err.Field, err.Reason)
		}
		return nil, errors.Trace(err)
	}

	ctx := p.Context
	directoryCli := client.Directory(p)
	threadingCli := client.Threading(p)

	query, err := threading.ParseQuery(in.Query)
	if err != nil {
		return &createSavedThreadQueryOutput{Success: false, ErrorMessage: "Query is invalid: " + err.Error()}, nil
	}
	if in.Ordinal <= 0 {
		return &createSavedThreadQueryOutput{Success: false, ErrorMessage: "Ordinal must be greater than 0"}, nil
	}

	createReq := &threading.CreateSavedQueryRequest{
		Type:                 threading.SAVED_QUERY_TYPE_NORMAL,
		Hidden:               false,
		EntityID:             in.EntityID,
		Query:                query,
		Ordinal:              int32(in.Ordinal),
		ShortTitle:           in.ShortTitle,
		LongTitle:            in.LongTitle,
		Description:          in.Description,
		NotificationsEnabled: in.NotificationsEnabled,
	}

	// Lookup the entity to make sure it exists and use the type to determine if this should be s template or not
	ent, err := directory.SingleEntity(ctx, directoryCli, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: in.EntityID,
		},
		RootTypes: []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_ORGANIZATION},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	switch ent.Type {
	case directory.EntityType_ORGANIZATION:
		createReq.Template = true
	case directory.EntityType_INTERNAL:
		createReq.Template = false
	}

	createRes, err := threadingCli.CreateSavedQuery(ctx, createReq)
	if err != nil {
		return nil, errors.Trace(err)
	}
	sqs, err := models.TransformSavedThreadQueriesToModel([]*threading.SavedQuery{createRes.SavedQuery})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return sqs[0], nil
}

// Delete

var deleteSavedThreadQueryField = &graphql.Field{
	Type: graphql.NewNonNull(deleteSavedThreadQueryOutputType),
	Args: graphql.FieldConfigArgument{
		common.InputFieldName: &graphql.ArgumentConfig{Type: graphql.NewNonNull(deleteSavedThreadQueryInputType)},
	},
	Resolve: deleteSavedThreadQueryResolve,
}

type deleteSavedThreadQueryInput struct {
	SavedThreadQueryID string `gql:"savedThreadQueryID,required"`
}

var deleteSavedThreadQueryInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "DeleteSavedThreadQueryInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"savedThreadQueryID": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		},
	},
)

type deleteSavedThreadQueryOutput struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

var deleteSavedThreadQueryOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "DeleteSavedThreadQueryOutput",
		Fields: graphql.Fields{
			"success":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorMessage": &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*deleteSavedThreadQueryOutput)
			return ok
		},
	},
)

func deleteSavedThreadQueryResolve(p graphql.ResolveParams) (interface{}, error) {
	var in deleteSavedThreadQueryInput
	if err := gqldecode.Decode(p.Args[common.InputFieldName].(map[string]interface{}), &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, errors.Errorf("%s is invalid: %s", err.Field, err.Reason)
		}
		return nil, errors.Trace(err)
	}

	ctx := p.Context
	threadingCli := client.Threading(p)

	res, err := threadingCli.SavedQuery(ctx, &threading.SavedQueryRequest{SavedQueryID: in.SavedThreadQueryID})
	if grpc.Code(err) == codes.NotFound {
		return &deleteSavedThreadQueryOutput{Success: false, ErrorMessage: "Saved thread query not found"}, nil
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	if res.SavedQuery.Type != threading.SAVED_QUERY_TYPE_NORMAL {
		return &deleteSavedThreadQueryOutput{Success: false, ErrorMessage: "Cannot delete thread query of type " + res.SavedQuery.Type.String()}, nil
	}

	if _, err := threadingCli.DeleteSavedQueries(ctx, &threading.DeleteSavedQueriesRequest{
		SavedQueryIDs: []string{in.SavedThreadQueryID},
	}); err != nil {
		return nil, errors.Trace(err)
	}

	return &deleteSavedThreadQueryOutput{Success: true}, nil
}

// Update

var updateSavedThreadQueryField = &graphql.Field{
	Type: graphql.NewNonNull(updateSavedThreadQueryOutputType),
	Args: graphql.FieldConfigArgument{
		common.InputFieldName: &graphql.ArgumentConfig{Type: graphql.NewNonNull(updateSavedThreadQueryInputType)},
	},
	Resolve: updateSavedThreadQueryResolve,
}

type updateSavedThreadQueryInput struct {
	SavedThreadQueryID string  `gql:"savedThreadQueryID"`
	Query              *string `gql:"query"`
	ShortTitle         *string `gql:"shortTitle"`
	LongTitle          *string `gql:"longTitle"`
	Description        *string `gql:"description"`
	Ordinal            *int    `gql:"orginal"`
}

var updateSavedThreadQueryInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "UpdateSavedThreadQueryInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"savedThreadQueryID": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"query":              &graphql.InputObjectFieldConfig{Type: graphql.String},
			"shortTitle":         &graphql.InputObjectFieldConfig{Type: graphql.String},
			"longTitle":          &graphql.InputObjectFieldConfig{Type: graphql.String},
			"description":        &graphql.InputObjectFieldConfig{Type: graphql.String},
			"ordinal":            &graphql.InputObjectFieldConfig{Type: graphql.Int},
		},
	},
)

type updateSavedThreadQueryOutput struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

var updateSavedThreadQueryOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "UpdateSavedThreadQueryOutput",
		Fields: graphql.Fields{
			"success":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorMessage": &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*updateSavedThreadQueryOutput)
			return ok
		},
	},
)

func updateSavedThreadQueryResolve(p graphql.ResolveParams) (interface{}, error) {
	var in updateSavedThreadQueryInput
	if err := gqldecode.Decode(p.Args[common.InputFieldName].(map[string]interface{}), &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, errors.Errorf("%s is invalid: %s", err.Field, err.Reason)
		}
		return nil, errors.Trace(err)
	}

	ctx := p.Context
	threadingCli := client.Threading(p)

	res, err := threadingCli.SavedQuery(ctx, &threading.SavedQueryRequest{SavedQueryID: in.SavedThreadQueryID})
	if grpc.Code(err) == codes.NotFound {
		return &updateSavedThreadQueryOutput{Success: false, ErrorMessage: "Saved thread query not found"}, nil
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	if res.SavedQuery.Type != threading.SAVED_QUERY_TYPE_NORMAL {
		return &updateSavedThreadQueryOutput{Success: false, ErrorMessage: "Cannot modify thread query of type " + res.SavedQuery.Type.String()}, nil
	}

	updateReq := &threading.UpdateSavedQueryRequest{
		SavedQueryID: in.SavedThreadQueryID,
	}
	if in.Query != nil {
		query, err := threading.ParseQuery(*in.Query)
		if err != nil {
			return &updateSavedThreadQueryOutput{Success: false, ErrorMessage: "Query is invalid: " + err.Error()}, nil
		}
		updateReq.Query = query
	}
	if in.Ordinal != nil {
		if *in.Ordinal <= 0 {
			return &updateSavedThreadQueryOutput{Success: false, ErrorMessage: "Ordinal must be greater than 0"}, nil
		}
		updateReq.Ordinal = int32(*in.Ordinal)
	}
	if in.ShortTitle != nil {
		updateReq.ShortTitle = strings.TrimSpace(*in.ShortTitle)
	}
	if in.LongTitle != nil {
		updateReq.LongTitle = strings.TrimSpace(*in.LongTitle)
	}
	if in.Description != nil {
		updateReq.Description = strings.TrimSpace(*in.Description)
	}

	if _, err := threadingCli.UpdateSavedQuery(ctx, updateReq); err != nil {
		return nil, errors.Trace(err)
	}

	return &updateSavedThreadQueryOutput{Success: true}, nil
}

// createDefaultSavedThreadQueryTemplates mutation

var createDefaultSavedThreadQueryTemplatesField = &graphql.Field{
	Description: "Actually create the default set of saved query templates for an organization instead of using the global defaults.",
	Type:        graphql.NewNonNull(createDefaultSavedThreadQueryTemplatesOutputType),
	Args: graphql.FieldConfigArgument{
		common.InputFieldName: &graphql.ArgumentConfig{Type: graphql.NewNonNull(createDefaultSavedThreadQueryTemplatesInputType)},
	},
	Resolve: createDefaultSavedThreadQueryTemplatesResolve,
}

type createDefaultSavedThreadQueryTemplatesInput struct {
	EntityID string `gql:"entityID,required"`
}

var createDefaultSavedThreadQueryTemplatesInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "CreateDefaultSavedThreadQueryTemplatesInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"entityID": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		},
	},
)

type createDefaultSavedThreadQueryTemplatesOutput struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

var createDefaultSavedThreadQueryTemplatesOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CreateDefaultSavedThreadQueryTemplatesOutput",
		Fields: graphql.Fields{
			"success":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorMessage": &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*createDefaultSavedThreadQueryTemplatesOutput)
			return ok
		},
	},
)

func createDefaultSavedThreadQueryTemplatesResolve(p graphql.ResolveParams) (interface{}, error) {
	var in createDefaultSavedThreadQueryTemplatesInput
	if err := gqldecode.Decode(p.Args[common.InputFieldName].(map[string]interface{}), &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, errors.Errorf("%s is invalid: %s", err.Field, err.Reason)
		}
		return nil, errors.Trace(err)
	}

	ctx := p.Context
	directoryCli := client.Directory(p)
	threadingCli := client.Threading(p)

	// Make sure to only work with organizations as they're the only ones with template saved queries
	_, err := directory.SingleEntity(ctx, directoryCli, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: in.EntityID,
		},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Get the defaults
	res, err := threadingCli.SavedQueryTemplates(ctx, &threading.SavedQueryTemplatesRequest{EntityID: in.EntityID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Make sure they're actually default templates rather than concrete.
	for _, q := range res.SavedQueries {
		if !q.Template || !q.DefaultTemplate {
			return &createDefaultSavedThreadQueryTemplatesOutput{
				Success:      false,
				ErrorMessage: "Saved queries contains a non-template or non-default template",
			}, nil
		}
	}

	par := conc.NewParallel()
	for _, sq := range res.SavedQueries {
		req := &threading.CreateSavedQueryRequest{
			EntityID:             in.EntityID,
			ShortTitle:           sq.ShortTitle,
			LongTitle:            sq.LongTitle,
			Description:          sq.Description,
			Hidden:               sq.Hidden,
			Type:                 sq.Type,
			Ordinal:              sq.Ordinal,
			NotificationsEnabled: sq.NotificationsEnabled,
			Query:                sq.Query,
			Template:             sq.Template,
		}
		par.Go(func() error {
			_, err := threadingCli.CreateSavedQuery(ctx, req)
			return errors.Trace(err)
		})
	}
	if err := par.Wait(); err != nil {
		return nil, errors.Trace(err)
	}

	return &createDefaultSavedThreadQueryTemplatesOutput{Success: true}, nil
}

// Query

func getSavedThreadQueriesForEntity(ctx context.Context, directoryCli directory.DirectoryClient, threadingCli threading.ThreadsClient, entityID string) ([]*models.SavedThreadQuery, error) {
	ent, err := directory.SingleEntity(ctx, directoryCli, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entityID,
		},
		RootTypes: []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_ORGANIZATION},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	var savedQueries []*threading.SavedQuery
	switch ent.Type {
	case directory.EntityType_ORGANIZATION:
		res, err := threadingCli.SavedQueryTemplates(ctx, &threading.SavedQueryTemplatesRequest{
			EntityID: entityID,
		})
		if err != nil {
			return nil, errors.Trace(err)
		}
		savedQueries = res.SavedQueries
	case directory.EntityType_INTERNAL:
		res, err := threadingCli.SavedQueries(ctx, &threading.SavedQueriesRequest{
			EntityID: entityID,
		})
		if err != nil {
			return nil, errors.Trace(err)
		}
		savedQueries = res.SavedQueries
	}
	sqs, err := models.TransformSavedThreadQueriesToModel(savedQueries)
	return sqs, errors.Trace(err)
}
