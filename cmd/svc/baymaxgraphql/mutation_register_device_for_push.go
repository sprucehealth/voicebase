package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/graphql"
)

var registerDeviceForPushMutation = &graphql.Field{
	Type: graphql.NewNonNull(registerDeviceForPushOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(registerDeviceForPushInputType)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)
		sh := devicectx.SpruceHeaders(ctx)
		golog.Debugf("Registering Device For Push: Account:%s Device:%+v", acc.ID, sh)
		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		deviceToken, _ := input["deviceToken"].(string)
		if err := svc.notification.RegisterDeviceForPush(&notification.DeviceRegistrationInfo{
			ExternalGroupID: acc.ID,
			DeviceToken:     deviceToken,
			Platform:        sh.Platform.String(),
			PlatformVersion: sh.PlatformVersion,
			AppVersion:      sh.AppVersion.String(),
			Device:          sh.Device,
			DeviceModel:     sh.DeviceModel,
			DeviceID:        sh.DeviceID,
		}); err != nil {
			golog.Errorf(err.Error())
			return nil, errors.New("device registration failed")
		}

		return &registerDeviceForPushOutput{
			ClientMutationID: mutationID,
			Success:          true,
		}, nil
	}),
}
