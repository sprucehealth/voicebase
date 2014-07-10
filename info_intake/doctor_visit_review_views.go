package info_intake

import (
	"fmt"

	"github.com/sprucehealth/backend/common"

	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/mapstructure"
)

// This TypeRegistry contains all views pertaining to the DVisitReview namespace
var DVisitReviewViewTypeRegistry *mapstructure.TypeRegistry = mapstructure.NewTypeRegistry()

func init() {
	DVisitReviewViewTypeRegistry.MustRegisterType(&DVisitReviewSectionListView{})
	DVisitReviewViewTypeRegistry.MustRegisterType(&DVisitReviewStandardPhotosSectionView{})
	DVisitReviewViewTypeRegistry.MustRegisterType(&DVisitReviewStandardPhotosSubsectionView{})
	DVisitReviewViewTypeRegistry.MustRegisterType(&DVisitReviewStandardPhotosListView{})
	DVisitReviewViewTypeRegistry.MustRegisterType(&DVisitReviewStandardSectionView{})
	DVisitReviewViewTypeRegistry.MustRegisterType(&DVisitReviewStandardSubsectionView{})
	DVisitReviewViewTypeRegistry.MustRegisterType(&DVisitReviewStandardOneColumnRowView{})
	DVisitReviewViewTypeRegistry.MustRegisterType(&DVisitReviewStandardTwoColumnRowView{})
	DVisitReviewViewTypeRegistry.MustRegisterType(&DVisitReviewDividedViewsList{})
	DVisitReviewViewTypeRegistry.MustRegisterType(&DVisitReviewAlertLabelsList{})
	DVisitReviewViewTypeRegistry.MustRegisterType(&DVisitReviewTitleLabelsList{})
	DVisitReviewViewTypeRegistry.MustRegisterType(&DVisitReviewContentLabelsList{})
	DVisitReviewViewTypeRegistry.MustRegisterType(&DVisitReviewCheckXItemsList{})
	DVisitReviewViewTypeRegistry.MustRegisterType(&DVisitReviewTitleSubtitleLabels{})
	DVisitReviewViewTypeRegistry.MustRegisterType(&DVisitReviewEmptyLabelView{})
	DVisitReviewViewTypeRegistry.MustRegisterType(&DVisitReviewEmptyTitleSubtitleLabelView{})
	DVisitReviewViewTypeRegistry.MustRegisterType(&DVisitReviewTitlePhotosItemsListView{})
	DVisitReviewViewTypeRegistry.MustRegisterType(&DVisitReviewTitleSubItemsLabelContentItemsList{})
}

// View definitions

type DVisitReviewSectionListView struct {
	Sections []common.View `json:"sections"`
}

func (d *DVisitReviewSectionListView) TypeName() string {
	return wrapNamespace("sections_list")
}

func (d *DVisitReviewSectionListView) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	renderedSections := make([]interface{}, 0)
	for _, section := range d.Sections {
		renderedSection, err := section.Render(context)
		if err != nil {
			return nil, err
		}

		if renderedSection != nil {
			renderedSections = append(renderedSections, renderedSection)
		}
	}

	renderedView["type"] = d.TypeName()
	renderedView["sections"] = renderedSections
	return renderedView, nil
}

type DVisitReviewStandardPhotosSectionView struct {
	Title       string        `json:"title"`
	Subsections []common.View `json:"subsections"`
}

func (d *DVisitReviewStandardPhotosSectionView) TypeName() string {
	return wrapNamespace("standard_photo_section")
}

func (d *DVisitReviewStandardPhotosSectionView) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	renderedSubSections := make([]interface{}, 0)

	for _, subSection := range d.Subsections {
		var err error
		renderedSubSection, err := subSection.Render(context)
		if err != nil {
			return nil, err
		}

		if renderedSubSection != nil {
			renderedSubSections = append(renderedSubSections, renderedSubSection)
		}
	}

	renderedView["title"] = d.Title
	renderedView["type"] = d.TypeName()
	renderedView["subsections"] = renderedSubSections

	return renderedView, nil
}

type DVisitReviewStandardPhotosSubsectionView struct {
	SubsectionView common.View `json:"view"`
}

func (d *DVisitReviewStandardPhotosSubsectionView) TypeName() string {
	return wrapNamespace("standard_photo_subsection")
}

