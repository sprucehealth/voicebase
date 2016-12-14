package gql

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/common"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/client"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// triggeredMessageType is a type representing an triggered message
var triggeredMessageType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "TriggeredMessage",
		Fields: graphql.Fields{
			"id": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"organizationEntityID": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"organization":         &graphql.Field{Type: graphql.NewNonNull(entityType), Resolve: triggeredMessageOrganizationResolve},
			"actorEntityID":        &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"actor":                &graphql.Field{Type: graphql.NewNonNull(entityType), Resolve: triggeredMessageActorResolve},
			"key":                  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"subkey":               &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"enabled":              &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"items":                &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(triggeredMessageItemType))},
			"created":              &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		},
	})

type triggeredMessageInput struct {
	EntityID string `gql:"entityID,nonempty"`
	Key      string `gql:"key,nonempty"`
	Subkey   string `gql:"subkey,nonempty"`
}

var triggeredMessageInputType = graphql.FieldConfigArgument{
	"entityID": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
	"key":      &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
	"subkey":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
}

// triggeredMessageField represents a graphql field for Querying a triggered message object
var triggeredMessageField = &graphql.Field{
	Type:    triggeredMessageType,
	Args:    triggeredMessageInputType,
	Resolve: triggeredMessageResolve,
}

func triggeredMessageResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	var in triggeredMessageInput
	if err := gqldecode.Decode(p.Args, &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, errors.Errorf("%s is invalid: %s", err.Field, err.Reason)
		}
		return nil, errors.Trace(err)
	}
	golog.ContextLogger(ctx).Debugf("Looking up Triggered Message with args %+v", in)

	key, ok := threading.TriggeredMessageKey_Key_value[in.Key]
	if !ok {
		return nil, errors.Errorf("%s is an invalid TriggeredMessage key", in.Key)
	}
	tkey := threading.TriggeredMessageKey_Key(key)

	return getTriggeredMessage(ctx, client.Threading(p), in.EntityID, tkey, in.Subkey)
}

func triggeredMessageOrganizationResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	tm := p.Source.(*models.TriggeredMessage)
	return getEntity(ctx, client.Directory(p), tm.OrganizationEntityID)
}

func triggeredMessageActorResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	tm := p.Source.(*models.TriggeredMessage)
	return getEntity(ctx, client.Directory(p), tm.ActorEntityID)
}

func getTriggeredMessage(ctx context.Context, threadingCli threading.ThreadsClient, entityID string, key threading.TriggeredMessageKey_Key, subkey string) (*models.TriggeredMessage, error) {
	resp, err := threadingCli.TriggeredMessages(ctx, &threading.TriggeredMessagesRequest{
		OrganizationEntityID: entityID,
		LookupKey: &threading.TriggeredMessagesRequest_Key{
			Key: &threading.TriggeredMessageKey{
				Key:    key,
				Subkey: subkey,
			},
		},
	})
	if grpc.Code(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	tms := models.TransformTriggeredMessagesToModel(resp.TriggeredMessages)
	if len(tms) > 1 {
		return nil, errors.Errorf("Expected only 1 Triggered Message mapped to %s %s but got %d", key, subkey, len(tms))
	}
	if len(tms) == 1 {
		return tms[0], nil
	}
	return nil, nil
}

// createTriggeredMessage
type createTriggeredMessageInput struct {
	OrganizationEntityID string   `gql:"organizationEntityID,nonempty"`
	ActorEntityID        string   `gql:"actorEntityID,nonempty"`
	SavedMessageIDs      []string `gql:"savedMessageIDs,nonempty"`
	Key                  string   `gql:"key,nonempty"`
	Subkey               string   `gql:"subkey,nonempty"`
	Enabled              bool     `gql:"enabled,nonempty"`
}

var createTriggeredMessageInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "CreateTriggeredMessageInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"organizationEntityID": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"actorEntityID":        &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"savedMessageIDs":      &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
			"key":                  &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"subkey":               &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"enabled":              &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Boolean)},
		},
	},
)

type createTriggeredMessageOutput struct {
	Success          bool                     `json:"success"`
	ErrorMessage     string                   `json:"errorMessage,omitempty"`
	TriggeredMessage *models.TriggeredMessage `json:"triggeredMessage"`
}

var createTriggeredMessageOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CreateTriggeredMessagePayload",
		Fields: graphql.Fields{
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"triggeredMessage": &graphql.Field{Type: triggeredMessageType},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*createTriggeredMessageOutput)
			return ok
		},
	},
)

var createTriggeredMessageField = &graphql.Field{
	Type: graphql.NewNonNull(createTriggeredMessageOutputType),
	Args: graphql.FieldConfigArgument{
		common.InputFieldName: &graphql.ArgumentConfig{Type: graphql.NewNonNull(createTriggeredMessageInputType)},
	},
	Resolve: createTriggeredMessageResolve,
}

