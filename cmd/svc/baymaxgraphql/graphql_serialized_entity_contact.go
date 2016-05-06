package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
)

var platformEnumType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "PlatformType",
		Description: "Type of platform",
		Values: graphql.EnumValueConfigMap{
			"IOS": &graphql.EnumValueConfig{
				Value:       "IOS",
				Description: "Apple IOS application",
			},
			"ANDROID": &graphql.EnumValueConfig{
				Value:       "ANDROID",
				Description: "Android applicatoin",
			},
		},
	},
)

func lookupSerializedEntityContact(ctx context.Context, ram raccess.ResourceAccessor, entityID string, platform directory.Platform) (interface{}, error) {
	sec, err := ram.SerializedEntityContact(ctx, entityID, platform)
	if err != nil {
		return nil, err
	}
	return string(sec.SerializedEntityContact), nil
}
