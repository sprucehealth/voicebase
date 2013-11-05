package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type ReadableCondition struct {
	OperationTag         string   `json:"op"`
	IsServerCondition    bool     `json:"server_condition,omitempty"`
	QuestionTag          string   `json:"question,omitempty"`
	PotentialAnswersTags []string `json:"potential_answers,omitempty"`
	FieldTag             string   `json:"field,omitempty"`
	ValueTag             string   `json:"value,omitempty"`
}

type ReadableTipSection struct {
	TipsSectionTag string   `json:"tips_section_tag"`
	PhotoTipsTags  []string `json:"photo_tips"`
	TipsTags       []string `json:"tips"`
}

type ReadableQuestion struct {
	QuestionTag      string             `json:"question"`
	PotentialAnswers []string           `json:"potential_answers"`
	Condition        ReadableCondition  `json:"condition,omitempty"`
	IsMultiSelect    bool               `json:"multiselect,omitempty"`
	Tips             ReadableTipSection `json:"tips,omitempty"`
}

type ReadableScreen struct {
	Description string             `json:"description,omitempty"`
	Questions   []ReadableQuestion `json:"questions"`
	ScreenType  string             `json:"screen_type,omitempty"`
	Condition   ReadableCondition  `json:"condition,omitempty"`
}

type ReadableSection struct {
	SectionTag string           `json:"section"`
	Screens    []ReadableScreen `json:"screens"`
}

type ReadableTreatment struct {
	TreatmentTag string            `json:"treatment"`
	Sections     []ReadableSection `json:"sections"`
}

func main() {
	fileContents, _ := ioutil.ReadFile("condition_intake.json")
	treatmentRes := &ReadableTreatment{}
	err := json.Unmarshal(fileContents, &treatmentRes)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v", treatmentRes.Sections[0].Screens[0].Questions[0].IsMultiSelect)
}
