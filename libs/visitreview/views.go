package visitreview

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/mapstructure"
)

// This TypeRegistry contains all views pertaining to the  namespace
var TypeRegistry *mapstructure.TypeRegistry = mapstructure.NewTypeRegistry()

func init() {
	TypeRegistry.MustRegisterType(&SectionListView{})
	TypeRegistry.MustRegisterType(&StandardPhotosSectionView{})
	TypeRegistry.MustRegisterType(&StandardPhotosSubsectionView{})
	TypeRegistry.MustRegisterType(&StandardPhotosListView{})
	TypeRegistry.MustRegisterType(&StandardMediaSectionView{})
	TypeRegistry.MustRegisterType(&StandardMediaSubsectionView{})
	TypeRegistry.MustRegisterType(&StandardMediaListView{})
	TypeRegistry.MustRegisterType(&StandardSectionView{})
	TypeRegistry.MustRegisterType(&StandardSubsectionView{})
	TypeRegistry.MustRegisterType(&StandardOneColumnRowView{})
	TypeRegistry.MustRegisterType(&StandardTwoColumnRowView{})
	TypeRegistry.MustRegisterType(&DividedViewsList{})
	TypeRegistry.MustRegisterType(&AlertLabelsList{})
	TypeRegistry.MustRegisterType(&TitleLabelsList{})
	TypeRegistry.MustRegisterType(&ContentLabelsList{})
	TypeRegistry.MustRegisterType(&CheckXItemsList{})
	TypeRegistry.MustRegisterType(&TitleSubtitleLabels{})
	TypeRegistry.MustRegisterType(&EmptyLabelView{})
	TypeRegistry.MustRegisterType(&EmptyTitleSubtitleLabelView{})
	TypeRegistry.MustRegisterType(&TitlePhotosItemsListView{})
	TypeRegistry.MustRegisterType(&TitleMediaItemsListView{})
	TypeRegistry.MustRegisterType(&TitleSubItemsLabelContentItemsList{})
}

// View definitions

type SectionListView struct {
	Sections []View `json:"sections"`
	Type     string `json:"type"`
}

func (d *SectionListView) TypeName() string {
	return wrapNamespace("sections_list")
}