func (d *DVisitReviewStandardPhotosSubsectionView) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})

	if d.SubsectionView != nil {
		renderedSubsectionView, err := d.SubsectionView.Render(context)
		if err != nil {
			return nil, err
		}
		if renderedSubsectionView != nil {
			renderedView["view"] = renderedSubsectionView
		}
	}
	renderedView["type"] = d.TypeName()

	return renderedView, nil
}

type DVisitReviewStandardPhotosListView struct {
	Photos        []PhotoData `json:"photos"`
	ContentConfig struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (d *DVisitReviewStandardPhotosListView) TypeName() string {
	return wrapNamespace("standard_photos_list")
}

func (d *DVisitReviewStandardPhotosListView) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})

	content, err := getContentFromContextForView(d, d.ContentConfig.Key, context)
	if err != nil {
		return nil, err
	}

	var ok bool
	d.Photos, ok = content.([]PhotoData)
	if !ok {
		return nil, common.NewViewRenderingError(fmt.Sprintf("Expected content in view context to be of type []PhotoData instead it was type %T", content))
	}

	renderedView["photos"] = d.Photos
	renderedView["type"] = d.TypeName()

	return renderedView, nil
}

type DVisitReviewTitlePhotosItemsListView struct {
	Items         []TitlePhotoListData `json:"items"`
	ContentConfig struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (d *DVisitReviewTitlePhotosItemsListView) TypeName() string {
	return wrapNamespace("title_photos_items_list")
}

func (d *DVisitReviewTitlePhotosItemsListView) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})

	content, err := getContentFromContextForView(d, d.ContentConfig.Key, context)
	if err != nil {
		return nil, err
	}

	var ok bool
	d.Items, ok = content.([]TitlePhotoListData)
	if !ok {
		return nil, common.NewViewRenderingError(fmt.Sprintf("Expected content in view context to be of type []TitlePhotoListData instead it was type %T", content))
	}

	renderedView["items"] = d.Items
	renderedView["type"] = d.TypeName()

	return renderedView, nil
}

type DVisitReviewStandardSectionView struct {
	Title       string        `json:"title"`
	Subsections []common.View `json:"subsections"`
}

func (d *DVisitReviewStandardSectionView) TypeName() string {
	return wrapNamespace("standard_section")
}

func (d *DVisitReviewStandardSectionView) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	renderedSubsections := make([]interface{}, 0)

	for _, subsection := range d.Subsections {
		renderedSubsection, err := subsection.Render(context)
		if err != nil {
			return nil, err
		}
		if renderedSubsection != nil {
			renderedSubsections = append(renderedSubsections, renderedSubsection)
		}
	}
	renderedView["type"] = d.TypeName()
	renderedView["title"] = d.Title
	renderedView["subsections"] = renderedSubsections
	return renderedView, nil
}

type DVisitReviewStandardSubsectionView struct {
	Title         string        `json:"title"`
	Rows          []common.View `json:"rows"`
	ContentConfig struct {
		common.ViewCondition `json:"condition"`
	} `json:"content_config"`
}

func (d *DVisitReviewStandardSubsectionView) TypeName() string {
	return wrapNamespace("standard_subsection")
}

func (d *DVisitReviewStandardSubsectionView) Render(context common.ViewContext) (map[string]interface{}, error) {
	if d.ContentConfig.ViewCondition.Op != "" {
		if result, err := common.EvaluateConditionForView(d, d.ContentConfig.ViewCondition, context); err != nil || !result {
			return nil, err
		}
	}
	renderedView := make(map[string]interface{})
	renderedRows := make([]interface{}, 0)

	for _, row := range d.Rows {
		renderedRow, err := row.Render(context)
		if err != nil {
			return nil, err
		}
		if renderedRow != nil {
			renderedRows = append(renderedRows, renderedRow)
		}
	}
	renderedView["type"] = d.TypeName()
	renderedView["title"] = d.Title
	renderedView["rows"] = renderedRows

	return renderedView, nil
}

type DVisitReviewStandardOneColumnRowView struct {
	SingleView common.View `json:"view"`
}

func (d *DVisitReviewStandardOneColumnRowView) TypeName() string {
	return wrapNamespace("standard_one_column_row")
}

