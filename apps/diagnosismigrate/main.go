package main

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/go-sql-driver/mysql"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/diagnosis/icd10"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/golog"
)

// command line options
var dbHost = flag.String("db_host", "", "mysql database host")
var dbPort = flag.Int("dp_port", 3306, "mysql database port")
var dbName = flag.String("db_name", "", "mysql database name")
var dbUsername = flag.String("db_username", "", "mysql database username")
var dbPassword = flag.String("db_password", "", "mysql database password")
var listCSV = flag.String("csv", "", "csv file with icd10 mappings")

// top level diagnosis question tags
var qTagDiagnosis = "q_acne_diagnosis"
var qTagDiagnosisSeverity = "q_acne_severity"
var qTagAcneType = "q_acne_type"
var qTagRosaceaType = "q_acne_rosacea_type"
var qTagDiagnosisDescription = "q_diagnosis_describe_condition"
var qTagNotSuitableReason = "q_diagnosis_reason_not_suitable"

// diagnosis details question tags
var qTagDiagDetailsSeverity = "q_diagnosis_severity"
var qTagDiagDetailsAcneType = "q_diagnosis_acne_vulgaris_type"
var qTagDiagDetailsRosaceaType = "q_diagnosis_acne_rosacea_type"

// top level diagnosis answers
var aAcneVulgars = "Acne vulgaris"
var aRosacea = "Acne rosacea"
var aPerioralDermatitis = "Perioral dermatitis"
var aNotSuitable = "Not Suitable For Spruce"
var aOther = "Other"

// answerChoiceToICD10Mapping maps existing answer choices to ICD10 codes wherever possible
var answerChoiceToICD10CodeMapping = map[string]string{
	aAcneVulgars:        "diag_l700",
	aRosacea:            "diag_l719",
	aPerioralDermatitis: "diag_l710",
}

// diagnosisDetailsQuestionToTopLevelQuestionMapping creates a mapping from the questionTag for the
// top level diagnosis question in the old model to the questionTag for the diagnosis details
// question in the new model.
var diagnosisDetailsQuestionToTopLevelQuestionMapping = map[string]string{
	qTagDiagDetailsSeverity:    qTagDiagnosisSeverity,
	qTagDiagDetailsAcneType:    qTagAcneType,
	qTagDiagDetailsRosaceaType: qTagRosaceaType,
}

var errIgnored = errors.New("Ignore adding diagnosis intake")

type context struct {
	tx                *sql.Tx
	pair              *visitDoctorPair
	selectedDiagnosis string
	intakeItemsMap    map[string][]*diagnosisIntakeItem
	intakeItems       []*diagnosisIntakeItem
}

// migrationFunctions keeps track of all the migration fucntions.
var migrationFuncs = map[string]func(*context, dataManagerIFace, diagnosisDetailsIFace, map[visitDoctorPair][]string) error{
	aAcneVulgars:        migrateCommonDiagnosisChoiceToNewModel,
	aRosacea:            migrateCommonDiagnosisChoiceToNewModel,
	aPerioralDermatitis: migrateCommonDiagnosisChoiceToNewModel,
	aOther:              migrateCustomDiagnosisChoiceToNewModel,
	aNotSuitable:        migrateNotSuitableDiagnosisToNewModel,
}

func main() {
	flag.Parse()
	golog.Default().SetLevel(golog.INFO)

	// connect to the database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&loc=Local&interpolateParams=true",
		*dbUsername, *dbPassword, *dbHost, *dbPort, *dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	err = func() error {

		// test the connection to the database by running a ping against it
		if err := db.Ping(); err != nil {
			return err
		}

		dataAPI, err := api.NewDataService(db)
		if err != nil {
			return err
		}

		dManager := &dataManager{
			db: db,
		}

		// load the csv into memory as a mapping of [patient_visit_id:doctor_id] -> []icd10 codes
		diagnosisMapping, err := createMapFromCSV(*listCSV)
		if err != nil {
			return err
		}

		tx, err := dManager.beginTransaction()
		if err != nil {
			return err
		}

		if err := migrateDiagnosisIntakeToNewModel(tx, diagnosisMapping, dManager, dataAPI); err != nil {
			dManager.rollbackTransaction(tx)
			return err
		}

		if err := dManager.commitTransaction(tx); err != nil {
			return err
		}
		return nil
	}()

	db.Close()

	if err != nil {
		golog.Fatalf(err.Error())
	}
}

