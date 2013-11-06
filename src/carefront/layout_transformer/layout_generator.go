package layout_transformer

import (
	"carefront/api"
)

type ClientLayoutProcessor struct {
	DataApi *api.DataService
}

func (c *ClientLayoutProcessor) TransformIntakeIntoClientLayout(treatment *Treatment) error {
	err := treatment.FillInDatabaseInfo(c.DataApi)
	if err != nil {
		return err
	}
	return nil
}
