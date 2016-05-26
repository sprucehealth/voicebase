package media

import (
	"net/url"
	"strconv"
)

func URL(mediaAPIDomain, mediaID string) string {
	return mediaAPIDomain + "/" + mediaID
}

func ThumbnailURL(mediaAPIDomain, mediaID string, height, width int, crop bool) string {
	var params url.Values
	if height != 0 {
		params.Set("height", strconv.Itoa(height))
	}
	if width != 0 {
		params.Set("width", strconv.Itoa(width))
	}
	if crop {
		params.Set("crop", "true")
	}
	tURL := mediaAPIDomain + "/" + mediaID + "/" + "thumbnail"
	if len(params) != 0 {
		tURL = tURL + "?" + params.Encode()
	}
	return tURL
}
