package tagging

import (
	"database/sql"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/tagging/model"
	"github.com/sprucehealth/backend/tagging/query"
	"github.com/sprucehealth/backend/tagging/response"
)

type Client interface {
	CaseAssociations(ms []*model.TagMembership, start, end int64) ([]*response.TagAssociation, error)
	CaseTagMemberships(caseID int64) (map[string]*model.TagMembership, error)
	DeleteTag(id int64) (int64, error)
	DeleteTagCaseAssociation(tagText string, caseID int64) error
	DeleteTagCaseMembership(tagID, caseID int64) error
	DeleteTagSavedSearch(ssID int64) (int64, error)
	InsertTag(tag *model.Tag) (int64, error)
	InsertTagAssociation(tag *model.Tag, membership *model.TagMembership) (int64, error)
	InsertTagSavedSearch(ss *model.TagSavedSearch) (int64, error)
	TagMembershipQuery(query string, pastTrigger bool) ([]*model.TagMembership, error)
	Tag(tagText string) (*response.Tag, error)
	Tags(tagText []string, common bool) ([]*response.Tag, error)
	TagSavedSearchs() ([]*model.TagSavedSearch, error)
	UpdateTag(tag *model.TagUpdate) error
	UpdateTagCaseMembership(membership *model.TagMembershipUpdate) error
}

type TaggingClient struct {
	db *sql.DB
}

func NewTaggingClient(db *sql.DB) Client {
	return &TaggingClient{db: db}
}

func (tc *TaggingClient) TagSavedSearchs() ([]*model.TagSavedSearch, error) {
	rows, err := tc.db.Query("SELECT id, title, query, created FROM tag_saved_search ORDER BY title LIMIT 100")
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	savedSearches := make([]*model.TagSavedSearch, 0)
	for rows.Next() {
		ss := &model.TagSavedSearch{}
		if err := rows.Scan(&ss.ID, &ss.Title, &ss.Query, &ss.CreatedTime); err != nil {
			return nil, errors.Trace(err)
		}
		savedSearches = append(savedSearches, ss)
	}
	return savedSearches, errors.Trace(rows.Err())
}

func (tc *TaggingClient) InsertTagSavedSearch(ss *model.TagSavedSearch) (int64, error) {
	res, err := tc.db.Exec("INSERT INTO tag_saved_search (title, query) VALUES (?, ?)", ss.Title, ss.Query)
	if err != nil {
		return 0, errors.Trace(err)
	}
	return res.LastInsertId()
}

func (tc *TaggingClient) DeleteTagSavedSearch(ssID int64) (int64, error) {
	res, err := tc.db.Exec("DELETE FROM tag_saved_search WHERE id = ?", ssID)
	if err != nil {
		return 0, errors.Trace(err)
	}
	return res.RowsAffected()
}

func (tc *TaggingClient) DeleteTag(id int64) (int64, error) {
	res, err := tc.db.Exec("DELETE FROM tag WHERE id = ?", id)
	if err != nil {
		return 0, errors.Trace(err)
	}
	return res.RowsAffected()
}

func (tc *TaggingClient) InsertTag(tag *model.Tag) (int64, error) {
	res, err := tc.db.Exec("INSERT INTO tag (tag_text, common) VALUES (?, ?)", tag.Text, tag.Common)
	if err != nil {
		return 0, errors.Trace(err)
	}
	return res.LastInsertId()
}

func (tc *TaggingClient) UpdateTag(tag *model.TagUpdate) error {
	_, err := tc.db.Exec("UPDATE tag SET common = ? WHERE id = ?", tag.Common, tag.ID)
	return errors.Trace(err)
}

