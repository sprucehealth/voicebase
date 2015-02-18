package patient_visit

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
)

type pathwayMenuHandler struct {
	dataAPI api.DataAPI
}

type pathwayMenuNode struct {
	Title string       `json:"title"`
	Type  string       `json:"type"`
	Data  common.Typed `json:"data"`
}

type pathwayMenuContainer struct {
	Title              string                `json:"title"`
	Children           []*pathwayMenuNode    `json:"children"`
	BottomButtonTitle  string                `json:"bottom_button_title,omitempty"`
	BottomButtonTapURL *app_url.SpruceAction `json:"bottom_button_tap_url,omitempty"`
}

func (p *pathwayMenuContainer) TypeName() string {
	return "container"
}

type pathwayMenuPathway struct {
	Tag string `json:"id"`
}

func (p *pathwayMenuPathway) TypeName() string {
	return "pathway"
}

func NewPathwayMenuHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(&pathwayMenuHandler{
			dataAPI: dataAPI,
		}),
		[]string{"GET"})
}

func (h *pathwayMenuHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := apiservice.GetContext(r)

	menu, err := h.dataAPI.PathwayMenu()
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// Set default context values for non-authenticated requests
	menuCtx := map[string]interface{}{
		"age":    0,
		"gender": "",
		"state":  r.FormValue("state"),
	}

	var patient *common.Patient
	if ctx.AccountID != 0 && ctx.Role == api.PATIENT_ROLE {
		patient, err = h.dataAPI.GetPatientFromAccountID(ctx.AccountID)
		if err != nil && !api.IsErrNotFound(err) {
			apiservice.WriteError(err, w, r)
			return
		}
	}
	if patient != nil {
		menuCtx["age"] = patient.DOB.Age()
		menuCtx["gender"] = patient.Gender
		menuCtx["state"] = patient.StateFromZipCode
	}

	container, err := transformMenu(menuCtx, menu)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if patient != nil {
		// only include bottom button for new users (no submitted visit)
		if sv, err := h.dataAPI.AnyVisitSubmitted(patient.PatientID.Int64()); err != nil {
			golog.Errorf(err.Error())
		} else if !sv {
			container.BottomButtonTitle = "Not ready to start a visit yet?"
			container.BottomButtonTapURL = app_url.ViewHomeAction()
		}
	}

	root := &pathwayMenuNode{
		Title: "",
		Type:  container.TypeName(),
		Data:  container,
	}

	httputil.JSONResponse(w, http.StatusOK, root)
}

func transformMenu(ctx map[string]interface{}, menu *common.PathwayMenu) (*pathwayMenuContainer, error) {
	container := &pathwayMenuContainer{
		Title:    menu.Title,
		Children: make([]*pathwayMenuNode, 0, len(menu.Items)),
	}
	for _, it := range menu.Items {
		if matched, err := matchesConditionals(ctx, it.Conditionals); err != nil {
			return nil, err
		} else if !matched {
			continue
		}
		node, err := transformMenuItem(ctx, it)
		if err != nil {
			return nil, err
		}
		container.Children = append(container.Children, node)
	}
	return container, nil
}

func transformPathway(ctx map[string]interface{}, pathwayTag string) (*pathwayMenuPathway, error) {
	return &pathwayMenuPathway{
		Tag: pathwayTag,
	}, nil
}

func transformMenuItem(ctx map[string]interface{}, item *common.PathwayMenuItem) (*pathwayMenuNode, error) {
	var err error
	var data common.Typed
	switch item.Type {
	default:
		return nil, fmt.Errorf("unknown pathway menu item type '%s'", item.Type)
	case common.PathwayMenuItemTypeMenu:
		data, err = transformMenu(ctx, item.Menu)
	case common.PathwayMenuItemTypePathway:
		data, err = transformPathway(ctx, item.PathwayTag)
	}
	if err != nil {
		return nil, err
	}
	return &pathwayMenuNode{
		Title: item.Title,
		Type:  data.TypeName(),
		Data:  data,
	}, nil
}

