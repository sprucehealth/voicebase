package feedback

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
)

var errNoFeedbackTemplate = errors.New("feedback_template doesn't exist")

// ErrNoAdditionalFeedback is an error to indicate that the patient did not provide
// any additional feebdback while leaving a rating.
var ErrNoAdditionalFeedback = errors.New("no additional feedback exists")

// ErrNoPatientFeedback is an error to indicate that patient feedback doesn't exist.
var ErrNoPatientFeedback = errors.New("patient_feedback doesn't exist")

// PatientFeedbackUpdate is struct used to update the patient feedback with the specified fields
type PatientFeedbackUpdate struct {
	// Dismissed indicates whether to update the dismised flag for the patient feedback.
	Dismissed *bool
}

// DAL represents the data access layer for feedback related functionality.
type DAL interface {

	// PatientFeedbackRecorded checks to see whether patient feedback has already been recorded
	// for the reason.
	PatientFeedbackRecorded(patientID common.PatientID, feedbackFor string) (bool, error)

	// RecordPatientFeedback records the provided feedback.
	RecordPatientFeedback(patientID common.PatientID, feedbackFor string, rating int, comment *string, structuredResponse StructuredResponse) error

	// CreatePendingPatientFeedback creates an entry to indicate that feedback from the patient is pending
	// for the specified reason.
	CreatePendingPatientFeedback(patientID common.PatientID, feedbackFor string) error

	// PatientFeedback retrieves feedback the patient may have provided for the specified reason.
	PatientFeedback(feedbackFor string) (*PatientFeedback, error)

	// AdditionalFeedback returns any structured feedback associated with the patient feedback
	AdditionalFeedback(patientFeedbackID int64) (*FeedbackTemplateData, []byte, error)

	// UpdatePatientFeedback updates the data for the specified feedback.
	UpdatePatientFeedback(feedbackFor string, update *PatientFeedbackUpdate) error

	// CreateFeedbackTemplate creates a new template with the id and type, and inactivates
	// any previous template with the same tag.
	CreateFeedbackTemplate(feedback FeedbackTemplateData) (int64, error)

	// FeedbackTemplate returns template with provided id
	FeedbackTemplate(id int64) (*FeedbackTemplateData, error)

	// ActiveFeedbackTemplate returns the current active feedback
	// template for the given tag
	ActiveFeedbackTemplate(tag string) (*FeedbackTemplateData, error)

	// ListActiveTemplates returns a list of active templates in the
	// system
	ListActiveTemplates() ([]*FeedbackTemplateData, error)

	// UpsertRatingConfig updates the rating level config for
	// the provided set of ratings
	UpsertRatingConfigs(configs map[int]string) error

	// RatingConfigs returns the current configuration
	// for each of the ratings
	RatingConfigs() (map[int]string, error)
}

type client struct {
	db *sql.DB
}

// NewDAL returns an object that interacts with the provided
// db to expose feedback functionality.
func NewDAL(db *sql.DB) DAL {
	return &client{
		db: db,
	}
}

func (c *client) PatientFeedbackRecorded(patientID common.PatientID, feedbackFor string) (bool, error) {
	var x bool
	err := c.db.QueryRow(`
		SELECT 1 
		FROM patient_feedback 
		WHERE patient_id = ? 
			AND feedback_for = ? 
			AND pending = false`, patientID, feedbackFor).Scan(&x)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return true, errors.Trace(err)
}

func (c *client) RecordPatientFeedback(patientID common.PatientID, feedbackFor string, rating int, comment *string, structuredResponse StructuredResponse) error {
	tx, err := c.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}

	res, err := tx.Exec(
		`REPLACE INTO patient_feedback  (patient_id, feedback_for, rating, comment, pending) VALUES (?, ?, ?, ?, false)`,
		patientID, feedbackFor, rating, comment)
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}
	patientFeedbackID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	if structuredResponse != (StructuredResponse)(nil) && !structuredResponse.IsZero() {
		jsonData, err := json.Marshal(structuredResponse)
		if err != nil {
			tx.Rollback()
			return errors.Trace(err)
		}

		_, err = tx.Exec(`
			REPLACE INTO patient_structured_feedback (feedback_template_id, patient_feedback_id, patient_id, json_data)
			VALUES (?,?,?,?)`, structuredResponse.TemplateID(), patientFeedbackID, patientID, jsonData)
		if err != nil {
			tx.Rollback()
			return errors.Trace(err)
		}
	}

	return errors.Trace(tx.Commit())
}

func (c *client) CreatePendingPatientFeedback(patientID common.PatientID, feedbackFor string) error {
	_, err := c.db.Exec(
		`REPLACE INTO patient_feedback (patient_id, feedback_for, pending) VALUES (?, ?, true)`,
		patientID, feedbackFor)
	return err
}

func (c *client) PatientFeedback(feedbackFor string) (*PatientFeedback, error) {
	var pf PatientFeedback
	err := c.db.QueryRow(`
		SELECT id, patient_id, rating, comment, pending, dismissed, created 
		FROM patient_feedback 
		WHERE feedback_for = ?`, feedbackFor).Scan(
		&pf.ID,
		&pf.PatientID,
		&pf.Rating,
		&pf.Comment,
		&pf.Pending,
		&pf.Dismissed,
		&pf.Created)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNoPatientFeedback)
	}
	return &pf, errors.Trace(err)
}

