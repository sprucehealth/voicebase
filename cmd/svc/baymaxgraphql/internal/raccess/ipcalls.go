package raccess

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/svc/excomms"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (m *resourceAccessor) InitiateIPCall(ctx context.Context, req *excomms.InitiateIPCallRequest) (*excomms.InitiateIPCallResponse, error) {
	// TODO: right now the caller is doing what is essentially authorization (matching account ID + org to get entity).
	//       We might want to do it here for completeness and safety, but for now leaving it out. Ideally restructuring auth
	//       would remove the need.
	return m.excomms.InitiateIPCall(ctx, req)
}

func (m *resourceAccessor) IPCall(ctx context.Context, id string) (*excomms.IPCall, error) {
	acc := gqlctx.Account(ctx)
	if acc == nil {
		return nil, errors.ErrNotAuthenticated(ctx)
	}
	res, err := m.excomms.IPCall(ctx, &excomms.IPCallRequest{
		IPCallID:  id,
		AccountID: acc.ID,
	})
	if grpc.Code(err) == codes.NotFound {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	// Make sure account is a participant
	authorized := false
	for _, p := range res.Call.Participants {
		if p.AccountID == acc.ID {
			authorized = true
			break
		}
	}
	if !authorized {
		return nil, errors.ErrNotAuthorized(ctx, id)
	}
	return res.Call, nil
}

func (m *resourceAccessor) PendingIPCalls(ctx context.Context) (*excomms.PendingIPCallsResponse, error) {
	acc := gqlctx.Account(ctx)
	if acc == nil {
		return nil, errors.ErrNotAuthenticated(ctx)
	}
	// No authorization required since we're querying by the authenticated account ID
	return m.excomms.PendingIPCalls(ctx, &excomms.PendingIPCallsRequest{AccountID: acc.ID})
}

func (m *resourceAccessor) UpdateIPCall(ctx context.Context, req *excomms.UpdateIPCallRequest) (*excomms.UpdateIPCallResponse, error) {
	acc := gqlctx.Account(ctx)
	if acc == nil {
		return nil, errors.ErrNotAuthenticated(ctx)
	}
	// Caller may have set it.. but force it here for auth purposes. No auth is thus required since the update
	// is keyed off the account ID, and it's not possible to update a call where the account ID doesn't match.
	req.AccountID = acc.ID
	return m.excomms.UpdateIPCall(ctx, req)
}
