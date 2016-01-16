package mediautils

import "fmt"

// URL is a utility function that generates the URL for media assets using an ID
func URL(apiDomain, mediaID string) string {
	return fmt.Sprintf("%s/media/%s", apiDomain, mediaID)
}

// ResizeURL is a utility function that generates the URL for media assets using an ID and desired dimensions
func ResizeURL(apiDomain, mediaID string, width, height int) string {
	return fmt.Sprintf("%s?width=%d&height=%d", URL(apiDomain, mediaID), width, height)
}