func (d *DVisitReviewStandardOneColumnRowView) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	if d.SingleView != nil {
		renderedSingleView, err := d.SingleView.Render(context)
		if err != nil {
			return nil, err
		}
		if renderedSingleView != nil {
			renderedView["view"] = renderedSingleView
		}
	}
	renderedView["type"] = d.TypeName()
	return renderedView, nil
}

type DVisitReviewStandardTwoColumnRowView struct {
	LeftView      common.View `json:"left_view"`
	RightView     common.View `json:"right_view"`
	ContentConfig struct {
		common.ViewCondition `json:"condition"`
	} `json:"content_config"`
}

func (d *DVisitReviewStandardTwoColumnRowView) TypeName() string {
	return wrapNamespace("standard_two_column_row")
}

func (d *DVisitReviewStandardTwoColumnRowView) Render(context common.ViewContext) (map[string]interface{}, error) {
	if d.ContentConfig.ViewCondition.Op != "" {
		if result, err := common.EvaluateConditionForView(d, d.ContentConfig.ViewCondition, context); err != nil || !result {
			return nil, err
		}
	}
	renderedView := make(map[string]interface{})
	if d.LeftView != nil {
		renderedLeftView, err := d.LeftView.Render(context)
		if err != nil {
			return nil, err
		}
		if renderedLeftView != nil {
			renderedView["left_view"] = renderedLeftView
		}
	}

	if d.RightView != nil {
		renderedRightView, err := d.RightView.Render(context)
		if err != nil {
			return nil, err
		}
		if renderedRightView != nil {
			renderedView["right_view"] = renderedRightView
		}
	}
	renderedView["type"] = d.TypeName()
	return renderedView, nil
}

type DVisitReviewDividedViewsList struct {
	DividedViews []common.View `json:"views"`
}

func (d *DVisitReviewDividedViewsList) TypeName() string {
	return wrapNamespace("divided_views_list")
}

func (d *DVisitReviewDividedViewsList) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	renderedView["type"] = d.TypeName()
	renderedSubviews := make([]interface{}, 0)
	for _, dividedView := range d.DividedViews {
		renderedSubview, err := dividedView.Render(context)
		if err != nil {
			return nil, err
		}
		if renderedSubview != nil {
			renderedSubviews = append(renderedSubviews, renderedSubview)
		}
	}
	renderedView["views"] = renderedSubviews
	return renderedView, nil
}

type DVisitReviewAlertLabelsList struct {
	Values         []string    `json:"values"`
	EmptyStateView common.View `json:"empty_state_view"`
	ContentConfig  struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (d *DVisitReviewAlertLabelsList) TypeName() string {
	return wrapNamespace("alert_labels_list")
}

func (d *DVisitReviewAlertLabelsList) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	var err error
	d.Values, err = getStringArrayFromContext(d, d.ContentConfig.Key, context)

	if err != nil {
		if d.EmptyStateView != nil {
			return handleRenderingOfEmptyStateViewOnError(err, d.EmptyStateView, d, context)
		}
		return nil, err
	}

	renderedView["type"] = d.TypeName()
	renderedView["values"] = d.Values

	return renderedView, nil
}

type DVisitReviewTitleLabelsList struct {
	Values        []string `json:"values"`
	ContentConfig struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (d *DVisitReviewTitleLabelsList) TypeName() string {
	return wrapNamespace("title_labels_list")
}

func (d *DVisitReviewTitleLabelsList) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	var err error

	content, err := getContentFromContextForView(d, d.ContentConfig.Key, context)
	if err != nil {
		return nil, err
	}

	switch content.(type) {
	case string:
		d.Values = []string{content.(string)}
	case []string:
		d.Values = content.([]string)
	default:
		return nil, common.NewViewRenderingError(fmt.Sprintf("Expected content to be either string or []string for view type %s", d.TypeName()))
	}

	renderedView["type"] = d.TypeName()
	renderedView["values"] = d.Values
	return renderedView, nil
}

