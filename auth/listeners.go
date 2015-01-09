package auth

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
)

func InitListeners(authAPI api.AuthAPI, dispatcher *dispatch.Dispatcher) {
	dispatcher.Subscribe(func(ev *AuthenticatedEvent) error {
		headers := ev.SpruceHeaders
		if err := authAPI.UpdateAppDevice(ev.AccountID,
			headers.AppVersion, headers.Platform,
			headers.PlatformVersion,
			headers.Device,
			headers.DeviceModel,
			headers.AppBuild); err != nil {
			golog.Errorf(err.Error())
		}
		return nil
	})
}
