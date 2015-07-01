package geckoboard

import (
	"encoding"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Widget interface {
	AppendData(cols []string, row []interface{}) error
}

type PieChart struct {
	Items []*PieChartItem `json:"item"`
	// Fields not part of the API
	Colors []string `json:"colors,omitempty"`
}

type PieChartItem struct {
	Value float64 `json:"value"`
	Label string  `json:"label"`
	Color string  `json:"color,omitempty"`
}

type BarChart struct {
	XAxis struct {
		Labels []string `json:"labels,omitempty"`
		Type   string   `json:"type,omitempty"` // standard, datetime
	} `json:"x_axis"`
	YAxis struct {
		Format string `json:"format,omitempty"` // decimal, percent, currency
		Unit   string `json:"unit,omitempty"`   // if format is currency then this must be ISO-4217
	} `json:"y_axis"`
	Series []*Series `json:"series"`
}

type LineChart struct {
	XAxis struct {
		Labels []string `json:"labels,omitempty"`
		Type   string   `json:"type,omitempty"` // standard, datetime
	} `json:"x_axis"`
	YAxis struct {
		Format string `json:"format,omitempty"` // decimal, percent, currency
		Unit   string `json:"unit,omitempty"`   // if format is currency then this must be ISO-4217
	} `json:"y_axis"`
	Series []*Series `json:"series"`
}

type NumberAndSecondaryStat struct {
	Absolute bool    `json:"absolute"`
	Type     string  `json:"type,omitempty"` // reverse
	Items    []*Item `json:"item"`
}

type Meter struct {
	Item float64 `json:"item"`
	Min  struct {
		Value float64 `json:"value"`
	} `json:"min"`
	Max struct {
		Value float64 `json:"value"`
	} `json:"max"`
}

type Map struct {
	Points struct {
		Points []*Point `json:"point"`
	} `json:"points"`
}

type List []*ListItem

type ListItem struct {
	Title struct {
		Text string `json:"text"`
	} `json:"title"`
	Label       *Label `json:"label,omitempty"`
	Description string `json:"description"`
}

type Label struct {
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

type Point struct {
	City      *City   `json:"city,omitempty"`
	Size      int     `json:"size,omitempty"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	IP        string  `json:"ip,omitempty"`
	Color     string  `json:"color,omitempty"`
}

type City struct {
	Name        string `json:"city_name,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	RegionCode  string `json:"region_code,omitempty"`
}

type Item struct {
	Value  float64 `json:"value"`
	Text   string  `json:"text,omitempty"`
	Prefix string  `json:"prefix,omitempty"`
	Type   string  `json:"type,omitempty"` // reverse, time_duration, text
}

type Series struct {
	Data           []float64 `json:"data"`
	Name           string    `json:"name,omitempty"`
	IncompleteFrom string    `json:"incomplete_from,omitempty"`
	Type           string    `json:"type,omitempty"` // line charts: main, secondary
}

type Funnel struct {
	Items      []*FunnelItem `json:"item"`
	Type       string        `json:"type,omitempty"`       // reverse
	Percentage string        `json:"percentage,omitempty"` // hide
}

type FunnelItem struct {
	Value float64 `json:"value"`
	Label string  `json:"label"`
}

type Leaderboard struct {
	Items  []*LeaderboardItem `json:"items"`
	Format string             `json:"format,omitempty"` // optional: Possible values are "decimal", "percent" and "currency". The default is "decimal".
	Unit   string             `json:"unit,omitempty"`   // optional: When the format is currency this must be an ISO 4217 currency code. E.g. "GBP", "USD", "EUR"
}

type LeaderboardItem struct {
	Label        string      `json:"label"`
	Value        interface{} `json:"value"`
	PreviousRank int         `json:"previous_rank,omitempty"`
}

// Text is a widget that shows arbitrary text. Up to 10 items can be shown.
//
// https://developer.geckoboard.com/#text
type Text struct {
	Items []*TextItem `json:"item"`
}

// TextItem is one item in a text widget
type TextItem struct {
	Text string       `json:"text"`
	Type TextItemType `json:"type"`
}

// TextItemType is the type of item in a text widget. It changes
// the icon shown for an item.
type TextItemType int

const (
	// AlertItem includes exclamation point on yellow background on the text item
	AlertItem TextItemType = 1
	// InfoItem includes 'i' icon on grey background on the text item
	InfoItem TextItemType = 2
)

var textItemTypeMap = map[string]TextItemType{
	"alert": AlertItem,
	"info":  InfoItem,
}

func (w *PieChart) AppendData(cols []string, row []interface{}) error {
	var err error
	it := &PieChartItem{}
	for i, c := range cols {
		switch c {
		case "value":
			it.Value, err = toFloat64(row[i])
		case "label":
			it.Label, err = toString(row[i])
		case "color":
			it.Color, err = toString(row[i])
		default:
			return fmt.Errorf("geckoboard: unknown PieChart column %s", c)
		}
		if err != nil {
			return fmt.Errorf("geckoboard: PieChart %s: %s", c, err)
		}
	}
	if it.Color == "" && len(w.Items) < len(w.Colors) {
		it.Color = w.Colors[len(w.Items)]
	}
	w.Items = append(w.Items, it)
	return nil
}

func (w *BarChart) AppendData(cols []string, row []interface{}) error {
	if len(w.Series) == 0 {
		w.Series = append(w.Series, &Series{})
	}
	for i, c := range cols {
		switch strings.ToLower(c) {
		case "data", "value":
			v, err := toFloat64(row[i])
			if err != nil {
				return fmt.Errorf("geckoboard: BarChart value: %s", err)
			}
			w.Series[0].Data = append(w.Series[0].Data, v)
		case "label":
			label, err := toString(row[i])
			if err != nil {
				return fmt.Errorf("geckoboard: BarChart label: %s", err)
			}
			w.XAxis.Labels = append(w.XAxis.Labels, label)
		default:
			return fmt.Errorf("geckoboard: unknown BarChart column %s", c)
		}
	}
	return nil
}

func (w *LineChart) AppendData(cols []string, row []interface{}) error {
	if len(w.Series) == 0 {
		w.Series = append(w.Series, &Series{})
	}
	for i, c := range cols {
		switch strings.ToLower(c) {
		case "data", "value":
			v, err := toFloat64(row[i])
			if err != nil {
				return fmt.Errorf("geckoboard: LineChart value: %s", err)
			}
			w.Series[0].Data = append(w.Series[0].Data, v)
		case "label":
			label, err := toString(row[i])
			if err != nil {
				return fmt.Errorf("geckoboard: LineChart label: %s", err)
			}
			w.XAxis.Labels = append(w.XAxis.Labels, label)
		default:
			return fmt.Errorf("geckoboard: unknown LineChart column %s", c)
		}
	}
	return nil
}

func (w *NumberAndSecondaryStat) AppendData(cols []string, row []interface{}) error {
	var err error
	item := &Item{}
	for i, c := range cols {
		switch c {
		case "item", "value":
			item.Value, err = toFloat64(row[i])
		case "text":
			item.Text, err = toString(row[i])
		case "prefix":
			item.Prefix, err = toString(row[i])
		case "type":
			item.Type, err = toString(row[i])
		default:
			return fmt.Errorf("geckoboard: unknown NumberAndSecondaryStat column %s", c)
		}
		if err != nil {
			return fmt.Errorf("geckoboard: NumberAndSecondaryStat %s: %s", c, err)
		}
	}
	w.Items = append(w.Items, item)
	return nil
}

func (w *Map) AppendData(cols []string, row []interface{}) error {
	var err error
	p := &Point{}
	for i, c := range cols {
		switch c {
		case "city_name":
			if p.City == nil {
				p.City = &City{}
			}
			p.City.Name, err = toString(row[i])
		case "country_code":
			if p.City == nil {
				p.City = &City{}
			}
			p.City.CountryCode, err = toString(row[i])
		case "region_code":
			if p.City == nil {
				p.City = &City{}
			}
			p.City.RegionCode, err = toString(row[i])
		case "size":
			p.Size, err = toInteger(row[i])
		case "latitude":
			p.Latitude, err = toFloat64(row[i])
		case "longitude":
			p.Longitude, err = toFloat64(row[i])
		case "color":
			p.Color, err = toString(row[i])
		default:
			return fmt.Errorf("geckoboard: unknown Map column %s", c)
		}
		if err != nil {
			return fmt.Errorf("geckoboard: Map %s: %s", c, err)
		}
	}
	w.Points.Points = append(w.Points.Points, p)
	return nil
}

func (w *List) AppendData(cols []string, row []interface{}) error {
	var err error
	it := &ListItem{}
	for i, c := range cols {
		switch c {
		case "title":
			it.Title.Text, err = toString(row[i])
		case "name":
			if it.Label == nil {
				it.Label = &Label{}
			}
			it.Label.Name, err = toString(row[i])
		case "color":
			if it.Label == nil {
				it.Label = &Label{}
			}
			it.Label.Color, err = toString(row[i])
		case "description":
			it.Description, err = toString(row[i])
		default:
			return fmt.Errorf("geckoboard: unknown List column %s", c)
		}
		if err != nil {
			return fmt.Errorf("geckoboard: List %s: %s", c, err)
		}
	}
	*w = append(*w, it)
	return nil
}

func (w *Funnel) AppendData(cols []string, row []interface{}) error {
	var err error
	it := &FunnelItem{}
	for i, c := range cols {
		switch c {
		case "value":
			it.Value, err = toFloat64(row[i])
		case "label":
			it.Label, err = toString(row[i])
		default:
			return fmt.Errorf("geckoboard: unknown Funnel column %s", c)
		}
		if err != nil {
			return fmt.Errorf("geckoboard: Funnel %s: %s", c, err)
		}
	}
	w.Items = append(w.Items, it)
	return nil
}

func (w *Text) AppendData(cols []string, row []interface{}) error {
	var err error
	it := &TextItem{}
	for i, c := range cols {
		switch c {
		case "text":
			it.Text, err = toString(row[i])
		case "type":
			var t int
			t, err = toInteger(row[i])
			if err == nil {
				it.Type = TextItemType(t)
			} else {
				var ts string
				ts, err = toString(row[i])
				if err == nil {
					it.Type = textItemTypeMap[ts]
				}
			}
		default:
			return fmt.Errorf("geckoboard: unknown Text column %s", c)
		}
		if err != nil {
			return fmt.Errorf("geckoboard: Text %s: %s", c, err)
		}
	}
	if it.Text != "" {
		w.Items = append(w.Items, it)
	}
	return nil
}

func (w *Leaderboard) AppendData(cols []string, row []interface{}) error {
	var err error
	it := &LeaderboardItem{}
	for i, c := range cols {
		switch c {
		case "label":
			it.Label, err = toString(row[i])
		case "value":
			it.Value, err = toFloat64(row[i])
			if err != nil {
				it.Value, err = toString(row[i])
			}
		case "previous_rank":
			it.PreviousRank, err = toInteger(row[i])
		default:
			return fmt.Errorf("geckoboard: unknown Leaderboard column %s", c)
		}
		if err != nil {
			return fmt.Errorf("geckoboard: Leaderboard %s: %s", c, err)
		}
	}
	if it.Label != "" {
		w.Items = append(w.Items, it)
	}
	return nil
}

func toString(v interface{}) (string, error) {
	switch vv := v.(type) {
	case string:
		return vv, nil
	case []byte:
		return string(vv), nil
	case int:
		return strconv.Itoa(vv), nil
	case int64:
		return strconv.FormatInt(vv, 10), nil
	case float64:
		return strconv.FormatFloat(vv, 'g', -1, 64), nil
	}
	if tm, ok := v.(encoding.TextMarshaler); ok {
		b, err := tm.MarshalText()
		return string(b), err
	}
	b, err := json.Marshal(v)
	return string(b), err
}

func toFloat64(v interface{}) (float64, error) {
	switch vv := v.(type) {
	case float64:
		return vv, nil
	case int:
		return float64(vv), nil
	case int64:
		return float64(vv), nil
	case string:
		return strconv.ParseFloat(vv, 64)
	case []byte:
		return strconv.ParseFloat(string(vv), 64)
	}
	return 0, fmt.Errorf("cannot convert type %T to float", v)
}

func toInteger(v interface{}) (int, error) {
	switch vv := v.(type) {
	case float64:
		return int(vv), nil
	case int:
		return vv, nil
	case int64:
		return int(vv), nil
	case string:
		return strconv.Atoi(vv)
	case []byte:
		return strconv.Atoi(string(vv))
	}
	return 0, fmt.Errorf("cannot convert type %T to int", v)
}
