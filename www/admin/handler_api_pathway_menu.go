package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
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

type pathwayMenuPostResponse struct {
	Success bool
	Error   string
}

func NewPathwayMenuHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&pathwayMenuHandler{
		dataAPI: dataAPI,
	}, []string{"GET", "PUT"})
}

func (h *pathwayMenuHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.get(w, r)
	case "PUT":
		h.put(w, r)
	}
}

func (h *pathwayMenuHandler) get(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "GetPathwayMenu", nil)
	menu, err := h.dataAPI.PathwayMenu()
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	www.JSONResponse(w, r, http.StatusOK, menu)
}

func (h *pathwayMenuHandler) put(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)
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
	www.JSONResponse(w, r, http.StatusOK, menu)
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
			if err := validatePathway(dataAPI, it.Pathway); err != nil {
				return err
			}
		}
	}
	return nil
}

func validatePathway(dataAPI api.DataAPI, pathway *common.Pathway) error {
	if pathway == nil {
		return fmt.Errorf("pathway not set")
	}
	if pathway.Tag == "" {
		return fmt.Errorf("pathway tag is required")
	}
	// TODO: this does one query per pathway which is very inefficient. Could optimize
	// to do in batch but it would complicate this code quite a bit. This is only
	// used when updating the menu from the admin so should be fine.
	p, err := dataAPI.PathwayForTag(pathway.Tag, api.PONone)
	if api.IsErrNotFound(err) {
		return fmt.Errorf("pathway with tag '%s' not found", pathway.Tag)
	} else if err != nil {
		golog.Errorf(err.Error())
		return fmt.Errorf("internal error checking pathway with tag '%s'", pathway.Tag)
	}
	// Fill in the rest of the pathway information
	pathway.ID = p.ID
	pathway.Name = p.Name
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
	switch c.Value.(type) {
	default:
		return fmt.Errorf("%T is not a valid type for a conditional value", c.Value)
	case int, string:
	}
	return nil
}
