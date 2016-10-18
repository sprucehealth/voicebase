package workers

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/auth/internal/dal/test"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

type workerTest struct {
	dal *test.MockDAL
}

func (wt *workerTest) finish() {
	mock.FinishAll(wt.dal)
}

func setup(t *testing.T) *workerTest {
	return &workerTest{
		dal: test.NewMockDAL(t),
	}
}

func TestCleanupExpiredTokens(t *testing.T) {
	wt := setup(t)
	wkrs := New(wt.dal)
	wkrs.clk = clock.NewManaged(time.Now())
	wt.dal.Expect(mock.NewExpectation(wt.dal.DeleteExpiredAuthTokens, wkrs.clk.Now().Add(-tokenCleanupDelay)))
	wkrs.cleanupExpiredTokens()
	wt.finish()
}
