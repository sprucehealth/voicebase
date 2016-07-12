package main

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/graphql"
)

func TestForceUpgradeStatus(t *testing.T) {
	ctx := context.Background()
	ctx = devicectx.WithSpruceHeaders(ctx, &device.SpruceHeaders{
		AppType:         "baymax",
		AppEnvironment:  "dev",
		AppVersion:      &encoding.Version{Major: 1},
		AppBuild:        "001",
		Platform:        device.IOS,
		PlatformVersion: "7.1.1",
		Device:          "Phone",
		DeviceModel:     "iPhone6,1",
		DeviceID:        "12917415",
	})
	p := graphql.ResolveParams{
		Context: ctx,
		Info: graphql.ResolveInfo{
			RootValue: map[string]interface{}{},
		},
	}

	res, err := forceUpgradeQuery.Resolve(p)
	test.OK(t, err)
	test.Equals(t, &models.ForceUpgradeStatus{
		Upgrade: false,
	}, res)
}

func TestForceUpgradeStatus_Hook(t *testing.T) {
	ctx := context.Background()
	ctx = devicectx.WithSpruceHeaders(ctx, &device.SpruceHeaders{
		AppType:         "baymax",
		AppEnvironment:  "dev",
		AppVersion:      &encoding.Version{Patch: 9999},
		AppBuild:        "001",
		Platform:        device.IOS,
		PlatformVersion: "7.1.1",
		Device:          "Phone",
		DeviceModel:     "iPhone6,1",
		DeviceID:        "12917415",
	})
	p := graphql.ResolveParams{
		Context: ctx,
		Info: graphql.ResolveInfo{
			RootValue: map[string]interface{}{},
		},
	}

	res, err := forceUpgradeQuery.Resolve(p)
	test.OK(t, err)
	test.Equals(t, &models.ForceUpgradeStatus{
		Upgrade:     true,
		UserMessage: "Force upgrade works!",
		URL:         "https://www.google.com",
	}, res)
}
