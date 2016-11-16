package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
)

// deleteScheduledMessage
type deleteScheduledMessageInput struct {
	ClientMutationID string `gql:"clientMutationId"`
	ID               string `gql:"id,nonempty"`
}

var deleteScheduledMessageInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "DeleteScheduledMessageInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"id":               &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		},
	},
)

const (
	deleteScheduledMessageError = "DELETE_SCHEDULED_MESSAGE_ERROR"
)

var deleteScheduledMessageErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "DeleteScheduledMessageErrorCode",
	Values: graphql.EnumValueConfigMap{
		deleteScheduledMessageError: &graphql.EnumValueConfig{
			Value:       deleteScheduledMessageError,
			Description: "There was an error deleting the scheduled message",
		},
	},
})

type deleteScheduledMessageOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var deleteScheduledMessageOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "DeleteScheduledMessagePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: deleteScheduledMessageErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*deleteScheduledMessageOutput)
			return ok
		},
	},
)

var deleteScheduledMessageMutation = &graphql.Field{
	Type: graphql.NewNonNull(deleteScheduledMessageOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(deleteScheduledMessageInputType)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		var in deleteScheduledMessageInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(p.Context, err)
		}
		return deleteScheduledMessage(p.Context, raccess.ResourceAccess(p), in)
	}),
}

func deleteScheduledMessage(ctx context.Context, ram raccess.ResourceAccessor, in deleteScheduledMessageInput) (interface{}, error) {
	if _, err := ram.DeleteScheduledMessage(ctx, &threading.DeleteScheduledMessageRequest{
		ScheduledMessageID: in.ID,
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &deleteScheduledMessageOutput{
		ClientMutationID: in.ClientMutationID,
		Success:          true,
	}, nil
}

// scheduleMessage
type scheduleMessageInput struct {
	ClientMutationID      string        `gql:"clientMutationId"`
	Message               *messageInput `gql:"message"`
	ActingEntityID        string        `gql:"actingEntityID"`
	ScheduledForTimestamp int           `gql:"scheduledForTimestamp"`
	ThreadID              string        `gql:"threadID"`
}

var scheduleMessageInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "ScheduleMessageInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId":      newClientMutationIDInputField(),
			"message":               &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(messageInputType)},
			"actingEntityID":        &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"threadID":              &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"scheduledForTimestamp": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Int)},
		},
	},
)

const (
	scheduleMessageErrorScheduledMessageError = "SCHEDULED_MESSAGE_ERROR"
)

var scheduleMessageErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "ScheduleMessageErrorCode",
	Values: graphql.EnumValueConfigMap{
		scheduleMessageErrorScheduledMessageError: &graphql.EnumValueConfig{
			Value:       scheduleMessageErrorScheduledMessageError,
			Description: "There was an error creating the scheduled message",
		},
	},
})

type scheduleMessageOutput struct {
	ClientMutationID  string                     `json:"clientMutationId,omitempty"`
	Success           bool                       `json:"success"`
	ErrorCode         string                     `json:"errorCode,omitempty"`
	ErrorMessage      string                     `json:"errorMessage,omitempty"`
	ScheduledMessages []*models.ScheduledMessage `json:"scheduledMessages"`
}

var scheduleMessageOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ScheduleMessagePayload",
		Fields: graphql.Fields{
			"clientMutationId":  newClientMutationIDOutputField(),
			"success":           &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":         &graphql.Field{Type: scheduleMessageErrorCodeEnum},
			"errorMessage":      &graphql.Field{Type: graphql.String},
			"scheduledMessages": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(scheduledMessageType))},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*scheduleMessageOutput)
			return ok
		},
	},
)

var scheduleMessageMutation = &graphql.Field{
	Type: graphql.NewNonNull(scheduleMessageOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(scheduleMessageInputType)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		var in scheduleMessageInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(p.Context, err)
		}
		return scheduleMessage(p.Context, serviceFromParams(p), raccess.ResourceAccess(p), in)
	}),
}

func scheduleMessage(ctx context.Context, svc *service, ram raccess.ResourceAccessor, in scheduleMessageInput) (interface{}, error) {
	// TODO: Move this up into a validation chain eventually
	ent, err := ram.AssertIsEntity(ctx, in.ActingEntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	thread, err := ram.Thread(ctx, in.ThreadID, "")
	if err != nil {
		return nil, errors.Trace(err)
	}

	// don't allow scheduling of message in the past
	if time.Now().Unix() > int64(in.ScheduledForTimestamp) {
		return &scheduleMessageOutput{
			Success:      false,
			ErrorCode:    scheduleMessageErrorScheduledMessageError,
			ErrorMessage: "Cannot schedule a message in the past",
		}, nil
	}

	var primaryEntity *directory.Entity
	if thread.PrimaryEntityID != "" {
		primaryEntity, err = raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: thread.PrimaryEntityID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth:             0,
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
			},
			Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
	}

	// TODO: if there are references in the message, we are currently resolving them at the time of the scheduling
	// of the message rather than at the time of posting. This means there is a possibility for someone to be @paged
	// that is later removed from the team. While possible, is an edge case but worth fixing in the future.
	msg, _, err := transformRequestToMessagePost(ctx, svc, ram, in.Message, thread, ent, primaryEntity)
	if e, ok := err.(errInvalidAttachment); ok {
		return &scheduleMessageOutput{
			Success:      false,
			ErrorCode:    scheduleMessageErrorScheduledMessageError,
			ErrorMessage: string(e),
		}, nil
	} else if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	_, err = ram.CreateScheduledMessage(ctx, &threading.CreateScheduledMessageRequest{
		ThreadID:      in.ThreadID,
		ActorEntityID: ent.ID,
		ScheduledFor:  uint64(in.ScheduledForTimestamp),
		Content: &threading.CreateScheduledMessageRequest_Message{
			Message: msg,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	scheduledMessages, err := getScheduledMessages(ctx, ram, thread.ID, thread.OrganizationID, svc.webDomain, svc.mediaAPIDomain)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &scheduleMessageOutput{
		ClientMutationID:  in.ClientMutationID,
		Success:           true,
		ScheduledMessages: scheduledMessages,
	}, nil
}
