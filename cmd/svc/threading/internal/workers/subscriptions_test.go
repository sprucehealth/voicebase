package workers

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal/dalmock"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory/directorymock"
	"github.com/sprucehealth/backend/svc/threading/threadingmock"
)

type subscriptionsTest struct {
	ctx             context.Context
	ctrl            *gomock.Controller
	threadsClient   *threadingmock.MockThreadsClient
	directoryClient *directorymock.MockDirectoryClient
	dal             *dalmock.DAL
}

func (t *subscriptionsTest) Finish() {
	t.ctrl.Finish()
	mock.FinishAll(t.dal)
}

func newSubscriptionsTest(t *testing.T) *subscriptionsTest {
	ctrl := gomock.NewController(t)
	return &subscriptionsTest{
		ctx:             context.Background(),
		ctrl:            ctrl,
		directoryClient: directorymock.NewMockDirectoryClient(ctrl),
		threadsClient:   threadingmock.NewMockThreadsClient(ctrl),
		dal:             dalmock.New(t),
	}
}
