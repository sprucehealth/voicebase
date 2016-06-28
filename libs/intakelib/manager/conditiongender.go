package manager

import "strings"

type genderCondition struct {
	Op     string `json:"op"`
	Gender string `json:"gender"`
}

func (g *genderCondition) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys(
		conditionTypeGenderEquals.String(),
		"op", "gender"); err != nil {
		return err
	}

	g.Op = data.mustGetString("op")
	g.Gender = data.mustGetString("gender")

	return nil
}

func (g *genderCondition) questionIDs() []string {
	return nil
}

func (g *genderCondition) evaluate(dataSource questionAnswerDataSource) bool {
	data := dataSource.valueForKey(keyTypePatientGender.String())
	if len(data) == 0 {
		return false
	}
	return strings.ToLower(g.Gender) == strings.ToLower(string(data))
}

func (g *genderCondition) layoutUnitDependencies(dataSource questionAnswerDataSource) []layoutUnit {
	return nil
}

func (g *genderCondition) staticInfoCopy(context map[string]string) interface{} {
	return &genderCondition{
		Op:     g.Op,
		Gender: g.Gender,
	}
}