type DVisitReviewContentLabelsList struct {
	Values         []string    `json:"values"`
	EmptyStateView common.View `json:"empty_state_view"`
	ContentConfig  struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (d *DVisitReviewContentLabelsList) TypeName() string {
	return wrapNamespace("content_labels_list")
}

func (d *DVisitReviewContentLabelsList) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})

	content, err := getContentFromContextForView(d, d.ContentConfig.Key, context)
	if err != nil {
		if d.EmptyStateView != nil {
			return handleRenderingOfEmptyStateViewOnError(err, d.EmptyStateView, d, context)
		}
		return nil, err
	}

	switch content.(type) {
	case string:
		d.Values = []string{content.(string)}
	case []string:
		d.Values = content.([]string)
	case []CheckedUncheckedData:
		// read the checked items to populate the content list
		items := content.([]CheckedUncheckedData)
		strItems := make([]string, 0)
		for _, item := range items {
			if item.IsChecked {
				strItems = append(strItems, item.Value)
			}
		}
		d.Values = strItems
	case []TitleSubItemsDescriptionContentData:
		// read the checked items to populate the content list
		items := content.([]TitleSubItemsDescriptionContentData)
		strItems := make([]string, 0)
		for _, item := range items {
			strItems = append(strItems, item.Title)
		}
		d.Values = strItems
	default:
		return nil, common.NewViewRenderingError(fmt.Sprintf("Expected content to be either string, []string, []CheckedUnCheckedData or []TitleSubtitleSubitemsData for view type %s and key %s but was %T", d.TypeName(), d.ContentConfig.Key, content))
	}

	renderedView["type"] = d.TypeName()
	renderedView["values"] = d.Values
	return renderedView, nil
}

type DVisitReviewCheckXItemsList struct {
	Items         []CheckedUncheckedData `json:"items"`
	ContentConfig struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (d *DVisitReviewCheckXItemsList) TypeName() string {
	return wrapNamespace("check_x_items_list")
}

func (d *DVisitReviewCheckXItemsList) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	content, err := getContentFromContextForView(d, d.ContentConfig.Key, context)
	if err != nil {
		return nil, err
	}

	checkedUncheckedItems, ok := content.([]CheckedUncheckedData)
	if !ok {
		return nil, common.NewViewRenderingError(fmt.Sprintf("Expected content of type []CheckedUncheckedData for view type %s and key %s", d.TypeName(), d.ContentConfig.Key))
	}
	d.Items = checkedUncheckedItems
	renderedView["items"] = checkedUncheckedItems
	renderedView["type"] = d.TypeName()

	return renderedView, nil
}

type DVisitReviewTitleSubItemsLabelContentItemsList struct {
	Items          []TitleSubItemsDescriptionContentData `json:"items"`
	EmptyStateView common.View                           `json:"empty_state_view"`
	ContentConfig  struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (d *DVisitReviewTitleSubItemsLabelContentItemsList) TypeName() string {
	return wrapNamespace("title_subitems_description_content_labels_divided_items_list")
}

func (d *DVisitReviewTitleSubItemsLabelContentItemsList) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	content, err := getContentFromContextForView(d, d.ContentConfig.Key, context)
	if err != nil {
		if d.EmptyStateView != nil {
			return handleRenderingOfEmptyStateViewOnError(err, d.EmptyStateView, d, context)
		}

		return nil, err
	}

	items, ok := content.([]TitleSubItemsDescriptionContentData)
	if !ok {
		return nil, common.NewViewRenderingError(fmt.Sprintf("Expected content of type []TitleSubItemsDescriptionContentData for view type %s", d.TypeName()))
	}
	d.Items = items
	renderedView["items"] = items
	renderedView["type"] = d.TypeName()

	return renderedView, nil
}

type DVisitReviewTitleSubtitleLabels struct {
	Title          string      `json:"title"`
	Subtitle       string      `json:"subtitle"`
	EmptyStateView common.View `json:"empty_state_view"`
	ContentConfig  struct {
		TitleKey             string `json:"title_key"`
		SubtitleKey          string `json:"subtitle_key"`
		common.ViewCondition `json:"condition"`
	} `json:"content_config"`
}

func (d *DVisitReviewTitleSubtitleLabels) TypeName() string {
	return wrapNamespace("title_subtitle_labels")
}

func (d *DVisitReviewTitleSubtitleLabels) Render(context common.ViewContext) (map[string]interface{}, error) {
	if d.ContentConfig.ViewCondition.Op != "" {
		if result, err := common.EvaluateConditionForView(d, d.ContentConfig.ViewCondition, context); err != nil || !result {
			return nil, err
		}
	}

	renderedView := make(map[string]interface{})
	var err error

	d.Title, err = getStringFromContext(d, d.ContentConfig.TitleKey, context)
	if err != nil {
		if d.EmptyStateView != nil {
			return handleRenderingOfEmptyStateViewOnError(err, d.EmptyStateView, d, context)
		}
		return nil, err
	}

	d.Subtitle, err = getStringFromContext(d, d.ContentConfig.SubtitleKey, context)
	if err != nil {
		if d.EmptyStateView != nil {
			return handleRenderingOfEmptyStateViewOnError(err, d.EmptyStateView, d, context)
		}
		return nil, err
	}

	renderedView["title"] = d.Title
	renderedView["subtitle"] = d.Subtitle
	renderedView["type"] = d.TypeName()
	return renderedView, nil
}

