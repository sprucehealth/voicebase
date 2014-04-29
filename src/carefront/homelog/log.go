/*
Package homelog provides the implementation of the home feed notifications and log.
*/
package homelog

import (
	"carefront/api"
	"carefront/common"

	"reflect"
)

const (
	logItemNamespace = "log_item"
)

type logItem interface {
	TypeName() string
	makeView(dataAPI api.DataAPI, patientId int64, item *common.HealthLogItem) (view, error)
}

type textLogItem struct {
	Text    string
	IconURL string
	TapURL  string
}

type titledLogItem struct {
	Title    string
	Subtitle string
	IconURL  string
	TapURL   string
}

func (*textLogItem) TypeName() string {
	return "text"
}

func (*titledLogItem) TypeName() string {
	return "title_subtitle"
}

func (n *textLogItem) makeView(dataAPI api.DataAPI, patientId int64, item *common.HealthLogItem) (view, error) {
	return &textView{
		Type:     logItemNamespace + ":text",
		DateTime: item.Timestamp,
		Text:     n.Text,
		IconURL:  n.IconURL,
		TapURL:   n.TapURL,
	}, nil
}

func (n *titledLogItem) makeView(dataAPI api.DataAPI, patientId int64, item *common.HealthLogItem) (view, error) {
	return &titleSubtitleView{
		Type:     logItemNamespace + ":title_subtitle",
		DateTime: item.Timestamp,
		Title:    n.Title,
		Subtitle: n.Subtitle,
		IconURL:  n.IconURL,
		TapURL:   n.TapURL,
	}, nil
}

var logItemTypes = map[string]reflect.Type{}

func init() {
	registerLogItemType(&textLogItem{})
	registerLogItemType(&titledLogItem{})
}

func registerLogItemType(n logItem) {
	logItemTypes[n.TypeName()] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(n)).Interface())
}