func (tc *TaggingClient) Tag(text string) (*response.Tag, error) {
	tag := &response.Tag{}
	if err := tc.db.QueryRow(`SELECT id, tag_text, common FROM tag WHERE tag_text = ?`, text).Scan(&tag.ID, &tag.Text, &tag.Common); err == sql.ErrNoRows {
		return nil, api.ErrNotFound(`tag`)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return tag, nil
}

func (tc *TaggingClient) Tags(conditionValues []string, common bool) ([]*response.Tag, error) {
	q := `SELECT id, tag_text, common FROM tag`
	conditionFields := make([]string, len(conditionValues))
	for i := range conditionValues {
		conditionFields[i] = `tag_text LIKE CONCAT(?,'%')`
	}
	if len(conditionValues) > 0 {
		q += ` WHERE (` + strings.Join(conditionFields, ` OR `) + `)`
	}
	interfaceValues := dbutil.AppendStringsToInterfaceSlice(nil, conditionValues)
	if common {
		if len(conditionValues) > 0 {
			q += ` AND `
		} else {
			q += ` WHERE `
		}
		q += ` common = ?`
		interfaceValues = append(interfaceValues, common)
	}
	q += ` ORDER BY tag_text DESC LIMIT 1000`
	rows, err := tc.db.Query(q, interfaceValues...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var tags []*response.Tag
	for rows.Next() {
		tag := &model.Tag{}
		if err := rows.Scan(&tag.ID, &tag.Text, &tag.Common); err != nil {
			return nil, errors.Trace(err)
		}
		tags = append(tags, &response.Tag{ID: tag.ID, Text: tag.Text, Common: tag.Common})
	}
	return tags, errors.Trace(rows.Err())
}

func (tc *TaggingClient) DeleteTagCaseAssociation(tagText string, caseID int64) error {
	var id int64
	err := tc.db.QueryRow("SELECT id FROM tag WHERE tag_text = ?", tagText).Scan(&id)
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return errors.Trace(err)
	}

	if err := tc.DeleteTagCaseMembership(id, caseID); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (tc *TaggingClient) DeleteTagCaseMembership(tagID, caseID int64) error {
	if _, err := tc.db.Exec("DELETE FROM tag_membership WHERE tag_id = ? AND case_id = ?", tagID, caseID); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (tc *TaggingClient) UpdateTagCaseMembership(membership *model.TagMembershipUpdate) error {
	if _, err := tc.db.Exec("UPDATE tag_membership SET trigger_time = ? WHERE tag_id = ? AND case_id = ?", membership.TriggerTime, membership.TagID, membership.CaseID); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (tc *TaggingClient) InsertTagAssociation(tag *model.Tag, membership *model.TagMembership) (int64, error) {
	tx, err := tc.db.Begin()
	if err != nil {
		return 0, errors.Trace(err)
	}

	var tagID int64
	var common bool
	if err := tx.QueryRow(`SELECT id, common FROM tag WHERE tag_text=?`, tag.Text).Scan(&tagID, &common); err != nil && err != sql.ErrNoRows {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	if tagID == 0 {
		res, err := tx.Exec(`INSERT INTO tag (tag_text, common) VALUES (?, ?)`, tag.Text, tag.Common)
		if err != nil {
			tx.Rollback()
			return 0, errors.Trace(err)
		}

		tagID, err = res.LastInsertId()
		if err != nil {
			tx.Rollback()
			return 0, errors.Trace(err)
		}
	} else if common != tag.Common {
		_, err = tx.Exec(`UPDATE tag SET common = ? WHERE id = ?`, tag.Common, tagID)
		if err != nil {
			tx.Rollback()
			return 0, errors.Trace(err)
		}
	}

	_, err = tx.Exec(
		`INSERT INTO tag_membership (tag_id, case_id, trigger_time, hidden) 
      VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE case_id=case_id`, tagID, membership.CaseID, membership.TriggerTime, membership.Hidden)
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Trace(err)
	}

	return tagID, nil
}

// CaseTagMembership returns a map of tag text mapped to the corresponding membership
func (tc *TaggingClient) CaseTagMemberships(caseID int64) (map[string]*model.TagMembership, error) {
	rows, err := tc.db.Query(
		`SELECT tag.tag_text, tag_id, case_id, trigger_time, hidden, created FROM tag_membership 
			JOIN tag ON tag.id = tag_id
			WHERE case_id = ?`, caseID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	memberships := make(map[string]*model.TagMembership)
	for rows.Next() {
		var tagText string
		m := &model.TagMembership{}
		if err := rows.Scan(&tagText, &m.TagID, &m.CaseID, &m.TriggerTime, &m.Hidden, &m.Created); err != nil {
			return nil, errors.Trace(err)
		}
		memberships[tagText] = m
	}
	return memberships, errors.Trace(rows.Err())
}

func (tc *TaggingClient) TagMembershipQuery(qs string, pastTrigger bool) ([]*model.TagMembership, error) {
	q, err := query.NewTagAssociationQuery(qs)
	if err != nil {
		return nil, errors.Trace(err)
	}
	sql, v, err := q.SQL(`case_id`, tc.db)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if pastTrigger {
		if !strings.Contains(sql, `WHERE`) {
			sql += ` WHERE `
		} else if qs != "" {
			sql += ` AND `
		}
		sql += ` trigger_time <= current_timestamp `
	}

	// Limit the query to only return the latest 1000 matches
	sql += ` ORDER BY created DESC LIMIT 1000`
	rows, err := tc.db.Query(sql, v...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var memberships []*model.TagMembership
	for rows.Next() {
		m := &model.TagMembership{}
		if err := rows.Scan(&m.TagID, &m.CaseID, &m.TriggerTime, &m.Hidden, &m.Created); err != nil {
			return nil, errors.Trace(err)
		}
		memberships = append(memberships, m)
	}

	return memberships, rows.Err()
}

// CaseAssociations returns the summarized case objects for the provided memberships and visit submission bounds
func (tc *TaggingClient) CaseAssociations(ms []*model.TagMembership, start, end int64) ([]*response.TagAssociation, error) {
	if len(ms) == 0 {
		return nil, nil
	}

	if start == 0 {
		return nil, errors.Trace(errors.New("CaseAssociations query without `start` parameter not allowed"))
	}

	startTime := time.Unix(start, 0)
	endTime := time.Unix(end, 0)
	if end == 0 {
		endTime = time.Now()
	}

	ids := make([]int64, 0, len(ms))
	for _, v := range ms {
		if v.CaseID != nil {
			ids = append(ids, *v.CaseID)
		}
	}

	rows, err := tc.db.Query(
		`SELECT patient_case.id, patient.first_name, patient.last_name, clinical_pathway.name, patient_visit.submitted_date FROM patient_case
      LEFT JOIN patient ON patient_case.patient_id = patient.id
      LEFT JOIN clinical_pathway ON patient_case.clinical_pathway_id = clinical_pathway.id
	  	JOIN patient_visit ON patient_case.id = patient_visit.patient_case_id
      WHERE patient_case.id IN (`+dbutil.MySQLArgs(len(ids))+`)
      AND patient_visit.submitted_date >= ?
      AND patient_visit.submitted_date <= ?
      ORDER BY patient_visit.submitted_date DESC`, append(dbutil.AppendInt64sToInterfaceSlice(nil, ids), startTime, endTime)...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	associations := make([]*response.TagAssociation, 0, len(ms))
	submittedEpochsByCaseID := make(map[int64][]int64)
	for rows.Next() {
		var id int64
		var submittedEpoch time.Time
		tad := &response.CaseAssociationDescription{}
		if err := rows.Scan(&id, &tad.PatientFirstName, &tad.PatientLastName, &tad.Pathway, &submittedEpoch); err != nil {
			return nil, errors.Trace(err)
		}

		// If we've seen this case before and this is a seperate visit just record the next visit epoch
		if _, ok := submittedEpochsByCaseID[id]; ok {
			submittedEpochsByCaseID[id] = append(submittedEpochsByCaseID[id], submittedEpoch.Unix())
		} else {
			submittedEpochs := []int64{submittedEpoch.Unix()}
			tad.VisitSubmittedEpochs = submittedEpochs
			submittedEpochsByCaseID[id] = submittedEpochs
			associations = append(associations, &response.TagAssociation{
				ID:          id,
				Description: tad,
				Type:        response.CaseAssociationType,
			})
		}
	}
	return associations, rows.Err()
}
