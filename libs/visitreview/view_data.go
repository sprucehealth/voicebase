package visitreview

type PhotoData struct {
	Title          string `json:"title"`
	PhotoID        string `json:"photo_id"`
	PhotoURL       string `json:"photo_url"`
	PlaceholderURL string `json:"placeholder_url"`
}

type TitlePhotoListData struct {
	Title  string      `json:"title"`
	Photos []PhotoData `json:"photos"`
}

type MediaData struct {
	Title          string `json:"title"`
	MediaID        string `json:"media_id"`
	URL            string `json:"url"`
	ThumbnailURL   string `json:"thumbnail_url,omitempty"`
	Type           string `json:"type"`
	PlaceholderURL string `json:"placeholder_url"`
}

type TitleMediaListData struct {
	Title string      `json:"title"`
	Media []MediaData `json:"media"`
}

type CheckedUncheckedData struct {
	Value     string `json:"value"`
	IsChecked bool   `json:"is_checked"`
}

type TitleSubItemsDescriptionContentData struct {
	Title    string                    `json:"title"`
	SubItems []*DescriptionContentData `json:"subitems"`
}

type DescriptionContentData struct {
	Description string `json:"description"`
	Content     string `json:"content"`
}
