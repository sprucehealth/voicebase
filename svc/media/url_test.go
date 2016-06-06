package media

import "testing"

func TestURL(t *testing.T) {
	url := URL("http://test.com", "12345")
	if url != "http://test.com/media/12345" {
		t.Fatalf("url %s does not match expected value", url)
	}
}

func TestThumbnailURL(t *testing.T) {
	thumbURL := ThumbnailURL("http://test.com", "12345", 10, 10, false)
	if thumbURL != "http://test.com/media/12345/thumbnail?height=10&width=10" {
		t.Fatalf("thumbnail url %s does not match expected value", thumbURL)
	}
}
