package server

// truncateUTF8 truncates the provided string if it's longer than the max length in runes (not bytes).
func truncateUTF8(s string, maxLen int) string {
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
