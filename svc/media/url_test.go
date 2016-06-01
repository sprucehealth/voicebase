package media

import "testing"

func TestThumbnailURL(t *testing.T) {
	thumbURL := ThumbnailURL("http://test.com", "12345", 10, 10, false)
	if thumbURL != "http://test.com/media/12345/thumbnail?height=10&width=10" {
		t.Fatalf("thumbnail url %s does not match expected value", thumbURL)
	}
}
