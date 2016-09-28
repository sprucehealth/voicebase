package media

import (
	"net/url"
	"strconv"
)

// URL returns a URL from which a client can request a particular media object.
func URL(mediaAPIDomain, mediaID, mimetype string) string {
	return mediaAPIDomain + "/media/" + mediaID + "?mimetype=" + url.QueryEscape(mimetype)
}

// ThumbnailURL populates a URL from which to access a thumbnail for a particular image.
func ThumbnailURL(mediaAPIDomain, mediaID, mimetype string, height, width int, crop bool) string {
	params := url.Values{}
	if height != 0 {
		params.Set("height", strconv.Itoa(height))
	}
	if width != 0 {
		params.Set("width", strconv.Itoa(width))
	}
	if crop {
		params.Set("crop", "true")
	}
	params.Set("mimetype", mimetype)
	tURL := mediaAPIDomain + "/media/" + mediaID + "/" + "thumbnail"
	if len(params) != 0 {
		tURL = tURL + "?" + params.Encode()
	}
	return tURL
}

// MIMEType returns the complete mimetype from the MIME object
func MIMEType(mType *MIME) string {
	return mType.Type + "/" + mType.Subtype
}
