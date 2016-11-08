package main

import (
	"context"
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/gqldecode"
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
	// Parse text and render as plain text so we can build a summary.
	textBML, err := bml.Parse(in.Message.Text)
	if e, ok := err.(bml.ErrParseFailure); ok {
		return nil, errors.Errorf("failed to parse text at pos %d: %s", e.Offset, e.Reason)
	} else if err != nil {
		return nil, errors.Trace(errors.New("text is not valid markup"))
	}
	plainText, err := textBML.PlainText()
	if err != nil {
		// Shouldn't fail here since the parsing should have done validation
		return nil, errors.Trace(err)
	}
	msg := &threading.MessagePost{
		Internal: in.Message.Internal,
		Summary:  plainText,
	}
	msg.Text, err = textBML.Format()
	if err != nil {
		// Shouldn't fail here since the parsing should have done validation
		return nil, errors.Trace(err)
	}

	attachments, _, err := processIncomingAttachments(ctx, ram, svc, ent, thread.OrganizationID, in.Message.Attachments, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	msg.Attachments = attachments

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
