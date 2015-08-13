package apiservice

import (
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
)

func TestContext(t *testing.T) {
	account, ok := CtxAccount(context.Background())
	test.Equals(t, false, ok)
	test.Equals(t, (*common.Account)(nil), account)

	account, ok = CtxAccount(CtxWithAccount(context.Background(), &common.Account{Role: api.RolePatient, ID: 1}))
	test.Equals(t, true, ok)
	test.Assert(t, account != nil, "Account should not be nil")
	test.Equals(t, int64(1), account.ID)
	test.Equals(t, api.RolePatient, account.Role)

	cache, ok := CtxCache(context.Background())
	test.Equals(t, false, ok)
	test.Assert(t, cache == nil, "Cache should be nil")

	cache, ok = CtxCache(CtxWithCache(context.Background(), nil))
	test.Equals(t, true, ok)
	test.Assert(t, cache != nil, "Cache should not be nil")
}
