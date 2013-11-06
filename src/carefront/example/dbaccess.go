package main

import (
	"carefront/api"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

func main() {

	db, err := sql.Open("mysql", "carefront:changethis@tcp(dev-db-3.ccvrwjdx3gvp.us-east-1.rds.amazonaws.com:3306)/carefront_db")

	if err != nil {
		panic(err.Error())
	}

	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	dataApi := &api.DataService{db}
	treatmentId, err := dataApi.GetTreatmentInfo("treatment_acne", 1)
	if err != nil {
		panic(err)
	}
	fmt.Printf("treatment id is %d", treatmentId)

	sectionId, sectionTitle, err := dataApi.GetSectionInfo("section_skin_hisory", 1)
	if err != nil {
		panic(err)
	}
	fmt.Printf("\n%d %s", sectionId, sectionTitle)

	questionId, questionTitle, questionType, err := dataApi.GetQuestionInfo("q_reason_visit", 1)
	if err != nil {
		panic(err)
	}
	fmt.Println(questionId, questionTitle, questionType)

	outcomeId, outcome, outcomeType, err := dataApi.GetOutcomeInfo("o_something_else", 1)
	if err != nil {
		panic(err)
	}
	fmt.Println(outcomeId, outcome, outcomeType)

	tipSectionId, tipSectionTitle, tipSectionSubtext, err := dataApi.GetTipSectionInfo("tips_section_face", 1)
	if err != nil {
		panic(err)
	}
	fmt.Println(tipSectionId, tipSectionTitle, tipSectionSubtext)

	tipId, tip, err := dataApi.GetTipInfo("tip_remove_jewellery", 1)
	if err != nil {
		panic(err)
	}
	fmt.Println(tipId, tip)
}
