package info_intake

// View definitions

type Context map[string]interface{}

type View interface {
	Render(context Context) (map[string]interface{}, error)
}

type ViewValidationError struct {
	Message string
}

func (v ViewValidationError) Error() string {
	return v.Message
}

type ViewCondition struct {
	Op  string `json:"op"`
	Key string `json:"key"`
}

type SectionListView struct {
	Sections []View `json:"sections"`
}

func (s SectionListView) TypeName() string {
	return "sections_list"
}

func (s SectionListView) Render(context Context) (map[string]interface{}, error) {
	// TODO
	return nil, nil
}

type StandardPhotosSectionView struct {
	Title       string `json:"title"`
	Subsections []View `json:"subsections"`
}

func (s StandardPhotosSectionView) TypeName() string {
	return "standard_photo_section"
}

func (s StandardPhotosSectionView) Render(context Context) (map[string]interface{}, error) {
	// TODO
	return nil, nil
}

type StandardPhotosSubsectionView struct {
	SubsectionView View `json:"view"`
}

func (s StandardPhotosSubsectionView) TypeName() string {
	return "standard_photo_subsection"
}

func (s StandardPhotosSubsectionView) Render(context Context) (map[string]interface{}, error) {
	// TODO
	return nil, nil
}

type StandardPhotosListView struct {
	Photos        []PhotoData `json:"photos"`
	ContentConfig struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (s StandardPhotosListView) TypeName() string {
	return "standard_photos_list"
}

func (s StandardPhotosListView) Render(context Context) (map[string]interface{}, error) {
	// TODO
	return nil, nil
}

type StandardSectionView struct {
	Title       string `json:"title"`
	Subsections []View `json:"subsections"`
}

func (s StandardSectionView) TypeName() string {
	return "standard_section"
}

func (s StandardSectionView) Render(context Context) (map[string]interface{}, error) {
	// TODO
	return nil, nil
}

type StandardSubsectionView struct {
	Title string `json:"title"`
	Rows  []View `json:"rows"`
}

func (s StandardSubsectionView) TypeName() string {
	return "standard_subsection"
}

func (s StandardSubsectionView) Render(context Context) (map[string]interface{}, error) {
	// TODO
	return nil, nil
}

type StandardOneColumnRowView struct {
	SingleView View `json:"view"`
}

func (s StandardOneColumnRowView) TypeName() string {
	return "standard_one_column_row"
}

func (s StandardOneColumnRowView) Render(context Context) (map[string]interface{}, error) {
	// TODO
	return nil, nil
}

type StandardTwoColumnRowView struct {
	LeftView      View `json:"left_view"`
	RightView     View `json:"right_view"`
	ContentConfig struct {
		ViewCondition `json:"condition"`
	} `json:"content_config"`
}

func (s StandardTwoColumnRowView) TypeName() string {
	return "standard_two_column_row"
}

func (s StandardTwoColumnRowView) Render(context Context) (map[string]interface{}, error) {
	// TODO
	return nil, nil
}

type DividedViewsList struct {
	DividedViews []View `json:"views"`
}

func (d DividedViewsList) TypeName() string {
	return "divided_views_list"
}

func (d DividedViewsList) Render(context Context) (map[string]interface{}, error) {
	// TODO
	return nil, nil
}

type AlertLabelsList struct {
	Values        []string `json:"values"`
	ContentConfig struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (a AlertLabelsList) TypeName() string {
	return "alert_labels_list"
}

func (a AlertLabelsList) Render(context Context) (map[string]interface{}, error) {
	// TODO
	return nil, nil
}

type TitleLabelsList struct {
	Values        []string `json:"values"`
	ContentConfig struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (t TitleLabelsList) TypeName() string {
	return "title_labels_list"
}

func (t TitleLabelsList) Render(context Context) (map[string]interface{}, error) {
	// TODO
	return nil, nil
}

type ContentLabelsList struct {
	Values        []string `json:"values"`
	ContentConfig struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (c ContentLabelsList) TypeName() string {
	return "content_labels_list"
}

func (c ContentLabelsList) Render(context Context) (map[string]interface{}, error) {
	// TODO
	return nil, nil
}

type CheckXItemsList struct {
	Items         []CheckedUncheckedData `json:"items"`
	ContentConfig struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (c CheckXItemsList) TypeName() string {
	return "check_x_items_list"
}

func (c CheckXItemsList) Render(context Context) (map[string]interface{}, error) {
	// TODO
	return nil, nil
}

type TitleSubtitleSubItemsDividedItemsList struct {
	Items         []TitleSubtitleSubItemsData `json:"items"`
	ContentConfig struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (t TitleSubtitleSubItemsDividedItemsList) TypeName() string {
	return "title_subtitle_subitems_divided_items_list"
}

func (t TitleSubtitleSubItemsDividedItemsList) Render(context Context) (map[string]interface{}, error) {
	// TODO
	return nil, nil
}

type TitleSubtitleLabels struct {
	Title         string `json:"title"`
	Subtitle      string `json:"subtitle"`
	ContentConfig struct {
		TitleKey      string `json:"title_key"`
		SubtitleKey   string `json:"subtitle_key"`
		ViewCondition `json:"condition"`
	} `json:"content_config"`
}

func (t TitleSubtitleLabels) TypeName() string {
	return "title_subtitle_labels"
}

func (t TitleSubtitleLabels) Render(context Context) (map[string]interface{}, error) {
	// TODO
	return nil, nil
}
