package tagging

import (
	"database/sql"
	"strings"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/tagging/model"
	"github.com/sprucehealth/backend/tagging/query"
	"github.com/sprucehealth/backend/tagging/response"
)

type Client interface {
	CaseAssociations(ms []*model.TagMembership) ([]*response.TagAssociation, error)
	DeleteTag(id int64) (int64, error)
	DeleteTagCaseAssociation(text string, caseID int64) error
	InsertTagAssociation(text string, membership *model.TagMembership) (int64, error)
	TagMembershipQuery(query string) ([]*model.TagMembership, error)
	Tags(tagText []string) ([]*response.Tag, error)
}

type TaggingClient struct {
	db *sql.DB
}

func NewTaggingClient(db *sql.DB) Client {
	return &TaggingClient{db: db}
}

func (tc *TaggingClient) DeleteTag(id int64) (int64, error) {
	res, err := tc.db.Exec("DELETE FROM tag WHERE id = ?", id)
	if err != nil {
		return 0, errors.Trace(err)
	}
	return res.RowsAffected()
}

func (tc *TaggingClient) Tags(conditionValues []string) ([]*response.Tag, error) {
	q := `SELECT id, tag_text FROM tag`
	conditionFields := make([]string, len(conditionValues))
	for i := range conditionValues {
		conditionFields[i] = `tag_text LIKE CONCAT(?,'%')`
	}
	if len(conditionValues) > 0 {
		q += ` WHERE ` + strings.Join(conditionFields, ` OR `)
	}
	rows, err := tc.db.Query(q, dbutil.AppendStringsToInterfaceSlice(nil, conditionValues)...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var tags []*response.Tag
	for rows.Next() {
		tag := &model.Tag{}
		if err := rows.Scan(&tag.ID, &tag.Text); err != nil {
			return nil, errors.Trace(err)
		}
		tags = append(tags, &response.Tag{ID: tag.ID, Text: tag.Text})
	}
	return tags, rows.Err()
}

func (tc *TaggingClient) DeleteTagCaseAssociation(text string, caseID int64) error {
	var id int64
	err := tc.db.QueryRow("SELECT id FROM tag WHERE tag_text = ?", text).Scan(&id)
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return err
	}

	if _, err := tc.db.Exec("DELETE FROM tag_membership WHERE tag_id = ? AND case_id = ?", id, caseID); err != nil {
		return err
	}
	return nil
}

func (tc *TaggingClient) InsertTagAssociation(text string, membership *model.TagMembership) (int64, error) {
	tx, err := tc.db.Begin()
	if err != nil {
		return 0, errors.Trace(err)
	}

	var tagID int64
	if err := tx.QueryRow(`SELECT id FROM tag WHERE tag_text=?`, text).Scan(&tagID); err != nil && err != sql.ErrNoRows {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	if tagID == 0 {
		res, err := tx.Exec(`INSERT INTO tag (tag_text) VALUES (?)`, text)
		if err != nil {
			tx.Rollback()
			return 0, errors.Trace(err)
		}

		tagID, err = res.LastInsertId()
		if err != nil {
			tx.Rollback()
			return 0, errors.Trace(err)
		}
	}

	res, err := tx.Exec(
		`INSERT INTO tag_membership (tag_id, case_id, trigger_time, hidden) 
      VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE case_id=case_id`, tagID, membership.CaseID, membership.TriggerTime, membership.Hidden)
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Trace(err)
	}

	return id, nil
}

func (tc *TaggingClient) TagMembershipQuery(qs string) ([]*model.TagMembership, error) {
	q, err := query.NewTagAssociationQuery(qs)
	if err != nil {
		return nil, errors.Trace(err)
	}
	sql, v, err := q.SQL(`case_id`, tc.db)
	if err != nil {
		return nil, errors.Trace(err)
	}

	rows, err := tc.db.Query(sql, v...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var memberships []*model.TagMembership
	for rows.Next() {
		m := &model.TagMembership{}
		if err := rows.Scan(&m.TagID, &m.CaseID, &m.TriggerTime, &m.Hidden); err != nil {
			return nil, errors.Trace(err)
		}
		memberships = append(memberships, m)
	}

	return memberships, rows.Err()
}

func (tc *TaggingClient) CaseAssociations(ms []*model.TagMembership) ([]*response.TagAssociation, error) {
	if len(ms) == 0 {
		return nil, nil
	}

	ids := make([]int64, 0, len(ms))
	for _, v := range ms {
		if v.CaseID != nil {
			ids = append(ids, *v.CaseID)
		}
	}

	rows, err := tc.db.Query(
		`SELECT patient_case.id, patient.first_name, patient.last_name, clinical_pathway.name FROM patient_case
      LEFT JOIN patient ON patient_case.patient_id = patient.id
      LEFT JOIN clinical_pathway ON patient_case.clinical_pathway_id = clinical_pathway.id
      WHERE patient_case.id IN (`+dbutil.MySQLArgs(len(ids))+`)`, dbutil.AppendInt64sToInterfaceSlice(nil, ids)...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	associations := make([]*response.TagAssociation, 0, len(ms))
	for rows.Next() {
		var id int64
		var pFirst, pLast string
		tad := &response.PHISafeCaseAssociationDescription{}
		if err := rows.Scan(&id, &pFirst, &pLast, &tad.Pathway); err != nil {
			return nil, errors.Trace(err)
		}
		tad.PatientInitials = common.Initials(pFirst, pLast)
		associations = append(associations, &response.TagAssociation{
			ID:          id,
			Description: tad,
			Type:        response.CaseAssociationType,
		})
	}
	return associations, rows.Err()
}
