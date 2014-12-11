package main

import (
	"database/sql"
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"strings"

	_ "github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/go-sql-driver/mysql"
	"github.com/sprucehealth/backend/diagnosis/icd10"
	"github.com/sprucehealth/backend/libs/golog"
)

type icd10cm struct {
	XMLName  xml.Name   `xml:"ICD10CM.tabular"`
	Chapters []*Chapter `xml:"chapter"`
}

type Chapter struct {
	XMLName  xml.Name   `xml:"chapter"`
	Sections []*Section `xml:"section"`
}

type Section struct {
	Diagnoses []*icd10.Diagnosis `xml:"diag"`
}

var (
	tabularListLocation = flag.String("file", "", "location of the diagnosis tabular list xml")
	dbHost              = flag.String("db_host", "", "database host for where to store diagnoses")
	dbPort              = flag.Int("db_port", 3306, "database port")
	dbUserName          = flag.String("db_username", "", "database username")
	dbName              = flag.String("db_name", "", "database name")
	dbPassword          = flag.String("db_password", "", "database password")
	dbType              = flag.String("db_type", "mysql", "database type")
)

func main() {
	flag.Parse()
	golog.Default().SetLevel(golog.INFO)

	db, err := sql.Open(*dbType, fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		*dbUserName, *dbPassword, *dbHost, *dbPort, *dbName))
	if err != nil {
		golog.Fatalf(err.Error())
	}

	// test the connection to the database by running a ping against it
	if err := db.Ping(); err != nil {
		db.Close()
		golog.Fatalf(err.Error())
	}
	defer db.Close()

	file, err := os.Open(*tabularListLocation)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	tabularStructure := &icd10cm{}
	if err := xml.NewDecoder(file).Decode(tabularStructure); err != nil {
		golog.Fatalf(err.Error())
	}

	diagnosisMap := make(map[string]*icd10.Diagnosis)
	buildDiagnosisMap(tabularStructure, diagnosisMap)

	if err := icd10.SetDiagnoses(db, diagnosisMap); err != nil {
		golog.Fatalf(err.Error())
	}
}

// buildDiagnosisMap goes through each section in each chapter to identify
// categories of diagnoses to then traverse
func buildDiagnosisMap(tabularList *icd10cm, diagnosisMap map[string]*icd10.Diagnosis) {
	for _, chapter := range tabularList.Chapters {
		for _, section := range chapter.Sections {
			for _, diag := range section.Diagnoses {
				traverseDiagnosisCategory(diag, diagnosisMap)
			}
		}
	}

}

// traverseDiagnosisCategory traverses a diagnosis category to identify billable diagnoses
// located at the leaf noes of each category. The following states are possible:
// 1. A category itself can be a billable diagnosis code if itself is a leaf node and does not have a seventh
//    character definition.
// 2. If it is a leaf node and has a seventh characte definition, then the rule is that each of the seventh characters
//    applied to the category (with appropriate number of placeholders to make it the seventh character) is considered billable.
// 3. If category has subcategories without seventh character definition, then recursively traverse each of the subcategories
//    to identify billable codes.
// 4. If category has subcategories with a seventh character definition at current node, then rule is to apply seventh character
//	  to each of the leaf nodes of each of the subcategories with appropriate placeholders.
func traverseDiagnosisCategory(category *icd10.Diagnosis, diagnosisMap map[string]*icd10.Diagnosis) {

	// include the category itself
	diagnosisMap[category.Code] = category

	if len(category.SeventhCharDef) > 0 {
		for _, note := range category.SeventhCharDef {
			expandSeventhCharacterDiagnosis(category, note, diagnosisMap)
		}
		return
	}

	if len(category.Subcategories) == 0 {
		category.Billable = true
		return
	}

	for _, subcategory := range category.Subcategories {
		diagnosisMap[subcategory.Code] = subcategory
		traverseDiagnosisCategory(subcategory, diagnosisMap)
	}
}

// expandSeventhCharacterDiagnosis applies the extension as the 7th character
// to the leaf nodes of all subcategories under the current category
func expandSeventhCharacterDiagnosis(
	diag *icd10.Diagnosis,
	ext *icd10.Extension,
	diagnosisMap map[string]*icd10.Diagnosis) {

	if len(diag.Subcategories) == 0 {
		extendedDiagnosis := appendSeventhCharacter(diag, ext)
		diagnosisMap[extendedDiagnosis.Code] = extendedDiagnosis
		return
	}

	for _, subcategory := range diag.Subcategories {
		diagnosisMap[subcategory.Code] = subcategory
		expandSeventhCharacterDiagnosis(subcategory, ext, diagnosisMap)
	}
}

// appendSeventhCharacter applies the extension as the 7th character to the diagnosis
// by appending the appropriate number of placeholders and picks up the rest of the definiton
// from the current diagnosis as well.
func appendSeventhCharacter(diag *icd10.Diagnosis, ext *icd10.Extension) *icd10.Diagnosis {
	// determine number of placeholders ('X') to add
	var numX int
	var includePeriod bool

	// determine if the period exists
	if strings.ContainsRune(diag.Code, '.') {
		numX = 7 - len(diag.Code)
	} else {
		numX = 3
		includePeriod = true
	}

	code := diag.Code
	if includePeriod {
		code += "."
	}
	if numX > 0 {
		code += strings.Repeat("X", numX)
	}
	code += ext.Character

	return &icd10.Diagnosis{
		Code:              code,
		Description:       diag.Description + ", " + ext.Value,
		Includes:          diag.Includes,
		InclusionTerms:    diag.InclusionTerms,
		Excludes1:         diag.Excludes1,
		Excludes2:         diag.Excludes2,
		UseAdditionalCode: diag.UseAdditionalCode,
		CodeFirst:         diag.CodeFirst,
		Billable:          true,
	}
}