func matchesConditionals(ctx map[string]interface{}, cond []*common.Conditional) (bool, error) {
	if len(cond) == 0 {
		return true, nil
	}
	for _, c := range cond {
		v := ctx[c.Key]
		if v == nil {
			return false, fmt.Errorf("no context value for key '%s'", c.Key)
		}
		if c.Value == nil {
			return false, fmt.Errorf("condition value is nil for key '%s'", c.Key)
		}
		switch c.Op {
		default:
			return false, fmt.Errorf("unknown condition op '%s'", c.Op)
		case "==":
			if b, err := isEqual(v, c.Value); b == c.Not || err != nil {
				return false, err
			}
		case "<":
			if b, err := isLessThan(v, c.Value); b == c.Not || err != nil {
				return false, err
			}
		case ">":
			if b, err := isGreaterThan(v, c.Value); b == c.Not || err != nil {
				return false, err
			}
		case "in":
			if b, err := isIn(v, c.Value); b == c.Not || err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func isEqual(v1, v2 interface{}) (bool, error) {
	switch v := v1.(type) {
	case int:
		// Check both int and float64 since this is coming back
		// from JSON which only has floats (still check int thought).
		if i, ok := v2.(int); ok {
			return i == v, nil
		}
		if f, ok := v2.(float64); ok {
			return int(f) == v, nil
		}
		return false, fmt.Errorf("mismatched type '%T' for equality condition, expected number", v2)
	case string:
		if s, ok := v2.(string); ok {
			return strings.EqualFold(s, v), nil
		}
		return false, fmt.Errorf("mismatched type '%T' for equality condition, expected string", v2)
	}
	return false, fmt.Errorf("unsupported conditional value type %T", v1)
}

func isLessThan(v1, v2 interface{}) (bool, error) {
	switch v := v1.(type) {
	case int:
		// Check both int and float64 since this is coming back
		// from JSON which only has floats (still check int thought).
		if i, ok := v2.(int); ok {
			return v < i, nil
		}
		if f, ok := v2.(float64); ok {
			return v < int(f), nil
		}
		return false, fmt.Errorf("mismatched type '%T' for less than condition, expected number", v2)
	case string:
		if s, ok := v2.(string); ok {
			return v < s, nil
		}
		return false, fmt.Errorf("mismatched type '%T' for less than condition, expected string", v2)
	}
	return false, fmt.Errorf("unsupported conditional value type %T", v1)
}

func isGreaterThan(v1, v2 interface{}) (bool, error) {
	switch v := v1.(type) {
	case int:
		// Check both int and float64 since this is coming back
		// from JSON which only has floats (still check int thought).
		if i, ok := v2.(int); ok {
			return v > i, nil
		}
		if f, ok := v2.(float64); ok {
			return v > int(f), nil
		}
		return false, fmt.Errorf("mismatched type '%T' for greater than condition, expected number", v2)
	case string:
		if s, ok := v2.(string); ok {
			return v > s, nil
		}
		return false, fmt.Errorf("mismatched type '%T' for greater than condition, expected string", v2)
	}
	return false, fmt.Errorf("unsupported conditional value type %T", v1)
}

func isIn(v1, v2 interface{}) (bool, error) {
	switch v := v1.(type) {
	case int:
		// Check both int and float64 since this is coming back
		// from JSON which only has floats (still check int thought).
		if si, ok := v2.([]int); ok {
			for _, i := range si {
				if i == v {
					return true, nil
				}
			}
			return false, nil
		}
		if sf, ok := v2.([]float64); ok {
			for _, f := range sf {
				if int(f) == v {
					return true, nil
				}
			}
			return false, nil
		}
		if si, ok := v2.([]interface{}); ok {
			for _, f := range si {
				switch v2 := f.(type) {
				case int:
					if v2 == v {
						return true, nil
					}
				case float64:
					if int(v2) == v {
						return true, nil
					}
				default:
					return false, fmt.Errorf("mismatched type '%T' for equality condition, expected number", v2)
				}
			}
			return false, nil
		}
		return false, fmt.Errorf("mismatched type '%T' for equality condition, expected []number", v2)
	case string:
		if ss, ok := v2.([]string); ok {
			for _, s := range ss {
				if strings.EqualFold(s, v) {
					return true, nil
				}
			}
			return false, nil
		}
		if si, ok := v2.([]interface{}); ok {
			for _, ss := range si {
				if s, ok := ss.(string); ok {
					if strings.EqualFold(s, v) {
						return true, nil
					}
				} else {
					return false, fmt.Errorf("mismatched type '%T' for equality condition, expected string", v2)
				}
			}
			return false, nil
		}
		return false, fmt.Errorf("mismatched type '%T' for equality condition, expected []string", v2)
	}
	return false, fmt.Errorf("unsupported conditional value type %T", v1)
}
