package voicebase

type Client struct {
	Media MediaClient
}

var defaultClient = getC()

func getC() *Client {
	return &Client{
		Media: &mediaClient{b: GetBackend()},
	}
}

// UploadMedia uploads a media to voicebase for transcribing.
func UploadMedia(url string) (string, error) {
	return defaultClient.Media.Upload(url)
}

// GetMedia returns a media from voicebase with the appropriate ID.
func GetMedia(id string) (*Media, error) {
	return defaultClient.Media.Get(id)
}

// DeleteMedia enables deleting of media on voicebase identified by its ID.
func DeleteMedia(id string) error {
	return defaultClient.Media.Delete(id)
}
