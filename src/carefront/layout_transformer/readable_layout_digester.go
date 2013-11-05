package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type ReadableAnswerEqualsCondition struct {
	OperationTag         string   `json:"op"`
	QuestionTag          string   `json:"question"`
	PotentialAnswersTags []string `json:"potential_answers"`
}

type ReadableTipSection struct {
	TipsSectionTag string   `json:"tips_section_tag"`
	PhotoTipsTags  []string `json:"photo_tips"`
	TipsTags       []string `json:"tips"`
}

type ReadableQuestion struct {
	QuestionTag      string                        `json:"question"`
	PotentialAnswers []string                      `json:"potential_answers"`
	Condition        ReadableAnswerEqualsCondition `json:"condition"`
	IsMultiSelect    bool                          `json:"multiselect"`
	Tips             ReadableTipSection            `json:"tips"`
}

type ReadableScreen struct {
	Description string             `json:"description"`
	Questions   []ReadableQuestion `json:"questions"`
	ScreenType  string             `json:"screen_type"`
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
