package mock

import (
	"context"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/media"
)

func (m *ResourceAccessor) CloneMedia(ctx context.Context, req *media.CloneMediaRequest) (*media.CloneMediaResponse, error) {
	rets := m.Record(req)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*media.CloneMediaResponse), mock.SafeError(rets[1])
}
