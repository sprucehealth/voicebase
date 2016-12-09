package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type deleteThreadOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var deleteThreadInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "DeleteThreadInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             newUUIDInputField(),
		"threadID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
	},
})

const (
	deleteThreadErrorCodePatientCreatedAccount = "PATIENT_ALREADY_CREATED_ACCOUNT"
)

var deleteThreadErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "DeleteThreadErrorCode",
	Values: graphql.EnumValueConfigMap{
		deleteThreadErrorCodePatientCreatedAccount: &graphql.EnumValueConfig{
			Value:       deleteThreadErrorCodePatientCreatedAccount,
			Description: "Patient has already created an account",
		},
	},
})

var deleteThreadOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "DeleteThreadPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientMutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: deleteThreadErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*deleteThreadOutput)
		return ok
	},
})

var deleteThreadMutation = &graphql.Field{
	Type: graphql.NewNonNull(deleteThreadOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(deleteThreadInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		svc := serviceFromParams(p)
		acc := gqlctx.Account(ctx)
		if acc == nil {
			return nil, errors.ErrNotAuthenticated(ctx)
		}

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		threadID := input["threadID"].(string)

		// Make sure thread exists (wasn't deleted) and get organization ID to be able to fetch entity for the account
		thread, err := ram.Thread(ctx, threadID, "")
		if err != nil {
			switch grpc.Code(err) {
			case codes.NotFound:
				return nil, errors.UserError(ctx, errors.ErrTypeNotFound, "Thread does not exist.")
			}
			return nil, errors.InternalError(ctx, err)
		}

		ent, err := entityInOrgForAccountID(ctx, ram, thread.OrganizationID, acc)
		if err != nil {
			return nil, err
		}
		if ent == nil || ent.Type != directory.EntityType_INTERNAL {
			return nil, errors.UserError(ctx, errors.ErrTypeNotAuthorized, "Permission denied.")
		}

		if thread.Type == threading.THREAD_TYPE_SECURE_EXTERNAL {
			// ensure that primary entity has not created account yet
			entity, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
				Key: &directory.LookupEntitiesRequest_EntityID{
					EntityID: thread.PrimaryEntityID,
				},
				Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
				RootTypes: []directory.EntityType{directory.EntityType_PATIENT}})
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			if entity.AccountID != "" {
				return &deleteThreadOutput{
					ClientMutationID: mutationID,
					Success:          false,
					ErrorMessage:     "Cannot delete thread if the patient has already created an account",
					ErrorCode:        deleteThreadErrorCodePatientCreatedAccount,
				}, nil
			}

			if _, err := svc.invite.DeleteInvite(ctx, &invite.DeleteInviteRequest{
				DeleteInviteKey: invite.DELETE_INVITE_KEY_PARKED_ENTITY_ID,
				Key: &invite.DeleteInviteRequest_ParkedEntityID{
					ParkedEntityID: entity.ID,
				},
			}); err != nil {
				return nil, errors.InternalError(ctx, err)
			}
		}

		if err := ram.DeleteThread(ctx, threadID, ent.ID); err != nil {
			return nil, err
		}

		return &deleteThreadOutput{
			ClientMutationID: mutationID,
			Success:          true,
		}, nil
	},
}
