package dal

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
	"golang.org/x/net/context"
)

type VisitUpdate struct {
	Submitted     *bool
	SubmittedTime *time.Time
}

type DAL interface {
	Transact(context.Context, func(context.Context, DAL) error) error
	CreateVisit(context.Context, *models.Visit) (models.VisitID, error)
	Visit(ctx context.Context, id models.VisitID, opts ...QueryOption) (*models.Visit, error)
	UpdateVisit(ctx context.Context, id models.VisitID, update *VisitUpdate) (int64, error)
	CreateVisitAnswer(ctx context.Context, visitID models.VisitID, actoryEntityID string, answer *models.Answer) error
	VisitAnswers(ctx context.Context, visitID models.VisitID, questionIDs []string) (map[string]*models.Answer, error)
}

var ErrNotFound = errors.New("care/dal: not found")

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

	_, err = d.db.Exec(`INSERT INTO visit (id, name, layout_version_id, entity_id, organization_id) VALUES (?,?,?,?,?)`, id, visit.Name, visit.LayoutVersionID, visit.EntityID, visit.OrganizationID)
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
	if err := d.db.QueryRow(`
		SELECT id, name, layout_version_id, entity_id, organization_id, submitted, created, submitted_timestamp
		FROM visit
		WHERE id = ?`+forUpdate, id).Scan(
		&visit.ID,
		&visit.Name,
		&visit.LayoutVersionID,
		&visit.EntityID,
		&visit.OrganizationID,
		&visit.Submitted,
		&visit.Created,
		&visit.SubmittedTimestamp); err == sql.ErrNoRows {
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

	rowsUpdated, err := res.LastInsertId()
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
