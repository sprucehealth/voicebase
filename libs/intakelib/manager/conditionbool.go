package manager

import (
	"github.com/sprucehealth/backend/libs/errors"
)

type boolCondition struct {
	Op         string `json:"op"`
	BoolValue  bool   `json:"bool_value"`
	DataSource string `json:"data_source"`
}

func (b *boolCondition) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys("bool_condition", "op", "bool_value", "data_source"); err != nil {
		return errors.Trace(err)
	}

	b.Op = data.mustGetString("op")
	b.BoolValue = data.mustGetBool("bool_value")
	b.DataSource = data.mustGetString("data_source")

	return nil
}

func (b *boolCondition) questionIDs() []string {
	return nil
}

func (b *boolCondition) evaluate(dataSource questionAnswerDataSource) bool {
	data := dataSource.valueForKey(b.DataSource)
	boolValue, ok := data.(bool)
	if !ok {
		return false
	}

	return boolValue == b.BoolValue
}

func (a *boolCondition) layoutUnitDependencies(dataSource questionAnswerDataSource) []layoutUnit {
	return nil
}

func (i *boolCondition) staticInfoCopy(context map[string]string) interface{} {
	return &boolCondition{
		Op:         i.Op,
		DataSource: i.DataSource,
		BoolValue:  i.BoolValue,
	}
}
