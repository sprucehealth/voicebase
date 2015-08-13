package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type pathwayMenuHandler struct {
	dataAPI api.DataAPI
}

func newPathwayMenuHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&pathwayMenuHandler{
		dataAPI: dataAPI,
	}, httputil.Get, httputil.Put)
}

func (h *pathwayMenuHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.get(ctx, w, r)
	case "PUT":
		h.put(ctx, w, r)
	}
}

func (h *pathwayMenuHandler) get(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "GetPathwayMenu", nil)
	menu, err := h.dataAPI.PathwayMenu()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, menu)
}

func (h *pathwayMenuHandler) put(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)
	audit.LogAction(account.ID, "AdminAPI", "UpdatePathwayMenu", nil)
	menu := &common.PathwayMenu{}
	if err := json.NewDecoder(r.Body).Decode(menu); err != nil {
		www.APIBadRequestError(w, r, err.Error())
		return
	}
	if err := validatePathwayMenu(h.dataAPI, menu); err != nil {
		www.APIBadRequestError(w, r, err.Error())
		return
	}
	if err := h.dataAPI.UpdatePathwayMenu(menu); err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, menu)
}

func validatePathwayMenu(dataAPI api.DataAPI, menu *common.PathwayMenu) error {
	if menu == nil {
		return errors.New("menu not set")
	}
	if menu.Title == "" {
		return errors.New("menu title required")
	}
	if len(menu.Items) == 0 {
		return errors.New("menu items cannot be empty")
	}
	for _, it := range menu.Items {
		if it.Title == "" {
			return errors.New("menu item title required")
		}
		for _, c := range it.Conditionals {
			if err := validateConditional(c); err != nil {
				return err
			}
		}
		switch it.Type {
		default:
			return fmt.Errorf("invalid menu item type '%s'", it.Type.String())
		case common.PathwayMenuItemTypeMenu:
			if err := validatePathwayMenu(dataAPI, it.Menu); err != nil {
				return err
			}
		case common.PathwayMenuItemTypePathway:
			if err := validatePathway(dataAPI, it.PathwayTag); err != nil {
				return err
			}
		}
	}
	return nil
}

func validatePathway(dataAPI api.DataAPI, pathwayTag string) error {
	if pathwayTag == "" {
		return fmt.Errorf("pathway tag is required")
	}
	// TODO: this does one query per pathway which is very inefficient. Could optimize
	// to do in batch but it would complicate this code quite a bit. This is only
	// used when updating the menu from the admin so should be fine.
	_, err := dataAPI.PathwayForTag(pathwayTag, api.PONone)
	if api.IsErrNotFound(err) {
		return fmt.Errorf("pathway with tag '%s' not found", pathwayTag)
	} else if err != nil {
		golog.Errorf(err.Error())
		return fmt.Errorf("internal error checking pathway with tag '%s'", pathwayTag)
	}
	return nil
}

var (
	// TODO: these can easily get out of sync but left it here for now for simplciity
	validConditionalOps = map[string]bool{
		"==": true,
		"<":  true,
		">":  true,
		"in": true,
	}
	validConditionalKeys = map[string]bool{
		"gender": true,
		"state":  true,
		"age":    true,
	}
)

func validateConditional(c *common.Conditional) error {
	if !validConditionalOps[c.Op] {
		return fmt.Errorf("'%s' is not a valid conditional op", c.Op)
	}
	if !validConditionalKeys[c.Key] {
		return fmt.Errorf("'%s' is not a valid conditional key", c.Key)
	}
	if c.Value == nil {
		return fmt.Errorf("conditional value missing")
	}
	switch v := c.Value.(type) {
	default:
		return fmt.Errorf("%T is not a valid type for a conditional value", c.Value)
	case int, string, float64:
	case []interface{}:
		if len(v) == 0 {
			return fmt.Errorf("conditional value may not be an empty list")
		}
		switch v[0].(type) {
		default:
			return fmt.Errorf("[]%T is not a valid type of a conditional value", v[0])
		case int, string, float64:
		}
	}
	return nil
}