func migrateDiagnosisIntakeToNewModel(
	tx *sql.Tx,
	diagnosisMapping map[visitDoctorPair][]string,
	dManager dataManagerIFace,
	diagDetails diagnosisDetailsIFace) error {

	// lookup all the distinct (patient_visit_id, doctor_id) pairs in diagnosis_intake.
	// only consider the intakes from Aug 2014 given that its around the time we launched.
	pairs, err := dManager.distinctVisitDoctorPairsSince(time.Date(2014, 8, 01, 0, 0, 0, 0, time.UTC))
	if err != nil {
		return err
	}

	for _, pair := range pairs {

		// skip if diagnosis set already exists
		if exists, err := dManager.diagnosisSetExistsForPair(pair); err != nil {
			return err
		} else if exists {
			golog.Warningf("EXISTS: Diagnosis info for [visitID: %d, doctorID: %d]", pair.visitID, pair.doctorID)
			continue
		}

		// lookup the diagnosis intake by the doctor for the visit
		intakeItems, err := dManager.diagnosisItems(pair)
		if err != nil {
			return err
		}

		intakeItemsMap := make(map[string][]*diagnosisIntakeItem)
		for _, intakeItem := range intakeItems {
			items := intakeItemsMap[intakeItem.questionTag]
			intakeItemsMap[intakeItem.questionTag] = append(items, intakeItem)
		}

		// determine the diagnosis choice selected
		var selectedDiagnosis string
		intakeItems, ok := intakeItemsMap[qTagDiagnosis]
		if ok {
			selectedDiagnosis = *intakeItems[0].answerSelected
		} else {
			return fmt.Errorf("Unable to find diagnosis selection for [visitID: %d, doctorID: %d]", pair.visitID, pair.doctorID)
		}

		ctx := &context{
			tx:                tx,
			selectedDiagnosis: selectedDiagnosis,
			pair:              pair,
			intakeItemsMap:    intakeItemsMap,
			intakeItems:       intakeItems,
		}

		migrationFunction, ok := migrationFuncs[selectedDiagnosis]
		if !ok {
			return fmt.Errorf("Unable to find migration function for %s", selectedDiagnosis)
		}

		if err := migrationFunction(ctx, dManager, diagDetails, diagnosisMapping); err != errIgnored && err != nil {
			return err
		} else if err != errIgnored {
			golog.Infof("CREATED: Diagnosis info for [visitID: %d, doctorID: %d]", pair.visitID, pair.doctorID)
		}
	}

	return nil
}

func migrateNotSuitableDiagnosisToNewModel(
	ctxt *context,
	dManager dataManagerIFace,
	diagDetails diagnosisDetailsIFace,
	diagnosisMapping map[visitDoctorPair][]string,
) error {
	// determine the reason for being marked as unsuitable
	var unsuitableReason string
	items, ok := ctxt.intakeItemsMap[qTagNotSuitableReason]
	if ok {
		unsuitableReason = items[0].text
	}

	// create a diagnosis set to indicate the visit as being unsuitable for spruce
	diagnosisSet := &common.VisitDiagnosisSet{
		VisitID:          ctxt.pair.visitID,
		DoctorID:         ctxt.pair.doctorID,
		Created:          items[0].created,
		Unsuitable:       true,
		UnsuitableReason: unsuitableReason,
	}

	if err := dManager.createVisitDiagnosisSet(ctxt.tx, diagnosisSet); err != nil {
		return err
	}

	return nil
}

func migrateCustomDiagnosisChoiceToNewModel(
	ctxt *context,
	dManager dataManagerIFace,
	diagDetails diagnosisDetailsIFace,
	diagnosisMapping map[visitDoctorPair][]string,
) error {
	// check if the pair exists in the csv mapping identified
	// as a pair that needed MD intervention
	// to identify the set of ICD10 codes
	icd10Codes, ok := diagnosisMapping[*ctxt.pair]
	if ok {
		// prepare the visit diagnosis set
		diagnosisSet := &common.VisitDiagnosisSet{
			VisitID:  ctxt.pair.visitID,
			DoctorID: ctxt.pair.doctorID,
			Created:  ctxt.intakeItems[0].created,
			Items:    make([]*common.VisitDiagnosisItem, len(icd10Codes)),
		}

		for i, icd10Code := range icd10Codes {
			diagnosisSet.Items[i] = &common.VisitDiagnosisItem{
				CodeID: icd10Code,
			}
		}

		if err := dManager.createVisitDiagnosisSet(ctxt.tx, diagnosisSet); err != nil {
			return err
		}

		return nil
	}

	// determine the custom diagnosis
	var customDiagnosis string
	items, ok := ctxt.intakeItemsMap[qTagDiagnosisDescription]
	if ok {
		customDiagnosis = items[0].text
	}

	golog.Errorf("IGNORED: Diagnosis '%s' for [visitID: %d, doctorID: %d]", customDiagnosis, ctxt.pair.visitID, ctxt.pair.doctorID)
	return errIgnored
}

