package features

import (
	"testing"

	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func TestContext(t *testing.T) {
	ctx := context.Background()
	test.Equals(t, nullSet{}, CtxSet(ctx))
	ctx = CtxWithSet(ctx, MapSet(map[string]struct{}{"foo": struct{}{}}))
	test.Equals(t, []string{"foo"}, CtxSet(ctx).Enumerate())
	test.Equals(t, true, CtxSet(ctx).Has("foo"))
}
