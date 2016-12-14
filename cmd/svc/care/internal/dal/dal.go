package dal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
)

type VisitUpdate struct {
	Submitted     *bool
	SubmittedTime *time.Time
	Triaged       *bool
	TriagedTime   *time.Time
	Deleted       *bool
	DeletedTime   *time.Time
}

type VisitQuery struct {
	CreatorID        *string
	ID               *string
	OrganizationID   *string
	Submitted        *bool
	Triaged          *bool
	PatientInitiated *bool
}

type CarePlanUpdate struct {
	ParentID *string
}

type DAL interface {
	Transact(context.Context, func(context.Context, DAL) error) error

	CarePlan(context.Context, models.CarePlanID) (*models.CarePlan, error)
	CreateCarePlan(context.Context, *models.CarePlan) (models.CarePlanID, error)
	CreateVisit(context.Context, *models.Visit) (models.VisitID, error)
	CreateVisitAnswer(ctx context.Context, visitID models.VisitID, actoryEntityID string, answer *models.Answer) error
	SubmitCarePlan(ctx context.Context, id models.CarePlanID, parentID string) error
	UpdateCarePlan(ctx context.Context, id models.CarePlanID, update *CarePlanUpdate) (int64, error)
	UpdateVisit(ctx context.Context, id models.VisitID, update *VisitUpdate) (int64, error)
	Visit(ctx context.Context, id models.VisitID, opts ...QueryOption) (*models.Visit, error)
	Visits(ctx context.Context, query *VisitQuery) ([]*models.Visit, error)
	DeleteVisitAnswers(ctx context.Context, visitID models.VisitID, questionIDs []string) (int64, error)
	VisitAnswers(ctx context.Context, visitID models.VisitID, questionIDs []string) (map[string]*models.Answer, error)
}

var (
	// ErrAlreadySubmitted is returned when an object is already submitted
	ErrAlreadySubmitted = errors.New("care/dal: already submitted")
	// ErrNotFound is returned when a requested object is not found
	ErrNotFound = errors.New("care/dal: not found")
)

type carePlanInstructions struct {
	Instructions []*models.CarePlanInstruction `json:"instructions"`
}

type dal struct {
	db tsql.DB
}

type QueryOption int

const (
	// ForUpdateOpt is an option to specify when you are selecting for update
	ForUpdateOpt QueryOption = iota << 1
)

type queryOptions []QueryOption

func (qos queryOptions) Has(opt QueryOption) bool {
	for _, o := range qos {
		if o == opt {
			return true
		}
	}
	return false
}

func New(db *sql.DB) DAL {
	return &dal{
		db: tsql.AsDB(db),
	}
}

func (d *dal) Transact(ctx context.Context, trans func(context.Context, DAL) error) (err error) {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}
	tdal := &dal{
		db: tsql.AsSafeTx(tx),
	}
	// Recover from any inner panics that happened and close the transaction
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			errString := fmt.Sprintf("Encountered panic during transaction execution: %v", r)
			golog.Errorf(errString)
			err = errors.Trace(errors.New(errString))
		}
	}()
	if err := trans(ctx, tdal); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}
	return errors.Trace(tx.Commit())
}

func (d *dal) CreateVisit(ctx context.Context, visit *models.Visit) (models.VisitID, error) {
	id, err := models.NewVisitID()
	if err != nil {
		return models.EmptyVisitID(), errors.Trace(err)
	}

	_, err = d.db.Exec(`INSERT INTO visit (id, name, layout_version_id, entity_id, creator_id, organization_id, patient_initiated) 
		VALUES (?,?,?,?,?,?,?)`, id, visit.Name, visit.LayoutVersionID, visit.EntityID, visit.CreatorID, visit.OrganizationID, visit.PatientInitiated)
	if err != nil {
		return models.EmptyVisitID(), errors.Trace(err)
	}

	visit.ID = id
	return id, nil
}

func (d *dal) Visit(ctx context.Context, id models.VisitID, opts ...QueryOption) (*models.Visit, error) {
	var forUpdate string
	if queryOptions(opts).Has(ForUpdateOpt) {
		forUpdate = `
		FOR UPDATE`
	}
	var visit models.Visit
	visit.ID = models.EmptyVisitID()
	row := d.db.QueryRow(`
		SELECT id, name, layout_version_id, entity_id, creator_id, organization_id, submitted, created, submitted_timestamp, triaged, triaged_timestamp, patient_initiated
		FROM visit
		WHERE id = ?
		AND deleted = 0`+forUpdate, id)

	return scanVisit(row)
}