func createTriggeredMessageResolve(p graphql.ResolveParams) (interface{}, error) {
	var in createTriggeredMessageInput
	if err := gqldecode.Decode(p.Args[common.InputFieldName].(map[string]interface{}), &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, errors.Errorf("%s is invalid: %s", err.Field, err.Reason)
		}
		return nil, errors.Trace(err)
	}

	golog.ContextLogger(p.Context).Debugf("Creating Triggered Message for %+v", in)
	triggeredMessage, err := createTriggeredMessage(p.Context, client.Threading(p), &in)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &createTriggeredMessageOutput{
		Success:          true,
		TriggeredMessage: triggeredMessage,
	}, nil
}

func createTriggeredMessage(ctx context.Context, threadingCli threading.ThreadsClient, in *createTriggeredMessageInput) (*models.TriggeredMessage, error) {
	key, ok := threading.TriggeredMessageKey_Key_value[in.Key]
	if !ok {
		return nil, errors.Errorf("%s is an invalid TriggeredMessage key", in.Key)
	}
	tkey := threading.TriggeredMessageKey_Key(key)

	smResp, err := threadingCli.SavedMessages(ctx, &threading.SavedMessagesRequest{
		By: &threading.SavedMessagesRequest_IDs{
			IDs: &threading.IDList{
				IDs: in.SavedMessageIDs,
			},
		},
	})
	if err != nil {
		return nil, errors.Errorf("Error while looking up saved messages for %+v: %s", in, err)
	}

	postMessages := make([]*threading.MessagePost, len(smResp.SavedMessages))
	for i, sm := range smResp.SavedMessages {
		switch c := sm.Content.(type) {
		case *threading.SavedMessage_Message:
			m := c.Message
			postMessages[i] = &threading.MessagePost{
				Internal:    sm.Internal,
				Text:        m.Text,
				Attachments: m.Attachments,
				Title:       m.Title,
				Summary:     m.Summary,
			}
		default:
			return nil, errors.Errorf("Unable to create post for unknown content %+v in saved message %s", sm, sm.ID)
		}
	}

	createResp, err := threadingCli.CreateTriggeredMessage(ctx, &threading.CreateTriggeredMessageRequest{
		ActorEntityID:        in.ActorEntityID,
		OrganizationEntityID: in.OrganizationEntityID,
		Key: &threading.TriggeredMessageKey{
			Key:    tkey,
			Subkey: in.Subkey,
		},
		Messages: postMessages,
		Enabled:  in.Enabled,
	})
	if err != nil {
		return nil, errors.Errorf("Error while creating triggered message messages for %+v: %s", in, err)
	}
	return models.TransformTriggeredMessageToModel(createResp.TriggeredMessage), nil
}

// modifyTriggeredMessage
type modifyTriggeredMessageInput struct {
	TriggeredMessageID string `gql:"triggeredMessageID,nonempty"`
	Enabled            bool   `gql:"enabled,nonempty"`
}

var modifyTriggeredMessageInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "ModifyTriggeredMessageInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"triggeredMessageID": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"enabled":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Boolean)},
		},
	},
)

type modifyTriggeredMessageOutput struct {
	Success          bool                     `json:"success"`
	ErrorMessage     string                   `json:"errorMessage,omitempty"`
	TriggeredMessage *models.TriggeredMessage `json:"triggeredMessage"`
}

var modifyTriggeredMessageOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ModifyTriggeredMessagePayload",
		Fields: graphql.Fields{
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"triggeredMessage": &graphql.Field{Type: triggeredMessageType},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*modifyTriggeredMessageOutput)
			return ok
		},
	},
)

var modifyTriggeredMessageField = &graphql.Field{
	Type: graphql.NewNonNull(modifyTriggeredMessageOutputType),
	Args: graphql.FieldConfigArgument{
		common.InputFieldName: &graphql.ArgumentConfig{Type: graphql.NewNonNull(modifyTriggeredMessageInputType)},
	},
	Resolve: modifyTriggeredMessageResolve,
}

func modifyTriggeredMessageResolve(p graphql.ResolveParams) (interface{}, error) {
	var in modifyTriggeredMessageInput
	if err := gqldecode.Decode(p.Args[common.InputFieldName].(map[string]interface{}), &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, errors.Errorf("%s is invalid: %s", err.Field, err.Reason)
		}
		return nil, errors.Trace(err)
	}

	golog.ContextLogger(p.Context).Debugf("Modifying Triggered Message for %+v", in)
	triggeredMessage, err := modifyTriggeredMessage(p.Context, client.Threading(p), &in)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &modifyTriggeredMessageOutput{
		Success:          true,
		TriggeredMessage: triggeredMessage,
	}, nil
}

func modifyTriggeredMessage(ctx context.Context, threadingCli threading.ThreadsClient, in *modifyTriggeredMessageInput) (*models.TriggeredMessage, error) {
	resp, err := threadingCli.UpdateTriggeredMessage(ctx, &threading.UpdateTriggeredMessageRequest{
		TriggeredMessageID: in.TriggeredMessageID,
		Enabled:            in.Enabled,
		UpdateEnabled:      true,
	})
	if err != nil {
		return nil, errors.Errorf("Error while creating triggered message messages for %+v: %s", in, err)
	}
	return models.TransformTriggeredMessageToModel(resp.TriggeredMessage), nil
}