type DVisitReviewEmptyLabelView struct {
	Text          string
	ContentConfig struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (d *DVisitReviewEmptyLabelView) TypeName() string {
	return wrapNamespace("empty_label")
}

func (d *DVisitReviewEmptyLabelView) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	text, err := getStringFromContext(d, d.ContentConfig.Key, context)
	if err != nil {
		return nil, err
	}

	renderedView["text"] = text
	renderedView["type"] = d.TypeName()
	return renderedView, nil
}

type DVisitReviewEmptyTitleSubtitleLabelView struct {
	Title         string
	Subtitle      string
	ContentConfig struct {
		TitleKey    string `json:"title_key"`
		SubtitleKey string `json:"subtitle_key"`
	} `json:"content_config"`
}

func (d *DVisitReviewEmptyTitleSubtitleLabelView) TypeName() string {
	return wrapNamespace("empty_title_subtitle_labels")
}

func (d *DVisitReviewEmptyTitleSubtitleLabelView) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	title, err := getStringFromContext(d, d.ContentConfig.TitleKey, context)
	if err != nil {
		return nil, err
	}

	subtitle, err := getStringFromContext(d, d.ContentConfig.SubtitleKey, context)
	if err != nil {
		return nil, err
	}

	renderedView["type"] = d.TypeName()
	renderedView["title"] = title
	renderedView["subtitle"] = subtitle
	return renderedView, nil
}

func wrapNamespace(viewtype string) string {
	return fmt.Sprintf("d_visit_review:%s", viewtype)
}

func getStringFromContext(view common.View, key string, context common.ViewContext) (string, error) {
	content, err := getContentFromContextForView(view, key, context)
	if err != nil {
		return "", err
	}

	str, ok := content.(string)
	if !ok {
		return "", common.NewViewRenderingError(fmt.Sprintf("Expected string for content of view type %s instead got %T for key %s", view.TypeName(), content, key))
	}

	return str, nil
}

func getStringArrayFromContext(view common.View, key string, context common.ViewContext) ([]string, error) {

	content, err := getContentFromContextForView(view, key, context)
	if err != nil {
		return nil, err
	}

	stringArray, ok := content.([]string)
	if !ok {
		return nil, common.NewViewRenderingError(fmt.Sprintf("Expected []string for content of view type %s instead got %T", view.TypeName(), content))
	}

	return stringArray, nil
}

func getContentFromContextForView(view common.View, key string, context common.ViewContext) (interface{}, error) {
	if key == "" {
		return nil, common.NewViewRenderingError(fmt.Sprintf("Content config key not specified for view type %s", view.TypeName()))
	}

	content, ok := context.Get(key)
	if !ok {
		return nil, &common.ViewRenderingError{
			Message:          fmt.Sprintf("Content with key %s not found in view context for view type %s", key, view.TypeName()),
			IsContentMissing: true,
		}
	}

	return content, nil
}

func handleRenderingOfEmptyStateViewOnError(err error, emptyStateView common.View, currentView common.View, context common.ViewContext) (map[string]interface{}, error) {
	e, ok := err.(*common.ViewRenderingError)
	if ok && e.IsContentMissing {
		// render the empty state view only if the content is indicated to be missing from the context
		emptyStateRenderedView, emptyStateRenderingError := emptyStateView.Render(context)
		// if rendering of the empty state view also fails, then capture errors from both the rendering
		// of the current view and the empty state view
		if emptyStateRenderingError != nil {
			return nil, common.NewViewRenderingError(fmt.Sprintf("Unable to render the view type %s (Reason: %s) or its empty state view %s (Reason: %s)",
				currentView.TypeName(), err, emptyStateView.TypeName(), emptyStateRenderingError))
		}

		return emptyStateRenderedView, nil
	}
	return nil, err
}
