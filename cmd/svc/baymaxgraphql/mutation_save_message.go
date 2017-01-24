package main

import (
	"fmt"

	segment "github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/analytics"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
)

type saveMessageInput struct {
	ClientMutationID string        `gql:"clientMutationId"`
	MessageID        string        `gql:"messageID"`
	Shared           bool          `gql:"shared"`
	Title            string        `gql:"title"`
	OrganizationID   string        `gql:"organizationID"`
	Message          *messageInput `gql:"message"`
}

var saveMessageInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "SaveMessageInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"messageID":        &graphql.InputObjectFieldConfig{Type: graphql.ID},
			"shared":           &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Boolean)},
			"title":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"organizationID":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"message":          &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(messageInputType)},
		},
	},
)

const saveMessageErrorCode = "SaveMessageErrorCode"

var saveMessageErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "SaveMessageErrorCode",
	Values: graphql.EnumValueConfigMap{
		saveMessageErrorCode: &graphql.EnumValueConfig{
			Value:       saveMessageErrorCode,
			Description: "Placeholder",
		},
	},
})

type saveMessageOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var saveMessageOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "SaveMessagePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: saveMessageErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*saveMessageOutput)
			return ok
		},
	},
)

var saveMessageMutation = &graphql.Field{
	Type: graphql.NewNonNull(saveMessageOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(saveMessageInputType)},
	},
	Resolve: apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		svc := serviceFromParams(p)
		ram := raccess.ResourceAccess(p)
		acc := gqlctx.Account(ctx)

		var in saveMessageInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(p.Context, err)
		}

		ent, err := raccess.EntityInOrgForAccountID(ctx, ram, &directory.LookupEntitiesRequest{
			Key: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: acc.ID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth:             0,
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
			},
			Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes: []directory.EntityType{directory.EntityType_INTERNAL},
		}, in.OrganizationID)
		if err == raccess.ErrNotFound {
			return nil, errors.ErrNotAuthorized(ctx, in.OrganizationID)
		} else if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		var oldMessage *threading.SavedMessage
		if in.MessageID != "" {
			res, err := ram.SavedMessages(ctx, &threading.SavedMessagesRequest{
				By: &threading.SavedMessagesRequest_IDs{
					IDs: &threading.IDList{
						IDs: []string{in.MessageID},
					},
				},
			})
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			if len(res.SavedMessages) == 0 {
				return nil, errors.ErrNotFound(ctx, in.MessageID)
			}
			oldMessage = res.SavedMessages[0]
			if oldMessage.OwnerEntityID != ent.ID && oldMessage.OwnerEntityID != in.OrganizationID {
				return nil, errors.ErrNotAuthorized(ctx, oldMessage.ID)
			}
		}

		// Parse text and render as plain text so we can build a summary.
		textBML, err := bml.Parse(in.Message.Text)
		if e, ok := err.(bml.ErrParseFailure); ok {
			return nil, fmt.Errorf("failed to parse text at pos %d: %s", e.Offset, e.Reason)
		} else if err != nil {
			return nil, errors.New("text is not valid markup")
		}
		plainText, err := textBML.PlainText()
		if err != nil {
			// Shouldn't fail here since the parsing should have done validation
			return nil, errors.InternalError(ctx, err)
		}
		msg := &threading.MessagePost{
			Internal: in.Message.Internal,
			Summary:  plainText,
		}
		msg.Text, err = textBML.Format()
		if err != nil {
			// Shouldn't fail here since the parsing should have done validation
			return nil, errors.InternalError(ctx, err)
		}

		attachments, err := processIncomingAttachments(ctx, ram, svc, ent, in.OrganizationID, in.Message.Attachments, nil)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		msg.Attachments = attachments

		var savedMessage *threading.SavedMessage
		if oldMessage != nil {
			updateSavedMessageRes, err := ram.UpdateSavedMessage(ctx, &threading.UpdateSavedMessageRequest{
				SavedMessageID: oldMessage.ID,
				Title:          in.Title,
				Content: &threading.UpdateSavedMessageRequest_Message{
					Message: msg,
				},
			})
			if err != nil {
				return nil, errors.Trace(err)
			}
			savedMessage = updateSavedMessageRes.SavedMessage
		} else {
			req := &threading.CreateSavedMessageRequest{
				OrganizationID:  in.OrganizationID,
				CreatorEntityID: ent.ID,
				OwnerEntityID:   ent.ID,
				Title:           in.Title,
				Content: &threading.CreateSavedMessageRequest_Message{
					Message: msg,
				},
			}
			if in.Shared {
				req.OwnerEntityID = in.OrganizationID
			}
			createSavedMessageRes, err := ram.CreateSavedMessage(ctx, in.OrganizationID, req)
			if err != nil {
				return nil, errors.Trace(err)
			}
			savedMessage = createSavedMessageRes.SavedMessage
		}

		attachmentProperties := attachmentsIncluded(attachments)
		properties := make(map[string]interface{}, len(attachmentProperties)+2)

		properties["organization_id"] = savedMessage.OrganizationID
		properties["private"] = savedMessage.OwnerEntityID != savedMessage.OrganizationID
		for key, value := range attachmentProperties {
			properties[key] = value
		}

		analytics.SegmentTrack(&segment.Track{
			Event:      fmt.Sprintf("saved-message"),
			UserId:     ent.AccountID,
			Properties: properties,
		})

		return &saveMessageOutput{
			Success: true,
		}, nil
	}),
}
