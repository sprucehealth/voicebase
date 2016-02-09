package main

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/svc/directory"
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

func lookupSerializedEntityContact(ctx context.Context, svc *service, entityID string, platform directory.Platform) (interface{}, error) {
	res, err := svc.directory.SerializedEntityContact(ctx, &directory.SerializedEntityContactRequest{
		EntityID: entityID,
		Platform: platform,
	})
	if grpc.Code(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, internalError(err)
	}
	return string(res.SerializedEntityContact.SerializedEntityContact), nil
}
