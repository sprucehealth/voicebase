package main

import (
	"context"
	"fmt"

	segment "github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/analytics"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
)

// batchPostMessage
type batchPostMessagesInput struct {
	ClientMutationID string          `gql:"clientMutationId"`
	UUID             string          `gql:"uuid"`
	ThreadIDs        []string        `gql:"threadIDs"`
	Messages         []*messageInput `gql:"msgs"`
}

var batchPostMessagesInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "BatchPostMessagesInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"uuid":             &graphql.InputObjectFieldConfig{Type: graphql.String},
			"threadIDs":        &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.ID))},
			"msgs":             &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(messageInputType))},
		},
	},
)

const (
	// TODO: This limit is currently arbitrary and due to the authorization/meta building we have to do per thread synchronously
	batchPostMessagesMaxThreads                 = 2000
	batchPostMessagesErrorCodeTooManyThreads    = "TOO_MANY_THREADS"
	batchPostMessagesErrorCodeAcrossOrgs        = "CANNOT_BATCH_POST_ACROSS_ORGANIZATIONS"
	batchPostMessagesErrorCodeInvalidAttachment = "INVALID_ATTACHMENT"
	batchPostMessagesErrorCodeInvalidInput      = "INVALID_INPUT"
)

var batchPostMessageErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "BatchPostMessagesErrorCode",
	Values: graphql.EnumValueConfigMap{
		batchPostMessagesErrorCodeTooManyThreads: &graphql.EnumValueConfig{
			Value:       batchPostMessagesErrorCodeTooManyThreads,
			Description: fmt.Sprintf("A maximum of %d threads can be posted to at once", batchPostMessagesMaxThreads),
		},
		batchPostMessagesErrorCodeAcrossOrgs: &graphql.EnumValueConfig{
			Value:       batchPostMessagesErrorCodeAcrossOrgs,
			Description: "A batch post cannot contain thread ids belonging to multiple organizations",
		},
		batchPostMessagesErrorCodeInvalidInput: &graphql.EnumValueConfig{
			Value:       batchPostMessagesErrorCodeInvalidInput,
			Description: "The input for the post was invalid",
		},
	},
})

type batchPostMessagesOutput struct {
	ClientMutationID string                `json:"clientMutationId,omitempty"`
	UUID             string                `json:"uuid,omitempty"`
	Success          bool                  `json:"success"`
	ErrorCode        string                `json:"errorCode,omitempty"`
	ErrorMessage     string                `json:"errorMessage,omitempty"`
	RequestStatus    *models.RequestStatus `json:"requestStatus"`
}

var batchPostMessagesOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "BatchPostMessagesPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"uuid":             &graphql.Field{Type: graphql.String},
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: postMessageErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"requestStatus":    &graphql.Field{Type: requestStatusType},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*batchPostMessagesOutput)
			return ok
		},
	},
)

var batchPostMessagesMutation = &graphql.Field{
	Type: graphql.NewNonNull(batchPostMessagesOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(batchPostMessagesInputType)},
	},
	Resolve: apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ram := raccess.ResourceAccess(p)
		ctx := p.Context

		input := p.Args["input"].(map[string]interface{})
		var in batchPostMessagesInput
		if err := gqldecode.Decode(input, &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(ctx, err)
		}

		return batchPostMessages(ctx, svc, ram, &in)
	}),
}

