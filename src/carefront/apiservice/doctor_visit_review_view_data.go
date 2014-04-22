package apiservice

type PhotoData struct {
	Title          string `json:"title"`
	PhotoUrl       string `json:"photo_url"`
	PlaceholderUrl string `json:"placeholder_url"`
}

type CheckedUncheckedData struct {
	Value     string `json:"value"`
	IsChecked bool   `json:"is_checked"`
}

type TitleSubtitleSubItemsData struct {
	Title    string   `json:"title"`
	Subtitle string   `json:"subtitle"`
	SubItems []string `json:"subitems"`
}
