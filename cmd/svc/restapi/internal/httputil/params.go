package httputil

import (
	"fmt"
	"net/http"
	"strconv"
)

func ParseFormInt(r *http.Request, formKey string) (int, error) {
	var v int
	var err error
	if vStr := r.FormValue(formKey); vStr != "" {
		v, err = strconv.Atoi(vStr)
		if err != nil {
			return 0, fmt.Errorf("Unable to parse %s %s: %s", formKey, vStr, err)
		}
	}
	return v, nil
}

func ParseFormBool(r *http.Request, formKey string) (bool, error) {
	var v bool
	var err error
	if vStr := r.FormValue(formKey); vStr != "" {
		v, err = strconv.ParseBool(vStr)
		if err != nil {
			return false, fmt.Errorf("Unable to parse %s %s: %s", formKey, vStr, err)
		}
	}
	return v, nil
}
