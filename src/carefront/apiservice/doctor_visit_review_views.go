package apiservice

import (
	"carefront/common"
	"fmt"
	"reflect"
)

var dVisitReviewViewTypeRegistry = new(common.TypeRegistry)

func init() {
	dVisitReviewViewTypeRegistry.
		RegisterType(DVisitReviewSectionListView{}).
		RegisterType(DVisitReviewStandardPhotosSectionView{}).
		RegisterType(DVisitReviewStandardPhotosSubsectionView{}).
		RegisterType(DVisitReviewStandardPhotosListView{}).
		RegisterType(DVisitReviewStandardSectionView{}).
		RegisterType(DVisitReviewStandardSubsectionView{}).
		RegisterType(DVisitReviewStandardSubsectionView{}).
		RegisterType(DVisitReviewStandardOneColumnRowView{}).
		RegisterType(DVisitReviewStandardTwoColumnRowView{}).
		RegisterType(DVisitReviewDividedViewsList{}).
		RegisterType(DVisitReviewAlertLabelsList{}).
		RegisterType(DVisitReviewTitleLabelsList{}).
		RegisterType(DVisitReviewContentLabelsList{}).
		RegisterType(DVisitReviewCheckXItemsList{}).
		RegisterType(DVisitReviewTitleSubtitleSubItemsDividedItemsList{}).
		RegisterType(DVisitReviewTitleSubtitleLabels{})
}

// View definitions

type DVisitReviewSectionListView struct {
	Sections []common.View `json:"sections"`
}

func (d DVisitReviewSectionListView) TypeName() string {
	return wrapNamespace("sections_list")
}

func (d *DVisitReviewSectionListView) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	renderedView["type"] = d.TypeName()
	renderedSections := make([]interface{}, len(d.Sections))
	for i, section := range d.Sections {
		var err error
		renderedSections[i], err = section.Render(context)
		if err != nil {
			return nil, err
		}
	}
	renderedView["sections"] = renderedSections
	return renderedView, nil
}

type DVisitReviewStandardPhotosSectionView struct {
	Title       string        `json:"title"`
	Subsections []common.View `json:"subsections"`
}

func (d DVisitReviewStandardPhotosSectionView) TypeName() string {
	return wrapNamespace("standard_photo_section")
}

func (d *DVisitReviewStandardPhotosSectionView) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	renderedView["title"] = d.Title
	renderedView["type"] = d.TypeName()
	renderedSubSections := make([]interface{}, len(d.Subsections))

	for i, subSection := range d.Subsections {
		var err error
		renderedSubSections[i], err = subSection.Render(context)
		if err != nil {
			return nil, err
		}
	}
	renderedView["subsections"] = renderedSubSections

	return renderedView, nil
}

type DVisitReviewStandardPhotosSubsectionView struct {
	SubsectionView common.View `json:"view"`
}

func (d DVisitReviewStandardPhotosSubsectionView) TypeName() string {
	return wrapNamespace("standard_photo_subsection")
}

func (d *DVisitReviewStandardPhotosSubsectionView) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	renderedView["type"] = d.TypeName()

	if d.SubsectionView != nil {
		var err error
		renderedView["view"], err = d.SubsectionView.Render(context)
		if err != nil {
			return nil, err
		}
	}
	return renderedView, nil
}

type DVisitReviewStandardPhotosListView struct {
	Photos        []PhotoData `json:"photos"`
	ContentConfig struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (d DVisitReviewStandardPhotosListView) TypeName() string {
	return wrapNamespace("standard_photos_list")
}

func (d *DVisitReviewStandardPhotosListView) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	renderedView["type"] = d.TypeName()

	content, err := getContentFromContextForView(d, d.ContentConfig.Key, context)
	if err != nil {
		return nil, err
	}

	var ok bool
	d.Photos, ok = content.([]PhotoData)
	if !ok {
		return nil, common.NewViewRenderingError(fmt.Sprintf("Expected content in view context to be of type []PhotoData instead it was type %s", reflect.TypeOf(content)))
	}

	renderedView["photos"] = d.Photos

	return renderedView, nil
}

type DVisitReviewStandardSectionView struct {
	Title       string        `json:"title"`
	Subsections []common.View `json:"subsections"`
}

func (d DVisitReviewStandardSectionView) TypeName() string {
	return wrapNamespace("standard_section")
}

func (d *DVisitReviewStandardSectionView) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	renderedView["type"] = d.TypeName()
	renderedView["title"] = d.Title
	renderedSubsections := make([]interface{}, len(d.Subsections))

	for i, subsection := range d.Subsections {
		var err error
		renderedSubsections[i], err = subsection.Render(context)
		if err != nil {
			return nil, err
		}
	}
	renderedView["subsections"] = renderedSubsections
	return renderedView, nil
}

