package settings

import (
	"net/http"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common/config"
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

func NewHandler(minimumAppVersionConfigs *config.MinimumAppVersionConfigs) http.Handler {
	return &handler{
		minimumAppVersionConfigs: minimumAppVersionConfigs,
	}
}

func (h *handler) NonAuthenticated() bool {
	return true
}

func (h *handler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_GET {
		return false, apiservice.NewResourceNotFoundError("", r)
	}
	return true, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sHeaders := apiservice.ExtractSpruceHeaders(r)

	if h.minimumAppVersionConfigs != nil {
		minAppVersionConfig, err := h.minimumAppVersionConfigs.Get(sHeaders.AppType + "-" + sHeaders.AppEnvironment)
		if err == nil && sHeaders.AppVersion.LessThan(minAppVersionConfig.AppVersion) {
			apiservice.WriteJSON(w, map[string]interface{}{
				"settings": SettingsResponse{
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
