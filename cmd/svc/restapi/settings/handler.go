package settings

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common/config"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type handler struct {
	minimumAppVersionConfigs *config.MinimumAppVersionConfigs
}

type SettingsResponse struct {
	UpgradeInfo *upgradeInfo `json:"upgrade_info"`
}

type upgradeInfo struct {
	UpgradeURL string `json:"upgrade_url"`
	Required   bool   `json:"required"`
}

func NewHandler(minimumAppVersionConfigs *config.MinimumAppVersionConfigs) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(&handler{
			minimumAppVersionConfigs: minimumAppVersionConfigs,
		}), httputil.Get)
}

func (h *handler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	sHeaders := device.ExtractSpruceHeaders(w, r)

	if h.minimumAppVersionConfigs != nil {
		minAppVersionConfig, err := h.minimumAppVersionConfigs.Get(sHeaders.AppType + "-" + sHeaders.AppEnvironment)
		if err == nil && sHeaders.AppVersion.LessThan(minAppVersionConfig.AppVersion) {
			httputil.JSONResponse(w, http.StatusOK, struct {
				Settings SettingsResponse `json:"settings"`
			}{
				Settings: SettingsResponse{
					UpgradeInfo: &upgradeInfo{
						UpgradeURL: minAppVersionConfig.AppStoreURL,
						Required:   true,
					},
				},
			})
			return
		}
	}

	apiservice.WriteJSONSuccess(w)
}