type DVisitReviewStandardSubsectionView struct {
	Title string        `json:"title"`
	Rows  []common.View `json:"rows"`
}

func (d DVisitReviewStandardSubsectionView) TypeName() string {
	return wrapNamespace("standard_subsection")
}

func (d *DVisitReviewStandardSubsectionView) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	renderedView["type"] = d.TypeName()
	renderedView["title"] = d.Title
	renderedRows := make([]interface{}, len(d.Rows))

	for i, row := range d.Rows {
		var err error
		renderedRows[i], err = row.Render(context)
		if err != nil {
			return nil, err
		}
	}
	renderedView["rows"] = renderedRows

	return renderedView, nil
}

type DVisitReviewStandardOneColumnRowView struct {
	SingleView common.View `json:"view"`
}

func (d DVisitReviewStandardOneColumnRowView) TypeName() string {
	return wrapNamespace("standard_one_column_row")
}

func (d *DVisitReviewStandardOneColumnRowView) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	renderedView["type"] = d.TypeName()
	if d.SingleView != nil {
		var err error
		renderedView["view"], err = d.SingleView.Render(context)
		if err != nil {
			return nil, err
		}
	}
	return renderedView, nil
}

type DVisitReviewStandardTwoColumnRowView struct {
	LeftView      common.View `json:"left_view"`
	RightView     common.View `json:"right_view"`
	ContentConfig struct {
		common.ViewCondition `json:"condition"`
	} `json:"content_config"`
}

func (d DVisitReviewStandardTwoColumnRowView) TypeName() string {
	return wrapNamespace("standard_two_column_row")
}

func (d *DVisitReviewStandardTwoColumnRowView) Render(context common.ViewContext) (map[string]interface{}, error) {
	if d.ContentConfig.ViewCondition.Op != "" {
		conditionEvaluator, ok := common.ConditionEvaluators[d.ContentConfig.ViewCondition.Op]
		if !ok {
			return nil, common.NewViewRenderingError(fmt.Sprintf("Condition with op %s not found for view type %s", d.ContentConfig.ViewCondition.Op, d.TypeName()))
		}

		if result, err := conditionEvaluator.EvaluateCondition(d.ContentConfig.ViewCondition, context); err != nil || !result {
			return nil, err
		}
	}
	renderedView := make(map[string]interface{})
	renderedView["type"] = d.TypeName()
	if d.LeftView != nil {
		var err error
		renderedView["left_view"], err = d.LeftView.Render(context)
		if err != nil {
			return nil, err
		}
	}

	if d.RightView != nil {
		var err error
		renderedView["right_view"], err = d.RightView.Render(context)
		if err != nil {
			return nil, err
		}
	}
	return renderedView, nil
}

type DVisitReviewDividedViewsList struct {
	DividedViews []common.View `json:"views"`
}

func (d DVisitReviewDividedViewsList) TypeName() string {
	return wrapNamespace("divided_views_list")
}

func (d *DVisitReviewDividedViewsList) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	renderedView["type"] = d.TypeName()
	renderedSubviews := make([]interface{}, len(d.DividedViews))
	for i, dividedView := range d.DividedViews {
		var err error
		renderedSubviews[i], err = dividedView.Render(context)
		if err != nil {
			return nil, err
		}
	}
	renderedView["views"] = renderedSubviews
	return renderedView, nil
}

type DVisitReviewAlertLabelsList struct {
	Values        []string `json:"values"`
	ContentConfig struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (d DVisitReviewAlertLabelsList) TypeName() string {
	return wrapNamespace("alert_labels_list")
}

func (d *DVisitReviewAlertLabelsList) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	renderedView["type"] = d.TypeName()
	var err error
	d.Values, err = getStringArrayFromContext(d, d.ContentConfig.Key, context)
	if err != nil {
		return nil, err
	}
	renderedView["values"] = d.Values

	return renderedView, nil
}

type DVisitReviewTitleLabelsList struct {
	Values        []string `json:"values"`
	ContentConfig struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (d DVisitReviewTitleLabelsList) TypeName() string {
	return wrapNamespace("title_labels_list")
}

func (d *DVisitReviewTitleLabelsList) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	renderedView["type"] = d.TypeName()
	var err error
	d.Values, err = getStringArrayFromContext(d, d.ContentConfig.Key, context)
	if err != nil {
		return nil, err
	}
	renderedView["values"] = d.Values
	return renderedView, nil
}