func (d *dal) Visits(ctx context.Context, query *VisitQuery) ([]*models.Visit, error) {
	queryStmt := `SELECT id, name, layout_version_id, entity_id, creator_id, organization_id, submitted, created, submitted_timestamp, triaged, triaged_timestamp, patient_initiated
	FROM visit`

	var whereClauses []string
	var vals []interface{}
	if query.Submitted != nil {
		whereClauses = append(whereClauses, " submitted = ?")
		vals = append(vals, *query.Submitted)
	}

	if query.Triaged != nil {
		whereClauses = append(whereClauses, " triaged = ?")
		vals = append(vals, *query.Triaged)
	}

	if query.CreatorID != nil {
		whereClauses = append(whereClauses, " creator_id = ?")
		vals = append(vals, *query.CreatorID)
	}

	if query.PatientInitiated != nil {
		whereClauses = append(whereClauses, " patient_initiated = ?")
		vals = append(vals, *query.PatientInitiated)
	}

	if query.OrganizationID != nil {
		whereClauses = append(whereClauses, " organization_id = ?")
		vals = append(vals, *query.OrganizationID)
	}

	// always exclude deleted visits
	whereClauses = append(whereClauses, " deleted = 0")

	if query.ID != nil {
		whereClauses = append(whereClauses, " id = ?")
		vals = append(vals, *query.ID)
	}

	if len(whereClauses) > 0 {
		queryStmt += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	rows, err := d.db.Query(queryStmt, vals...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var visits []*models.Visit
	for rows.Next() {
		visit, err := scanVisit(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}

		visits = append(visits, visit)
	}

	return visits, nil
}

func scanVisit(row dbutil.Scanner) (*models.Visit, error) {
	var visit models.Visit
	visit.ID = models.EmptyVisitID()
	if err := row.Scan(
		&visit.ID,
		&visit.Name,
		&visit.LayoutVersionID,
		&visit.EntityID,
		&visit.CreatorID,
		&visit.OrganizationID,
		&visit.Submitted,
		&visit.Created,
		&visit.SubmittedTimestamp,
		&visit.Triaged,
		&visit.TriagedTimestamp,
		&visit.PatientInitiated); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &visit, nil
}

func (d *dal) UpdateVisit(ctx context.Context, id models.VisitID, update *VisitUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.Submitted != nil {
		args.Append("submitted", *update.Submitted)
	}
	if update.SubmittedTime != nil {
		args.Append("submitted_timestamp", *update.SubmittedTime)
	}
	if update.Triaged != nil {
		args.Append("triaged", *update.Triaged)
	}
	if update.TriagedTime != nil {
		args.Append("triaged_timestamp", *update.TriagedTime)
	}
	if update.Deleted != nil {
		args.Append("deleted", *update.Deleted)
	}
	if update.DeletedTime != nil {
		args.Append("deleted_timestamp", *update.DeletedTime)
	}

	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(`
		UPDATE visit
		SET `+args.ColumnsForUpdate()+`
		WHERE id = ?`, append(args.Values(), id)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	rowsUpdated, err := res.RowsAffected()
	if err != nil {
		return 0, errors.Trace(err)
	}

	return rowsUpdated, nil
}

func (d *dal) CreateVisitAnswer(ctx context.Context, visitID models.VisitID, actoryEntityID string, answer *models.Answer) error {
	answerData, err := answer.Marshal()
	if err != nil {
		return errors.Trace(err)
	}

	_, err = d.db.Exec(`REPLACE INTO visit_answer (visit_id, question_id, actor_entity_id, data) VALUES (?,?,?,?)`, visitID, answer.QuestionID, actoryEntityID, answerData)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (d *dal) DeleteVisitAnswers(ctx context.Context, visitID models.VisitID, questionIDs []string) (int64, error) {
	if len(questionIDs) == 0 {
		return 0, nil
	}
	res, err := d.db.Exec(`
		DELETE FROM visit_answer
		WHERE visit_id = ?
		AND question_id in (`+dbutil.MySQLArgs(len(questionIDs))+`)`, dbutil.AppendStringsToInterfaceSlice([]interface{}{visitID}, questionIDs)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	rowsDeleted, err := res.RowsAffected()
	if err != nil {
		return 0, errors.Trace(err)
	}

	return rowsDeleted, nil
}

func (d *dal) VisitAnswers(ctx context.Context, visitID models.VisitID, questionIDs []string) (map[string]*models.Answer, error) {
	rows, err := d.db.Query(`
		SELECT  data
		FROM visit_answer
		WHERE visit_id = ?
		AND question_id in (`+dbutil.MySQLArgs(len(questionIDs))+`)`,
		dbutil.AppendStringsToInterfaceSlice([]interface{}{visitID}, questionIDs)...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	answerMap := make(map[string]*models.Answer)
	for rows.Next() {
		var answerData []byte
		if err := rows.Scan(&answerData); err != nil {
			return nil, errors.Trace(err)
		}

		var answer models.Answer
		if err := answer.Unmarshal(answerData); err != nil {
			return nil, errors.Trace(err)
		}
		answerMap[answer.QuestionID] = &answer
	}
	return answerMap, errors.Trace(rows.Err())
}

func (d *dal) CarePlan(ctx context.Context, id models.CarePlanID) (*models.CarePlan, error) {
	cp := &models.CarePlan{ID: id}
	var parentID sql.NullString
	var instructionsJSON []byte
	row := d.db.QueryRow(`
		SELECT name, creator_id, instructions_json, created, parent_id, submitted
		FROM care_plan
		WHERE id = ?`, id)
	err := row.Scan(&cp.Name, &cp.CreatorID, &instructionsJSON, &cp.Created, &parentID, &cp.Submitted)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	cp.ParentID = parentID.String
	var ins carePlanInstructions
	if err := json.Unmarshal(instructionsJSON, &ins); err != nil {
		return nil, errors.Trace(err)
	}
	cp.Instructions = ins.Instructions

	rows, err := d.db.Query(`
		SELECT id, medication_id, eprescribe, name, form, route, availability, dosage,
			dispense_type, dispense_number, refills, substitutions_allowed, days_supply, sig,
			pharmacy_id, pharmacy_instructions
		FROM care_plan_treatment
		WHERE care_plan_id = ?`, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()
	for rows.Next() {
		t := &models.CarePlanTreatment{ID: models.EmptyCarePlanTreatmentID()}
		err := rows.Scan(
			&t.ID, &t.MedicationID, &t.EPrescribe, &t.Name, &t.Form, &t.Route, &t.Availability, &t.Dosage,
			&t.DispenseType, &t.DispenseNumber, &t.Refills, &t.SubstitutionsAllowed, &t.DaysSupply, &t.Sig,
			&t.PharmacyID, &t.PharmacyInstructions)
		if err != nil {
			return nil, errors.Trace(err)
		}
		cp.Treatments = append(cp.Treatments, t)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Trace(err)
	}

	return cp, nil
}

func (d *dal) CreateCarePlan(ctx context.Context, cp *models.CarePlan) (models.CarePlanID, error) {
	id, err := models.NewCarePlanID()
	if err != nil {
		return id, errors.Trace(err)
	}

	instructionsJSON, err := json.Marshal(carePlanInstructions{Instructions: cp.Instructions})
	if err != nil {
		return id, errors.Trace(err)
	}

	tx, err := d.db.Begin()
	if err != nil {
		return id, errors.Trace(err)
	}

	_, err = tx.Exec(`INSERT INTO care_plan (id, name, creator_id, instructions_json) VALUES (?,?,?,?)`,
		id, cp.Name, cp.CreatorID, instructionsJSON)
	if err != nil {
		tx.Rollback()
		return id, errors.Trace(err)
	}

	ins := dbutil.MySQLMultiInsert(len(cp.Treatments))
	for _, t := range cp.Treatments {
		tID, err := models.NewCarePlanTreatmentID()
		if err != nil {
			return id, errors.Trace(err)
		}
		t.ID = tID
		ins.Append(tID, id, t.MedicationID, t.EPrescribe, t.Name, t.Form, t.Route, t.Availability, t.Dosage,
			t.DispenseType, t.DispenseNumber, t.Refills, t.SubstitutionsAllowed, t.DaysSupply, t.Sig,
			t.PharmacyID, t.PharmacyInstructions)
	}
	if !ins.IsEmpty() {
		_, err := tx.Exec(`
			INSERT INTO care_plan_treatment (
				id, care_plan_id, medication_id, eprescribe, name, form, route, availability, dosage,
				dispense_type, dispense_number, refills, substitutions_allowed, days_supply, sig,
				pharmacy_id, pharmacy_instructions) VALUES `+ins.Query(), ins.Values()...)
		if err != nil {
			tx.Rollback()
			return id, errors.Trace(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return id, errors.Trace(err)
	}

	cp.ID = id
	return id, nil
}

func (d *dal) SubmitCarePlan(ctx context.Context, id models.CarePlanID, parentID string) error {
	// Make sure the care plan exists to be able to return a proper error. Might as well preemptively check the submitted state at the same time.
	var submitted *time.Time
	if err := d.db.QueryRow(`SELECT submitted FROM care_plan WHERE id = ?`, id).Scan(&submitted); err == sql.ErrNoRows {
		return errors.Trace(ErrNotFound)
	} else if err != nil {
		return errors.Trace(err)
	}
	if submitted != nil {
		return errors.Trace(ErrAlreadySubmitted)
	}
	res, err := d.db.Exec(`UPDATE care_plan SET submitted = NOW(), parent_id = ? WHERE id = ? AND submitted IS NULL`, parentID, id)
	if err != nil {
		return errors.Trace(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return errors.Trace(err)
	}
	if n == 0 {
		return errors.Trace(ErrAlreadySubmitted)
	}
	return nil
}

func (d *dal) UpdateCarePlan(ctx context.Context, id models.CarePlanID, update *CarePlanUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update == nil {
		return 0, nil
	}
	if update.ParentID != nil {
		args.Append("parent_id", *update.ParentID)
	}

	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(`
		UPDATE care_plan
		SET `+args.ColumnsForUpdate()+`
		WHERE id = ?`, append(args.Values(), id)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	rowsUpdated, err := res.RowsAffected()
	if err != nil {
		return 0, errors.Trace(err)
	}

	return rowsUpdated, nil
}
