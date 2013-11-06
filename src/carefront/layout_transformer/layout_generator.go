package layout_transformer

import (
	"carefront/api"
)

type TreatmentLayoutProcessor struct {
	DataApi *api.DataService
}

func (c *TreatmentLayoutProcessor) TransformIntakeIntoClientLayout(treatment *Treatment) error {
	// TODO currently, calling the FillDataBaseInfo results in each section, questio, potential outcome and tip
	// making indepedent roundtrips to the database, as opposed to batch querying the database which would save time
	// and improve performance
	err := treatment.FillInDatabaseInfo(c.DataApi)
	if err != nil {
		return err
	}
	return nil
}
