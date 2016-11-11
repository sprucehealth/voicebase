package threading

import (
	"regexp"
	"strings"
)

// HiddenTagPrefix is the prefix attached to hidden tags to avoid collision with user tags.
const HiddenTagPrefix = "$"

// reValidTag matches valid tags. Only unicode letters, digits, dash, and underscore are allowed with an optional # as a prefix.
var reValidTag = regexp.MustCompile(`^#?[\pL\d_-]+$`)

// ValidateTag returns true if the provided tag is valid.
func ValidateTag(tag string, allowHidden bool) bool {
	if allowHidden && strings.HasPrefix(tag, HiddenTagPrefix) {
		tag = tag[len(HiddenTagPrefix):]
	}
	if len(tag) == 0 {
		return false
	}
	return reValidTag.MatchString(tag)
}
