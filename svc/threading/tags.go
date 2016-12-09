package threading

import (
	"regexp"
	"strings"
)

// HiddenTagPrefix is the prefix attached to hidden tags to avoid collision with user tags.
const HiddenTagPrefix = "$"

// REValidTag matches valid tags. Only unicode letters, digits, dash, and underscore are allowed with an optional # as a prefix.
const RegexpValidTag = `^#?[\pL\d_-]+$`

var reValidTag = regexp.MustCompile(RegexpValidTag)

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
