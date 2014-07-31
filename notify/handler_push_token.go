package notify

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/aws/sns"
)

type notificationHandler struct {
	dataApi             api.DataAPI
	notificationConfigs *config.NotificationConfigs
	snsClient           sns.SNSService
}

type requestData struct {
	DeviceToken string `schema:"device_token,required"`
}

func NewNotificationHandler(dataApi api.DataAPI, configs *config.NotificationConfigs, snsClient sns.SNSService) http.Handler {
	return apiservice.SupportedMethods(&notificationHandler{
		dataApi:             dataApi,
		notificationConfigs: configs,
		snsClient:           snsClient,
	}, []string{apiservice.HTTP_POST})
}

func (n *notificationHandler) IsAuthorized(r *http.Request) (bool, error) {
	return true, nil
}

func (n *notificationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rData := &requestData{}
	if err := apiservice.DecodeRequestData(rData, r); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	if rData.DeviceToken == "" {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Device token cannot be empty")
		return
	}

	sHeaders := apiservice.ExtractSpruceHeaders(r)

	// we need the minimum headers set to be able to accept the token
	if sHeaders.Platform == "" || sHeaders.AppEnvironment == "" || sHeaders.AppType == "" {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to determine which endpoint to use for push notifications: need platform, app-environment and app-type to be set in request header")
		return
	}

	// lookup the application config for configuring push notifications
	configName := config.DetermineNotificationConfigName(sHeaders.Platform, sHeaders.AppType, sHeaders.AppEnvironment)
	notificationConfig, err := n.notificationConfigs.Get(configName)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to find right notification config for "+configName)
		return
	}

	// lookup any existing push config associated with this device token
	existingPushConfigData, err := n.dataApi.GetPushConfigData(rData.DeviceToken)
	if err != nil && err != api.NoRowsError {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get push config data for device token: "+err.Error())
		return
	}

	var pushEndpoint string
	if existingPushConfigData != nil {
		pushEndpoint = existingPushConfigData.PushEndpoint
	}

	// if the device token exists and has changed, register the device token for the user to get the application endpoint
	if existingPushConfigData == nil || rData.DeviceToken != existingPushConfigData.DeviceToken {
		pushEndpoint, err = n.snsClient.CreatePlatformEndpoint(notificationConfig.SNSApplicationEndpoint, rData.DeviceToken)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to register token for push notifications: "+err.Error())
			return
		}
	}

	newPushConfigData := &common.PushConfigData{
		AccountId:       apiservice.GetContext(r).AccountId,
		DeviceToken:     rData.DeviceToken,
		PushEndpoint:    pushEndpoint,
		Platform:        sHeaders.Platform,
		PlatformVersion: sHeaders.PlatformVersion,
		AppType:         sHeaders.AppType,
		AppEnvironment:  sHeaders.AppEnvironment,
		AppVersion:      sHeaders.AppVersion,
		Device:          sHeaders.Device,
		DeviceModel:     sHeaders.DeviceModel,
		DeviceID:        sHeaders.DeviceID,
	}

	// update the device token for the user
	if err := n.dataApi.SetOrReplacePushConfigData(newPushConfigData); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update push config data: "+err.Error())
		return
	}

	// return success
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.SuccessfulGenericJSONResponse())
}
