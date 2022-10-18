package voicebase

import "context"

type Client struct {
	Media MediaClient
}

var DefaultClient = getC()

func getC() *Client {
	return &Client{
		Media: &mediaClient{b: GetBackend()},
	}
}

// UploadMedia uploads a media to voicebase for transcribing.
func UploadMedia(ctx context.Context, url string) (string, error) {
	return DefaultClient.Media.Upload(ctx, url)
}

// GetMedia returns a media from voicebase with the appropriate ID.
func GetMedia(ctx context.Context, id string) (*Media, error) {
	return DefaultClient.Media.Get(ctx, id)
}

// DeleteMedia enables deleting of media on voicebase identified by its ID.
func DeleteMedia(ctx context.Context, id string) error {
	return DefaultClient.Media.Delete(ctx, id)
}

func (c *Client) Init(bearerToken string) {
	c.Media = &mediaClient{b: GetBackend(), bearerToken: bearerToken}
}
