package info_intake

import (
	"carefront/api"
)

type TreatmentIntakeModelProcessor struct {
	DataApi api.DataAPI
}

func (c *TreatmentIntakeModelProcessor) FillInDetailsFromDatabase(treatment *Treatment, languageId int64) error {
	// TODO currently, calling the FillDataBaseInfo results in each section, questio, potential outcome and tip
	// making indepedent roundtrips to the database, as opposed to batch querying the database which would save time
	// and improve performance
	err := treatment.FillInDatabaseInfo(c.DataApi, languageId)
	if err != nil {
		return err
	}
	return nil
}
