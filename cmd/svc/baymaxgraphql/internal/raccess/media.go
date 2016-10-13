package raccess

import (
	"context"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/svc/media"
)

func (m *resourceAccessor) ClaimMedia(ctx context.Context, req *media.ClaimMediaRequest) error {
	account := gqlctx.Account(ctx)
	if account == nil {
		return errors.ErrNotAuthenticated(ctx)
	}
	canAccess, err := m.media.CanAccess(ctx, &media.CanAccessRequest{
		MediaIDs:  req.MediaIDs,
		AccountID: account.ID,
	})
	if err != nil {
		return err
	} else if !canAccess.CanAccess {
		return errors.ErrNotAuthorized(ctx, strings.Join(req.MediaIDs, ", "))
	}
	_, err = m.media.ClaimMedia(ctx, req)
	return err
}

func (m *resourceAccessor) CloneMedia(ctx context.Context, req *media.CloneMediaRequest) (*media.CloneMediaResponse, error) {
	return m.media.CloneMedia(ctx, req)
}

func (m *resourceAccessor) MediaInfo(ctx context.Context, mediaID string) (*media.MediaInfo, error) {
	// TODO: Auth the resource once it comes back and we know who it belongs to
	res, err := m.media.MediaInfos(ctx, &media.MediaInfosRequest{
		MediaIDs: []string{mediaID},
	})
	if err != nil {
		return nil, err
	}
	info := res.MediaInfos[mediaID]
	if info == nil {
		return nil, ErrNotFound
	}
	return info, nil
}

func (m *resourceAccessor) UpdateMedia(ctx context.Context, req *media.UpdateMediaRequest) (*media.MediaInfo, error) {
	resp, err := m.media.UpdateMedia(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.MediaInfo, nil
}