func batchPostMessages(ctx context.Context, svc *service, ram raccess.ResourceAccessor, in *batchPostMessagesInput) (*batchPostMessagesOutput, error) {
	acc := gqlctx.Account(ctx)
	if len(in.ThreadIDs) > batchPostMessagesMaxThreads {
		return &batchPostMessagesOutput{
			ClientMutationID: in.ClientMutationID,
			ErrorCode:        batchPostMessagesErrorCodeTooManyThreads,
			ErrorMessage:     fmt.Sprintf("A maximum of %d threads are allowed in a single batch post", batchPostMessagesMaxThreads),
		}, nil
	}
	if len(in.ThreadIDs) == 0 {
		return &batchPostMessagesOutput{
			ClientMutationID: in.ClientMutationID,
			ErrorCode:        batchPostMessagesErrorCodeInvalidInput,
			ErrorMessage:     "At least 1 thread id is required",
		}, nil
	}
	if len(in.Messages) == 0 {
		return &batchPostMessagesOutput{
			ClientMutationID: in.ClientMutationID,
			ErrorCode:        batchPostMessagesErrorCodeInvalidInput,
			ErrorMessage:     "At least 1 message is required",
		}, nil
	}

	// TODO: Dedupe the input id list with the results and check for missing ids in results

	// Lookup the threads since we need primary entity info to build out endpoints
	threadsResp, err := ram.Threads(ctx, &threading.ThreadsRequest{
		ThreadIDs: in.ThreadIDs,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Error while looking up threads to build primary entity info")
	}
	if len(threadsResp.Threads) == 0 {
		return nil, errors.Errorf("No threads returned for ID list %v", in.ThreadIDs)
	}
	threads := threadsResp.Threads

	// Assert that we're not cross org posting
	orgID := threads[0].OrganizationID
	for _, t := range threads {
		if t.OrganizationID != orgID {
			return &batchPostMessagesOutput{
				ClientMutationID: in.ClientMutationID,
				ErrorCode:        batchPostMessagesErrorCodeAcrossOrgs,
				ErrorMessage:     "Batch messages cannot be applied across organizations",
			}, nil
		}
	}

	requestingEntity, err := entityInOrgForAccountID(ctx, ram, threads[0].OrganizationID, acc)
	if err != nil {
		return nil, err
	}

	// Build out this strange backwards map to facilitate building the thread entity map in the future
	primaryEntityIDToThreadIDList := make(map[string][]string)
	// Collect our primary entities in bulk
	primaryEntityIDs := make([]string, 0, len(threads))
	for _, thread := range threads {
		if thread.PrimaryEntityID != "" {
			primaryEntityIDs = append(primaryEntityIDs, thread.PrimaryEntityID)
			primaryEntityIDToThreadIDList[thread.PrimaryEntityID] = append(primaryEntityIDToThreadIDList[thread.PrimaryEntityID], thread.ID)
		}
	}
	primaryEntities, err := ram.Entities(ctx, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_BatchEntityID{
			BatchEntityID: &directory.IDList{
				IDs: primaryEntityIDs,
			},
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	})
	if err != nil {
		return nil, err
	}

	// Build a map of threadID to primary entity
	threadIDToPrimaryEntity := make(map[string]*directory.Entity)
	for _, primaryEntity := range primaryEntities {
		threadIDs := primaryEntityIDToThreadIDList[primaryEntity.ID]
		for _, threadID := range threadIDs {
			threadIDToPrimaryEntity[threadID] = primaryEntity
		}
	}

	postMessagesRequests := make([]*threading.PostMessagesRequest, len(threads))
	parallel := conc.NewParallel()
	for i, t := range threads {
		// Capture our request index and thread
		idx := i
		thread := t
		parallel.Go(func() error {
			messages := make([]*threading.MessagePost, len(in.Messages))
			for j, message := range in.Messages {
				msg, err := transformRequestToMessagePost(ctx, svc, ram, message, thread, requestingEntity, threadIDToPrimaryEntity[thread.ID])
				if err != nil {
					return errors.Trace(err)
				}
				messages[j] = msg
			}
			postMessagesRequests[idx] = &threading.PostMessagesRequest{
				UUID:     in.UUID,
				ThreadID: thread.ID,
				// TODO: Do we want to parameterize this?
				FromEntityID: requestingEntity.ID,
				Messages:     messages,
			}
			return nil
		})
	}

	if err := parallel.Wait(); err != nil {
		if e, ok := errors.Cause(err).(errInvalidAttachment); ok {
			return &batchPostMessagesOutput{
				ErrorCode:    batchPostMessagesErrorCodeInvalidAttachment,
				ErrorMessage: string(e),
			}, nil
		}
		return nil, errors.Trace(err)
	}

	req := &threading.BatchPostMessagesRequest{
		UUID:                 in.UUID,
		RequestingEntity:     requestingEntity.ID,
		PostMessagesRequests: postMessagesRequests,
	}
	resp, err := ram.BatchPostMessages(ctx, req)
	if err != nil {
		return nil, errors.Trace(err)
	}

	trackBatchPostMessages(ctx, req, threads[0].OrganizationID, resp.BatchJob)
	return &batchPostMessagesOutput{
		ClientMutationID: in.ClientMutationID,
		Success:          true,
		RequestStatus:    transformRequestStatusToResponse(ctx, resp.BatchJob),
	}, nil
}

func trackBatchPostMessages(ctx context.Context, req *threading.BatchPostMessagesRequest, orgID string, batchJob *threading.BatchJob) {
	acc := gqlctx.Account(ctx)
	properties := make(map[string]interface{}, 3)

	properties["organization_id"] = orgID
	properties["thread_count"] = len(req.PostMessagesRequests)
	properties["request_status_id"] = batchJob.ID

	analytics.SegmentTrack(&segment.Track{
		Event:      "batch-posted-messages",
		UserId:     acc.ID,
		Properties: properties,
	})
}