func (c *client) AdditionalFeedback(patientFeedbackID int64) (*FeedbackTemplateData, []byte, error) {
	var templateID int64
	var jsonData []byte
	if err := c.db.QueryRow(`
		SELECT feedback_template_id, json_data 
		FROM patient_structured_feedback
		WHERE patient_feedback_id = ?`, patientFeedbackID).Scan(
		&templateID,
		&jsonData); err == sql.ErrNoRows {
		return nil, nil, errors.Trace(ErrNoAdditionalFeedback)
	} else if err != nil {
		return nil, nil, errors.Trace(err)
	}

	ft, err := c.FeedbackTemplate(templateID)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	return ft, jsonData, nil
}

func (c *client) UpdatePatientFeedback(feedbackFor string, update *PatientFeedbackUpdate) error {
	vars := dbutil.MySQLVarArgs()

	if update.Dismissed != nil {
		vars.Append("dismissed", *update.Dismissed)
		vars.Append("pending", false)
	} else {
		return nil
	}

	_, err := c.db.Exec(`
		UPDATE patient_feedback 
		SET `+vars.Columns()+` WHERE feedback_for = ?`, append(vars.Values(), feedbackFor)...)

	return errors.Trace(err)
}

func (c *client) CreateFeedbackTemplate(feedback FeedbackTemplateData) (int64, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return 0, errors.Trace(err)
	}

	// inactivate any pre-existing template
	// with the same tag
	_, err = tx.Exec(`
		UPDATE feedback_template 
		SET active = false
		WHERE tag = ?`, feedback.Tag)
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	jsonData, err := json.Marshal(feedback.Template)
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	res, err := tx.Exec(`
		INSERT INTO feedback_template (tag, type, json_data, active) 
		VALUES (?,?,?,?)`, feedback.Tag, feedback.Type, jsonData, true)
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	templateID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	return templateID, errors.Trace(tx.Commit())
}

func (c *client) FeedbackTemplate(id int64) (*FeedbackTemplateData, error) {
	var jsonData []byte
	var templateData FeedbackTemplateData
	err := c.db.QueryRow(`
		SELECT id, tag, type, json_data, created, active
		FROM feedback_template WHERE id = ?`, id).Scan(
		&templateData.ID,
		&templateData.Tag,
		&templateData.Type,
		&jsonData,
		&templateData.Created,
		&templateData.Active)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(fmt.Errorf("feedback_template id %d not found", id))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	templateData.Template, err = TemplateFromJSON(templateData.Type, jsonData)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &templateData, nil
}

func (c *client) ActiveFeedbackTemplate(tag string) (*FeedbackTemplateData, error) {
	var jsonData []byte
	var templateData FeedbackTemplateData
	err := c.db.QueryRow(`
		SELECT id, tag, type, json_data, created, active
		FROM feedback_template WHERE tag = ? AND active = true`, tag).Scan(
		&templateData.ID,
		&templateData.Tag,
		&templateData.Type,
		&jsonData,
		&templateData.Created,
		&templateData.Active)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(errNoFeedbackTemplate)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	templateData.Template, err = TemplateFromJSON(templateData.Type, jsonData)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &templateData, nil
}

func (c *client) ListActiveTemplates() ([]*FeedbackTemplateData, error) {
	rows, err := c.db.Query(`
		SELECT id, tag, type, json_data, created, active
		FROM feedback_template WHERE active = true`)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var activeTemplates []*FeedbackTemplateData
	for rows.Next() {
		var template FeedbackTemplateData
		var jsonData []byte
		if err := rows.Scan(
			&template.ID,
			&template.Tag,
			&template.Type,
			&jsonData,
			&template.Created,
			&template.Active); err != nil {
			return nil, errors.Trace(err)
		}

		template.Template, err = TemplateFromJSON(template.Type, jsonData)
		if err != nil {
			return nil, errors.Trace(err)
		}

		activeTemplates = append(activeTemplates, &template)
	}
	return activeTemplates, errors.Trace(rows.Err())
}

func (c *client) UpsertRatingConfigs(configs map[int]string) error {
	tx, err := c.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}

	for rating, templateTagsCSV := range configs {
		// ensure that each tag currently exists and has an active template for
		templateTags := strings.Split(templateTagsCSV, ",")
		trimmedTags := make([]string, len(templateTags))

		if len(strings.TrimSpace(templateTagsCSV)) > 0 {
			for i, tag := range templateTags {
				trimmedTags[i] = strings.TrimSpace(tag)

				var id int64
				if err := tx.QueryRow(`
				SELECT id 
				FROM feedback_template 
				WHERE tag = ? AND active = true`, trimmedTags[i]).Scan(&id); err == sql.ErrNoRows {
					tx.Rollback()
					return errors.Trace(fmt.Errorf("no active template exists for tag '%s'", trimmedTags[i]))
				} else if err != nil {
					tx.Rollback()
					return errors.Trace(err)
				}
			}
		}

		_, err = tx.Exec(`
			REPLACE INTO feedback_template_config (rating, template_tags_csv) VALUES (?,?)`,
			rating, strings.Join(trimmedTags, ","))
		if err != nil {
			tx.Rollback()
			return errors.Trace(err)
		}
	}

	return errors.Trace(tx.Commit())
}

func (c *client) RatingConfigs() (map[int]string, error) {
	rows, err := c.db.Query(`
		SELECT rating, template_tags_csv
		FROM feedback_template_config`)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	configs := make(map[int]string)
	for rows.Next() {
		var rating int
		var templateTagsCSV string
		if err := rows.Scan(&rating, &templateTagsCSV); err != nil {
			return nil, errors.Trace(err)
		}

		configs[rating] = templateTagsCSV
	}

	return configs, errors.Trace(rows.Err())
}
