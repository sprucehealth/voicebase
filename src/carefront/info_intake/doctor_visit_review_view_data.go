package info_intake

type PhotoData struct {
	Title          string `json:"title"`
	PhotoUrl       string `json:"photo_url"`
	PlaceholderUrl string `json:"placeholder_url"`
}

type TitlePhotoListData struct {
	Title  string      `json:"title"`
	Photos []PhotoData `json:"photos"`
}
type CheckedUncheckedData struct {
	Value     string `json:"value"`
	IsChecked bool   `json:"is_checked"`
}

type TitleSubItemsLabelContentData struct {
	Title    string              `json:"title"`
	SubItems []*LabelContentData `json:"subitems"`
}

type LabelContentData struct {
	Label   string `json:"key"`
	Content string `json:"value"`
}
