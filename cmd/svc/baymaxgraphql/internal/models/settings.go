package models

type StringListSetting struct {
	Key         string                  `json:"key"`
	Subkey      string                  `json:"subkey,omitempty"`
	Title       string                  `json:"title"`
	Description string                  `json:"description"`
	Value       *StringListSettingValue `json:"value"`
}

type BooleanSetting struct {
	Key         string               `json:"key"`
	Subkey      string               `json:"subkey,omitempty"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Value       *BooleanSettingValue `json:"value"`
}

type TextSetting struct {
	Key         string            `json:"key"`
	Subkey      string            `json:"subkey,omitempty"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Value       *TextSettingValue `json:"value"`
}

type SelectableItem struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	AllowFreeText bool   `json:"allowFreeText"`
}

type SelectSetting struct {
	Key                     string                  `json:"key"`
	Subkey                  string                  `json:"subkey,omitempty"`
	Title                   string                  `json:"title"`
	Description             string                  `json:"description"`
	Options                 []*SelectableItem       `json:"options"`
	Value                   *SelectableSettingValue `json:"value"`
	AllowsMultipleSelection bool                    `json:"allowsMultipleSelection"`
}

// setting values

type StringListSettingValue struct {
	Values []string `json:"list"`
	Key    string   `json:"key"`
	Subkey string   `json:"subkey,omitempty"`
}

type BooleanSettingValue struct {
	Value  bool   `json:"set"`
	Key    string `json:"key"`
	Subkey string `json:"subkey,omitempty"`
}

type TextSettingValue struct {
	Value  string `json:"set"`
	Key    string `json:"key"`
	Subkey string `json:"subkey,omitempty"`
}

type SelectableItemValue struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type SelectableSettingValue struct {
	Items  []*SelectableItemValue `json:"items"`
	Key    string                 `json:"key"`
	Subkey string                 `json:"subkey,omitempty"`
}
