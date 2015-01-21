package admin

package admin

import (
  "net/http"

  "github.com/sprucehealth/backend/api"
  "github.com/sprucehealth/backend/apiservice"
  "github.com/sprucehealth/backend/libs/httputil"
  "github.com/sprucehealth/backend/www"
)

type layoutTemplateHandler struct {
  dataAPI api.DataAPI
}

type 

type layoutVersionGETResponse []byte

func NewLayoutTemplateHandler(dataAPI api.DataAPI) http.Handler {
  return httputil.SupportedMethods(
    apiservice.SupportedRoles(
      &layoutTemplateHandler{
        dataAPI: dataAPI,
      }, []string{api.ADMIN_ROLE}), []string{"GET"})
}

func (h *layoutTemplateHandler) IsAuthorized(r *http.Request) (bool, error) {
  return true, nil
}

func (h *layoutTemplateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  // get a map of layout versions and info
  versionMapping, err := h.dataAPI.LayoutVersionMapping()
  if err != nil {
    www.InternalServerError(w, r, err)
    return
  }

  www.JSONResponse(w, r, http.StatusOK, versionMapping)
}
