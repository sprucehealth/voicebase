package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/graphql"
)

var forceUpgradeStatusType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ForceUpgradeStatus",
		Fields: graphql.Fields{
			"url":         &graphql.Field{Type: graphql.String},
			"userMessage": &graphql.Field{Type: graphql.String},
			"upgrade":     &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*models.ForceUpgradeStatus)
			return ok
		},
	},
)

var forceUpgradeQuery = &graphql.Field{
	Type: graphql.NewNonNull(forceUpgradeStatusType),
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {

		sh := gqlctx.SpruceHeaders(p.Context)

		// TODO: The logic of whether or not to force upgrade is intentionally left out for now
		// so that we can add the work if and when we need it. For now just ensuring that we have
		// all the information we need to make the decision of whether or not to force upgrade.
		if sh.AppVersion == nil {
			return nil, fmt.Errorf("app version not specified in request header")
		} else if sh.Platform == "" {
			return nil, fmt.Errorf("platform not specified in request header")
		} else if sh.PlatformVersion == "" {
			return nil, fmt.Errorf("platform versin required")
		} else if sh.AppType == "" {
			return nil, fmt.Errorf("app type not specified in request header")
		} else if sh.AppEnvironment == "" {
			return nil, fmt.Errorf("app environment not specified in request header")
		} else if sh.AppBuild == "" {
			return nil, fmt.Errorf("build number not specified in request header")
		} else if sh.Device == "" {
			return nil, fmt.Errorf("device not specified in request header")
		} else if sh.DeviceModel == "" {
			return nil, fmt.Errorf("device model not specified in request header")
		}

		// Putting a hook in place to test force upgrade
		if sh.AppVersion.Equals(&encoding.Version{Major: 0, Minor: 0, Patch: 9999}) {
			return &models.ForceUpgradeStatus{
				Upgrade:     true,
				URL:         "https://www.google.com",
				UserMessage: "Force upgrade works!",
			}, nil
		}

		return &models.ForceUpgradeStatus{
			Upgrade: false,
		}, nil
	},
}
