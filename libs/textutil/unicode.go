package textutil

import (
	"unicode/utf8"
)

// TruncateUTF8 truncates the provided string if it's longer than the max length in runes (not bytes).
func TruncateUTF8(s string, maxLen int) string {
	// Shortcuts for the simple cases
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	// At this point we don't know if the number of runes is greater than the max
	for i := range s {
		if maxLen == 0 {
			return s[:i]
		}
		maxLen--
	}
	return s
}

// IsValidPlane0Unicode returns true iff the provided string only has valid plane 0 unicode (no emoji)
func IsValidPlane0Unicode(s string) bool {
	for _, r := range s {
		if !utf8.ValidRune(r) {
			return false
		}
		if utf8.RuneLen(r) > 3 {
			return false
		}
	}
	return true
}
