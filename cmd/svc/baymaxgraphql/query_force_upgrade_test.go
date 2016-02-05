package main

import (
	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
	"testing"
)

func TestForceUpgradeStatus(t *testing.T) {
	ctx := context.Background()
	ctx = ctxWithSpruceHeaders(ctx, &apiservice.SpruceHeaders{
		AppType:         "baymax",
		AppEnvironment:  "dev",
		AppVersion:      &encoding.Version{Major: 1},
		AppBuild:        "001",
		Platform:        common.IOS,
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
	test.Equals(t, &forceUpgradeStatus{
		Upgrade: false,
	}, res)
}

func TestForceUpgradeStatus_Hook(t *testing.T) {
	ctx := context.Background()
	ctx = ctxWithSpruceHeaders(ctx, &apiservice.SpruceHeaders{
		AppType:         "baymax",
		AppEnvironment:  "dev",
		AppVersion:      &encoding.Version{Patch: 9999},
		AppBuild:        "001",
		Platform:        common.IOS,
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
	test.Equals(t, &forceUpgradeStatus{
		Upgrade:     true,
		UserMessage: "Force upgrade works!",
		URL:         "https://www.google.com",
	}, res)
}
