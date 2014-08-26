package httputil

import (
	"strings"
)

func ParseBool(s string) bool {
	switch strings.ToLower(s) {
	case "yes", "1", "true":
		return true
	}
	return false
}