func migrateCommonDiagnosisChoiceToNewModel(
	ctxt *context,
	dManager dataManagerIFace,
	diagDetails diagnosisDetailsIFace,
	diagnosisMapping map[visitDoctorPair][]string,
) error {

	codeID, ok := answerChoiceToICD10CodeMapping[ctxt.selectedDiagnosis]
	if !ok {
		golog.Errorf("IGNORED: Unable to determine icd10 code for selection %s [visitID: %d, doctorID: %d]", ctxt.selectedDiagnosis, ctxt.pair.visitID, ctxt.pair.doctorID)
		return errIgnored
	}

	// lookup any existing diagnosis details layout
	diagnosisDetails, err := diagDetails.ActiveDiagnosisDetailsIntake(codeID, diagnosis.DetailTypes)
	if err != nil && !api.IsErrNotFound(err) {
		return err
	}

	var layoutVersionID *int64
	if diagnosisDetails != nil {
		layoutVersionID = &diagnosisDetails.ID
	}

	// create the diagnosis set
	diagnosisSet := &common.VisitDiagnosisSet{
		VisitID:  ctxt.pair.visitID,
		DoctorID: ctxt.pair.doctorID,
		Created:  ctxt.intakeItems[0].created,
		Items: []*common.VisitDiagnosisItem{
			{
				CodeID:          codeID,
				LayoutVersionID: layoutVersionID,
			},
		},
	}

	if err := dManager.createVisitDiagnosisSet(ctxt.tx, diagnosisSet); err != nil {
		return err
	}

	// create diagnosis details intake items from the top level diagnosis answers
	if diagnosisDetails != nil {
		for _, question := range diagnosisDetails.Layout.(*diagnosis.QuestionIntake).Questions() {

			topLevelQuestionTag, ok := diagnosisDetailsQuestionToTopLevelQuestionMapping[question.QuestionTag]
			if !ok {
				golog.Warningf("DiagDetails: Skipped question %s for [visitID: %d, doctorID: %d]",
					question.QuestionTag, ctxt.pair.visitID, ctxt.pair.doctorID)
				return nil
			}

			items, ok := ctxt.intakeItemsMap[topLevelQuestionTag]
			if !ok {
				golog.Warningf("DiagDetails: No top level intake found for question %s for [visitID: %d, doctorID: %d]",
					topLevelQuestionTag, ctxt.pair.visitID, ctxt.pair.doctorID)
				return nil
			}

			for _, intakeItem := range items {
				var potentialAnswer *info_intake.PotentialAnswer
				for _, pa := range question.PotentialAnswers {
					if pa.Answer == *intakeItem.answerSelected {
						potentialAnswer = pa
						break
					}
				}

				if potentialAnswer == nil {
					golog.Warningf("DiagDetails: No answer choice for in diagnosis detals for answer %s for [visitID: %d, doctorID: %d]",
						*intakeItem.answerSelected, ctxt.pair.visitID, ctxt.pair.doctorID)
					return nil
				}

				// create diagnosis details intake
				if err := dManager.createDiagnosisDetailsIntake(ctxt.tx, &diagnosisDetailsIntake{
					visitDiagnosisItemID: diagnosisSet.Items[0].ID,
					visitID:              ctxt.pair.visitID,
					doctorID:             ctxt.pair.doctorID,
					layoutVersionID:      diagnosisDetails.ID,
					answeredDate:         intakeItem.created,
					questionID:           question.QuestionID,
					potentialAnswerID:    &potentialAnswer.AnswerID,
				}); err != nil {
					return err
				}
			}

		}
	}

	return nil
}

// createMapFromCSV reads the CSV file and creates a map of
// [patient_visit_id,doctor_id] -> array of icd10 codes
func createMapFromCSV(fileName string) (map[visitDoctorPair][]string, error) {
	csvFile, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	diagnosisMapping := make(map[visitDoctorPair][]string)
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		visitID, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil {
			return nil, err
		}

		doctorID, err := strconv.ParseInt(row[1], 10, 64)
		if err != nil {
			return nil, err
		}

		pair := visitDoctorPair{
			visitID:  visitID,
			doctorID: doctorID,
		}

		codes := strings.Split(row[3], ",")

		icd10Codes := make([]string, len(codes))
		for i, code := range codes {
			icd10Codes[i] = icd10.Code(code).Key()
		}
		diagnosisMapping[pair] = icd10Codes
	}
	return diagnosisMapping, nil
}