type DVisitReviewContentLabelsList struct {
	Values        []string `json:"values"`
	ContentConfig struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (d DVisitReviewContentLabelsList) TypeName() string {
	return wrapNamespace("content_labels_list")
}

func (d *DVisitReviewContentLabelsList) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	renderedView["type"] = d.TypeName()
	var err error
	d.Values, err = getStringArrayFromContext(d, d.ContentConfig.Key, context)
	if err != nil {
		return nil, err
	}
	renderedView["values"] = d.Values
	return renderedView, nil
}

type DVisitReviewCheckXItemsList struct {
	Items         []CheckedUncheckedData `json:"items"`
	ContentConfig struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (d DVisitReviewCheckXItemsList) TypeName() string {
	return wrapNamespace("check_x_items_list")
}

func (d *DVisitReviewCheckXItemsList) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	renderedView["type"] = d.TypeName()
	content, err := getContentFromContextForView(d, d.ContentConfig.Key, context)
	if err != nil {
		return nil, err
	}

	checkedUncheckedItems, ok := content.([]CheckedUncheckedData)
	if !ok {
		return nil, common.NewViewRenderingError(fmt.Sprintf("Expected content of type []CheckedUncheckedData for view type %s", d.TypeName()))
	}
	d.Items = checkedUncheckedItems
	renderedView["items"] = checkedUncheckedItems

	return renderedView, nil
}

type DVisitReviewTitleSubtitleSubItemsDividedItemsList struct {
	Items         []TitleSubtitleSubItemsData `json:"items"`
	ContentConfig struct {
		Key string `json:"key"`
	} `json:"content_config"`
}

func (d DVisitReviewTitleSubtitleSubItemsDividedItemsList) TypeName() string {
	return wrapNamespace("title_subtitle_subitems_divided_items_list")
}

func (d *DVisitReviewTitleSubtitleSubItemsDividedItemsList) Render(context common.ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	renderedView["type"] = d.TypeName()
	content, err := getContentFromContextForView(d, d.ContentConfig.Key, context)
	if err != nil {
		return nil, err
	}

	items, ok := content.([]TitleSubtitleSubItemsData)
	if !ok {
		return nil, common.NewViewRenderingError(fmt.Sprintf("Expected content of type []TitleSubtitleSubItemsData for view type %s", d.TypeName()))
	}
	d.Items = items
	renderedView["items"] = items

	return renderedView, nil
}

type DVisitReviewTitleSubtitleLabels struct {
	Title         string `json:"title"`
	Subtitle      string `json:"subtitle"`
	ContentConfig struct {
		TitleKey             string `json:"title_key"`
		SubtitleKey          string `json:"subtitle_key"`
		common.ViewCondition `json:"condition"`
	} `json:"content_config"`
}

func (d DVisitReviewTitleSubtitleLabels) TypeName() string {
	return wrapNamespace("title_subtitle_labels")
}

func (d *DVisitReviewTitleSubtitleLabels) Render(context common.ViewContext) (map[string]interface{}, error) {
	if d.ContentConfig.ViewCondition.Op != "" {
		conditionEvaluator, ok := common.ConditionEvaluators[d.ContentConfig.ViewCondition.Op]
		if !ok {
			return nil, common.NewViewRenderingError(fmt.Sprintf("Unable to find condition with op %s for view type %s", d.ContentConfig.ViewCondition.Op, d.TypeName()))
		}

		if result, err := conditionEvaluator.EvaluateCondition(d.ContentConfig.ViewCondition, context); err != nil || !result {
			return nil, err
		}
	}

	renderedView := make(map[string]interface{})
	renderedView["type"] = d.TypeName()
	var err error

	d.Title, err = getStringFromContext(d, d.ContentConfig.TitleKey, context)
	if err != nil {
		return nil, err
	}
	renderedView["title"] = d.Title

	d.Subtitle, err = getStringFromContext(d, d.ContentConfig.SubtitleKey, context)
	if err != nil {
		return nil, err
	}
	renderedView["subtitle"] = d.Subtitle

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
		return "", common.NewViewRenderingError(fmt.Sprintf("Expected string for content of view type %s instead got %s", view.TypeName(), reflect.TypeOf(content)))
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
		return nil, common.NewViewRenderingError(fmt.Sprintf("Expected []string for content of view type %s instead got %s", view.TypeName(), reflect.TypeOf(content)))
	}

	return stringArray, nil
}

func getContentFromContextForView(view common.View, key string, context common.ViewContext) (interface{}, error) {
	if key == "" {
		return nil, common.NewViewRenderingError(fmt.Sprintf("Content config key not specified for view type %s", view.TypeName()))
	}

	content, ok := context.Get(key)
	if !ok {
		return nil, common.NewViewRenderingError(fmt.Sprintf("Content with key %s not found in view context for view type %s", key, view.TypeName()))
	}

	return content, nil
}
