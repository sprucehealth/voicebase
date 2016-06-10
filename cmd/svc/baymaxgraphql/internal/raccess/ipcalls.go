package raccess

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/svc/excomms"
	"golang.org/x/net/context"
)

func (m *resourceAccessor) InitiateIPCall(ctx context.Context, req *excomms.InitiateIPCallRequest) (*excomms.InitiateIPCallResponse, error) {
	// TODO: right now the caller is doing what is essentially authorization (matching account ID + org to get entity).
	//       We might want to do it here for completeness and safety, but for now leaving it out. Ideally restructuring auth
	//       would remove the need.
	return m.excomms.InitiateIPCall(ctx, req)
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
