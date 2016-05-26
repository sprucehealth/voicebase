package media

// POSTResponse represents the data expected to be returned from a successful POST call to the media endpoint
type POSTResponse struct {
	MediaID string `json:"media_id"`
}

// VideoPOSTResponse represents the data expected to be returned from a successful POST call to the media/video endpoint
type VideoPOSTResponse struct {
	MediaID string `json:"media_id"`
}
