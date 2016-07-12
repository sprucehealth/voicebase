package gqlctx

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

func TestFeatures_Unset(t *testing.T) {
	// Unset feature should always be false
	test.Equals(t, false, FeatureEnabled(context.Background(), VideoCalling))
}

func TestFeatures_Set(t *testing.T) {
	ctx := WithFeature(context.Background(), VideoCalling, true)
	test.Equals(t, true, FeatureEnabled(ctx, VideoCalling))
}

func TestFeatures_Lazy(t *testing.T) {
	calls := 0
	ctx := WithLazyFeature(context.Background(), VideoCalling, func(context.Context) bool {
		calls++
		return true
	})
	test.Equals(t, true, FeatureEnabled(ctx, VideoCalling))
	test.Equals(t, 1, calls)

	// Lazy feature should only execute the function once
	test.Equals(t, true, FeatureEnabled(ctx, VideoCalling))
	test.Equals(t, 1, calls)
}