func (d *SectionListView) Validate() error {
	d.Type = d.TypeName()
	for _, section := range d.Sections {
		if err := section.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (d *SectionListView) Render(context *ViewContext) (map[string]interface{}, error) {
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

type StandardPhotosSectionView struct {
	Title         string         `json:"title"`
	Subsections   []View         `json:"subsections"`
	ContentConfig *ContentConfig `json:"content_config,omitempty"`
	Type          string         `json:"type"`
}

func (d *StandardPhotosSectionView) TypeName() string {
	return wrapNamespace("standard_photo_section")
}

func (d *StandardPhotosSectionView) Validate() error {
	d.Type = d.TypeName()
	for _, subsection := range d.Subsections {
		if err := subsection.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (d *StandardPhotosSectionView) Render(context *ViewContext) (map[string]interface{}, error) {
	if d.ContentConfig != nil && d.ContentConfig.ViewCondition.Op != "" {
		if result, err := EvaluateConditionForView(d, d.ContentConfig.ViewCondition, context); err != nil || !result {
			return nil, err
		}
	}

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

type StandardPhotosSubsectionView struct {
	SubsectionView View           `json:"view"`
	ContentConfig  *ContentConfig `json:"content_config"`
	Type           string         `json:"type"`
}

func (d *StandardPhotosSubsectionView) TypeName() string {
	return wrapNamespace("standard_photo_subsection")
}

func (d *StandardPhotosSubsectionView) Validate() error {
	d.Type = d.TypeName()
	if d.SubsectionView != nil {
		if err := d.SubsectionView.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (d *StandardPhotosSubsectionView) Render(context *ViewContext) (map[string]interface{}, error) {
	if d.ContentConfig != nil && d.ContentConfig.ViewCondition.Op != "" {
		if result, err := EvaluateConditionForView(d, d.ContentConfig.ViewCondition, context); err != nil || !result {
			return nil, err
		}
	}

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

type StandardPhotosListView struct {
	Photos        []PhotoData    `json:"photos"`
	ContentConfig *ContentConfig `json:"content_config,omitempty"`
	Type          string         `json:"type"`
}

func (d *StandardPhotosListView) TypeName() string {
	return wrapNamespace("standard_photos_list")
}

func (d *StandardPhotosListView) Validate() error {
	d.Type = d.TypeName()
	return nil
}

func (d *StandardPhotosListView) Render(context *ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})

	content, err := getContentFromContextForView(d, d.ContentConfig.Key, context)
	if err != nil {
		return nil, err
	} else if content == nil {
		return nil, nil
	}

	var ok bool
	d.Photos, ok = content.([]PhotoData)
	if !ok {
		return nil, NewViewRenderingError(fmt.Sprintf("Expected content in view context to be of type []PhotoData instead it was type %T", content))
	}

	renderedView["photos"] = d.Photos
	renderedView["type"] = d.TypeName()

	return renderedView, nil
}

type TitlePhotosItemsListView struct {
	Items         []TitlePhotoListData `json:"items"`
	ContentConfig *ContentConfig       `json:"content_config"`
	Type          string               `json:"type"`
}

func (d *TitlePhotosItemsListView) TypeName() string {
	return wrapNamespace("title_photos_items_list")
}

func (d *TitlePhotosItemsListView) Validate() error {
	d.Type = d.TypeName()
	return nil
}

func (d *TitlePhotosItemsListView) Render(context *ViewContext) (map[string]interface{}, error) {
	if d.ContentConfig != nil && d.ContentConfig.ViewCondition.Op != "" {
		if result, err := EvaluateConditionForView(d, d.ContentConfig.ViewCondition, context); err != nil || !result {
			return nil, err
		}
	}

	renderedView := make(map[string]interface{})
	content, err := getContentFromContextForView(d, d.ContentConfig.Key, context)
	if err != nil {
		return nil, err
	} else if content == nil {
		return nil, nil
	}

	var ok bool
	d.Items, ok = content.([]TitlePhotoListData)
	if !ok {
		return nil, NewViewRenderingError(fmt.Sprintf("Expected content in view context to be of type []TitlePhotoListData instead it was type %T", content))
	}

	renderedView["items"] = d.Items
	renderedView["type"] = d.TypeName()

	return renderedView, nil
}

type StandardMediaSectionView struct {
	Title         string         `json:"title"`
	Subsections   []View         `json:"subsections"`
	ContentConfig *ContentConfig `json:"content_config,omitempty"`
	Type          string         `json:"type"`
}

func (d *StandardMediaSectionView) TypeName() string {
	return wrapNamespace("standard_media_section")
}

func (d *StandardMediaSectionView) Validate() error {
	d.Type = d.TypeName()
	for _, subsection := range d.Subsections {
		if err := subsection.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (d *StandardMediaSectionView) Render(context *ViewContext) (map[string]interface{}, error) {
	if d.ContentConfig != nil && d.ContentConfig.ViewCondition.Op != "" {
		if result, err := EvaluateConditionForView(d, d.ContentConfig.ViewCondition, context); err != nil || !result {
			return nil, err
		}
	}

	renderedView := make(map[string]interface{})
	var renderedSubSections []interface{}
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

type StandardMediaSubsectionView struct {
	SubsectionView View           `json:"view"`
	ContentConfig  *ContentConfig `json:"content_config"`
	Type           string         `json:"type"`
}

func (d *StandardMediaSubsectionView) TypeName() string {
	return wrapNamespace("standard_media_subsection")
}

func (d *StandardMediaSubsectionView) Validate() error {
	d.Type = d.TypeName()
	if d.SubsectionView != nil {
		if err := d.SubsectionView.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (d *StandardMediaSubsectionView) Render(context *ViewContext) (map[string]interface{}, error) {
	if d.ContentConfig != nil && d.ContentConfig.ViewCondition.Op != "" {
		if result, err := EvaluateConditionForView(d, d.ContentConfig.ViewCondition, context); err != nil || !result {
			return nil, err
		}
	}

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

type StandardMediaListView struct {
	Media         []MediaData    `json:"media"`
	ContentConfig *ContentConfig `json:"content_config,omitempty"`
	Type          string         `json:"type"`
}

func (d *StandardMediaListView) TypeName() string {
	return wrapNamespace("standard_media_list")
}

func (d *StandardMediaListView) Validate() error {
	d.Type = d.TypeName()
	return nil
}

func (d *StandardMediaListView) Render(context *ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})

	content, err := getContentFromContextForView(d, d.ContentConfig.Key, context)
	if err != nil {
		return nil, err
	} else if content == nil {
		return nil, nil
	}

	var ok bool
	d.Media, ok = content.([]MediaData)
	if !ok {
		return nil, NewViewRenderingError(fmt.Sprintf("Expected content in view context to be of type []MediaData instead it was type %T", content))
	}

	renderedView["media"] = d.Media
	renderedView["type"] = d.TypeName()

	return renderedView, nil
}

type TitleMediaItemsListView struct {
	Items         []TitleMediaListData `json:"items"`
	ContentConfig *ContentConfig       `json:"content_config"`
	Type          string               `json:"type"`
}

func (d *TitleMediaItemsListView) TypeName() string {
	return wrapNamespace("title_media_items_list")
}

func (d *TitleMediaItemsListView) Validate() error {
	d.Type = d.TypeName()
	return nil
}

func (d *TitleMediaItemsListView) Render(context *ViewContext) (map[string]interface{}, error) {
	if d.ContentConfig != nil && d.ContentConfig.ViewCondition.Op != "" {
		if result, err := EvaluateConditionForView(d, d.ContentConfig.ViewCondition, context); err != nil || !result {
			return nil, err
		}
	}

	renderedView := make(map[string]interface{})
	content, err := getContentFromContextForView(d, d.ContentConfig.Key, context)
	if err != nil {
		return nil, err
	} else if content == nil {
		return nil, nil
	}

	var ok bool
	d.Items, ok = content.([]TitleMediaListData)
	if !ok {
		return nil, NewViewRenderingError(fmt.Sprintf("Expected content in view context to be of type []TitleMediaListData instead it was type %T", content))
	}

	renderedView["items"] = d.Items
	renderedView["type"] = d.TypeName()

	return renderedView, nil
}

type StandardSectionView struct {
	Title         string         `json:"title"`
	Subsections   []View         `json:"subsections"`
	ContentConfig *ContentConfig `json:"content_config,omitempty"`
	Type          string         `json:"type"`
}

func (d *StandardSectionView) Validate() error {
	d.Type = d.TypeName()
	for _, subsection := range d.Subsections {
		if err := subsection.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (d *StandardSectionView) TypeName() string {
	return wrapNamespace("standard_section")
}

func (d *StandardSectionView) Render(context *ViewContext) (map[string]interface{}, error) {
	if d.ContentConfig != nil && d.ContentConfig.ViewCondition.Op != "" {
		if result, err := EvaluateConditionForView(d, d.ContentConfig.ViewCondition, context); err != nil || !result {
			return nil, errors.Trace(err)
		}
	}
	renderedView := make(map[string]interface{})
	renderedSubsections := make([]interface{}, 0)

	for _, subsection := range d.Subsections {
		renderedSubsection, err := subsection.Render(context)
		if err != nil {
			return nil, errors.Trace(err)
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

type StandardSubsectionView struct {
	Title         string         `json:"title"`
	Rows          []View         `json:"rows"`
	ContentConfig *ContentConfig `json:"content_config,omitempty"`
	Type          string         `json:"type"`
}

func (d *StandardSubsectionView) TypeName() string {
	return wrapNamespace("standard_subsection")
}

func (d *StandardSubsectionView) Validate() error {
	d.Type = d.TypeName()
	for _, row := range d.Rows {
		if err := row.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (d *StandardSubsectionView) Render(context *ViewContext) (map[string]interface{}, error) {
	if d.ContentConfig != nil && d.ContentConfig.ViewCondition.Op != "" {
		if result, err := EvaluateConditionForView(d, d.ContentConfig.ViewCondition, context); err != nil || !result {
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

type StandardOneColumnRowView struct {
	SingleView    View           `json:"view"`
	ContentConfig *ContentConfig `json:"content_config,omitempty"`
	Type          string         `json:"type"`
}

func (d *StandardOneColumnRowView) TypeName() string {
	return wrapNamespace("standard_one_column_row")
}

func (d *StandardOneColumnRowView) Validate() error {
	d.Type = d.TypeName()
	if d.SingleView != nil {
		if err := d.SingleView.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (d *StandardOneColumnRowView) Render(context *ViewContext) (map[string]interface{}, error) {
	if d.ContentConfig != nil && d.ContentConfig.ViewCondition.Op != "" {
		if result, err := EvaluateConditionForView(d, d.ContentConfig.ViewCondition, context); err != nil || !result {
			return nil, err
		}
	}
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

type StandardTwoColumnRowView struct {
	LeftView      View           `json:"left_view"`
	RightView     View           `json:"right_view"`
	ContentConfig *ContentConfig `json:"content_config,omitempty"`
	Type          string         `json:"type"`
}

func (d *StandardTwoColumnRowView) TypeName() string {
	return wrapNamespace("standard_two_column_row")
}

func (d *StandardTwoColumnRowView) Validate() error {
	d.Type = d.TypeName()
	if d.LeftView != nil {
		if err := d.LeftView.Validate(); err != nil {
			return err
		}
	}

	if d.RightView != nil {
		if err := d.RightView.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (d *StandardTwoColumnRowView) Render(context *ViewContext) (map[string]interface{}, error) {
	if d.ContentConfig != nil && d.ContentConfig.ViewCondition.Op != "" {
		if result, err := EvaluateConditionForView(d, d.ContentConfig.ViewCondition, context); err != nil || !result {
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

type DividedViewsList struct {
	DividedViews []View `json:"views"`
	Type         string `json:"type"`
}

func (d *DividedViewsList) TypeName() string {
	return wrapNamespace("divided_views_list")
}

func (d *DividedViewsList) Validate() error {
	d.Type = d.TypeName()
	for _, view := range d.DividedViews {
		if err := view.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (d *DividedViewsList) Render(context *ViewContext) (map[string]interface{}, error) {
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

type AlertLabelsList struct {
	Values         []string       `json:"values"`
	EmptyStateView View           `json:"empty_state_view"`
	ContentConfig  *ContentConfig `json:"content_config"`
	Type           string         `json:"type"`
}

func (d *AlertLabelsList) TypeName() string {
	return wrapNamespace("alert_labels_list")
}

func (d *AlertLabelsList) Validate() error {
	d.Type = d.TypeName()
	if d.EmptyStateView != nil {
		if err := d.EmptyStateView.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (d *AlertLabelsList) Render(context *ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	var err error
	d.Values, err = getStringArrayFromContext(d, d.ContentConfig.Key, context)

	if err != nil {
		if d.EmptyStateView != nil {
			return handleRenderingOfEmptyStateViewOnError(err, d.EmptyStateView, d, context)
		}
		return nil, err
	} else if d.Values == nil {
		return nil, nil
	}

	renderedView["type"] = d.TypeName()
	renderedView["values"] = d.Values

	return renderedView, nil
}

type TitleLabelsList struct {
	Values        []string       `json:"values"`
	ContentConfig *ContentConfig `json:"content_config,omitempty"`
	Type          string         `json:"type"`
}

func (d *TitleLabelsList) TypeName() string {
	return wrapNamespace("title_labels_list")
}

func (d *TitleLabelsList) Validate() error {
	d.Type = d.TypeName()
	return nil
}

func (d *TitleLabelsList) Render(context *ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	var err error

	content, err := getContentFromContextForView(d, d.ContentConfig.Key, context)
	if err != nil {
		return nil, err
	} else if content == nil {
		return nil, nil
	}

	switch content.(type) {
	case string:
		d.Values = []string{content.(string)}
	case []string:
		d.Values = content.([]string)
	default:
		return nil, NewViewRenderingError(fmt.Sprintf("Expected content to be either string or []string for view type %s", d.TypeName()))
	}

	renderedView["type"] = d.TypeName()
	renderedView["values"] = d.Values
	return renderedView, nil
}

type ContentLabelsList struct {
	Values         []string       `json:"values"`
	EmptyStateView View           `json:"empty_state_view"`
	ContentConfig  *ContentConfig `json:"content_config"`
	Type           string         `json:"type"`
}

func (d *ContentLabelsList) TypeName() string {
	return wrapNamespace("content_labels_list")
}

func (d *ContentLabelsList) Validate() error {
	d.Type = d.TypeName()
	if d.EmptyStateView != nil {
		if err := d.EmptyStateView.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (d *ContentLabelsList) Render(context *ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})

	content, err := getContentFromContextForView(d, d.ContentConfig.Key, context)
	if err != nil {
		if d.EmptyStateView != nil {
			return handleRenderingOfEmptyStateViewOnError(err, d.EmptyStateView, d, context)
		}
		return nil, err
	} else if content == nil {
		return nil, nil
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
		return nil, NewViewRenderingError(fmt.Sprintf("Expected content to be either string, []string, []CheckedUnCheckedData or []TitleSubtitleSubitemsData for view type %s and key %s but was %T", d.TypeName(), d.ContentConfig.Key, content))
	}

	renderedView["type"] = d.TypeName()
	renderedView["values"] = d.Values
	return renderedView, nil
}

type CheckXItemsList struct {
	Items         []CheckedUncheckedData `json:"items"`
	ContentConfig *ContentConfig         `json:"content_config"`
	Type          string                 `json:"type"`
}

func (d *CheckXItemsList) TypeName() string {
	return wrapNamespace("check_x_items_list")
}

func (d *CheckXItemsList) Validate() error {
	d.Type = d.TypeName()
	return nil
}

func (d *CheckXItemsList) Render(context *ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	content, err := getContentFromContextForView(d, d.ContentConfig.Key, context)
	if err != nil {
		return nil, err
	} else if content == nil {
		return nil, nil
	}

	checkedUncheckedItems, ok := content.([]CheckedUncheckedData)
	if !ok {
		return nil, NewViewRenderingError(fmt.Sprintf("Expected content of type []CheckedUncheckedData for view type %s and key %s", d.TypeName(), d.ContentConfig.Key))
	}
	d.Items = checkedUncheckedItems
	renderedView["items"] = checkedUncheckedItems
	renderedView["type"] = d.TypeName()

	return renderedView, nil
}

type TitleSubItemsLabelContentItemsList struct {
	Items          []TitleSubItemsDescriptionContentData `json:"items"`
	EmptyStateView View                                  `json:"empty_state_view"`
	ContentConfig  *ContentConfig                        `json:"content_config"`
	Type           string                                `json:"type"`
}

func (d *TitleSubItemsLabelContentItemsList) TypeName() string {
	return wrapNamespace("title_subitems_description_content_labels_divided_items_list")
}

func (d *TitleSubItemsLabelContentItemsList) Validate() error {
	d.Type = d.TypeName()

	if d.EmptyStateView != nil {
		if err := d.EmptyStateView.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (d *TitleSubItemsLabelContentItemsList) Render(context *ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	content, err := getContentFromContextForView(d, d.ContentConfig.Key, context)
	if err != nil {
		if d.EmptyStateView != nil {
			return handleRenderingOfEmptyStateViewOnError(err, d.EmptyStateView, d, context)
		}

		return nil, err
	} else if content == nil {
		return nil, nil
	}

	items, ok := content.([]TitleSubItemsDescriptionContentData)
	if !ok {
		return nil, NewViewRenderingError(fmt.Sprintf("Expected content of type []TitleSubItemsDescriptionContentData for view type %s for key %s", d.TypeName(), d.ContentConfig.Key))
	}
	d.Items = items
	renderedView["items"] = items
	renderedView["type"] = d.TypeName()

	return renderedView, nil
}

type TitleSubtitleLabels struct {
	Title          string         `json:"title"`
	Subtitle       string         `json:"subtitle"`
	EmptyStateView View           `json:"empty_state_view"`
	ContentConfig  *ContentConfig `json:"content_config"`
	Type           string         `json:"type"`
}

func (d *TitleSubtitleLabels) TypeName() string {
	return wrapNamespace("title_subtitle_labels")
}

func (d *TitleSubtitleLabels) Validate() error {
	d.Type = d.TypeName()
	if d.EmptyStateView != nil {
		if err := d.EmptyStateView.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (d *TitleSubtitleLabels) Render(context *ViewContext) (map[string]interface{}, error) {
	if d.ContentConfig != nil && d.ContentConfig.ViewCondition.Op != "" {
		if result, err := EvaluateConditionForView(d, d.ContentConfig.ViewCondition, context); err != nil || !result {
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

	content, err := getContentFromContextForView(d, d.ContentConfig.SubtitleKey, context)
	if err != nil {
		if d.EmptyStateView != nil {
			return handleRenderingOfEmptyStateViewOnError(err, d.EmptyStateView, d, context)
		}
		return nil, err
	} else if content == nil {
		return nil, nil
	}

	switch c := content.(type) {
	case string:
		d.Subtitle = c
	case []CheckedUncheckedData:
		strArray := make([]string, 0, len(c))
		for _, item := range c {
			if item.IsChecked {
				strArray = append(strArray, item.Value)
			}
		}
		d.Subtitle = strings.Join(strArray, "\n")
	}

	renderedView["title"] = d.Title
	renderedView["subtitle"] = d.Subtitle
	renderedView["type"] = d.TypeName()
	return renderedView, nil
}

type ContentConfig struct {
	Key           string `json:"key,omitempty"`
	ViewCondition `json:"condition,omitempty"`
	TitleKey      string `json:"title_key,omitempty"`
	SubtitleKey   string `json:"subtitle_key,omitempty"`
}

type EmptyLabelView struct {
	Text          string
	ContentConfig *ContentConfig `json:"content_config,omitempty"`
	Type          string         `json:"type"`
}

func (d *EmptyLabelView) TypeName() string {
	return wrapNamespace("empty_label")
}

func (d *EmptyLabelView) Validate() error {
	d.Type = d.TypeName()
	return nil
}

func (d *EmptyLabelView) Render(context *ViewContext) (map[string]interface{}, error) {
	renderedView := make(map[string]interface{})
	text, err := getStringFromContext(d, d.ContentConfig.Key, context)
	if err != nil {
		return nil, err
	}

	renderedView["text"] = text
	renderedView["type"] = d.TypeName()
	return renderedView, nil
}

type EmptyTitleSubtitleLabelView struct {
	Title         string
	Subtitle      string
	ContentConfig *ContentConfig `json:"content_config,omitempty"`
	Type          string         `json:"type"`
}

func (d *EmptyTitleSubtitleLabelView) TypeName() string {
	return wrapNamespace("empty_title_subtitle_labels")
}

func (d *EmptyTitleSubtitleLabelView) Validate() error {
	d.Type = d.TypeName()
	return nil
}

func (d *EmptyTitleSubtitleLabelView) Render(context *ViewContext) (map[string]interface{}, error) {
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

func getStringFromContext(view View, key string, context *ViewContext) (string, error) {
	content, err := getContentFromContextForView(view, key, context)
	if err != nil {
		return "", err
	} else if content == nil {
		return "", nil
	}

	str, ok := content.(string)
	if !ok {
		return "", NewViewRenderingError(fmt.Sprintf("Expected string for content of view type %s instead got %T for key %s", view.TypeName(), content, key))
	}

	return str, nil
}

func getStringArrayFromContext(view View, key string, context *ViewContext) ([]string, error) {

	content, err := getContentFromContextForView(view, key, context)
	if err != nil {
		return nil, err
	} else if content == nil {
		return nil, nil
	}

	stringArray, ok := content.([]string)
	if !ok {
		return nil, NewViewRenderingError(fmt.Sprintf("Expected []string for content of view type %s instead got %T", view.TypeName(), content))
	}

	return stringArray, nil
}

func getContentFromContextForView(view View, key string, context *ViewContext) (interface{}, error) {
	if key == "" {
		return nil, NewViewRenderingError(fmt.Sprintf("Content config key not specified for view type %s", view.TypeName()))
	}

	content, ok := context.Get(key)
	if !ok && !context.IgnoreMissingKeys {
		return nil, &ViewRenderingError{
			Message:          fmt.Sprintf("Content with key %s not found in view context for view type %s", key, view.TypeName()),
			IsContentMissing: true,
		}
	}
	return content, nil
}

func handleRenderingOfEmptyStateViewOnError(err error, emptyStateView View, currentView View, context *ViewContext) (map[string]interface{}, error) {
	e, ok := err.(*ViewRenderingError)
	if ok && e.IsContentMissing {
		// render the empty state view only if the content is indicated to be missing from the context
		emptyStateRenderedView, emptyStateRenderingError := emptyStateView.Render(context)
		// if rendering of the empty state view also fails, then capture errors from both the rendering
		// of the current view and the empty state view
		if emptyStateRenderingError != nil {
			return nil, NewViewRenderingError(fmt.Sprintf("Unable to render the view type %s (Reason: %s) or its empty state view %s (Reason: %s)",
				currentView.TypeName(), err, emptyStateView.TypeName(), emptyStateRenderingError))
		}

		return emptyStateRenderedView, nil
	}
	return nil, err
}
