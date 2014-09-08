package api

import (
	"database/sql"
	"strings"

	"github.com/sprucehealth/backend/common"
)

func (d *DataService) ListEmailSenders() ([]*common.EmailSender, error) {
	rows, err := d.db.Query(`
		SELECT id, name, email, created, modified
		FROM email_sender
		ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var senders []*common.EmailSender
	for rows.Next() {
		snd := &common.EmailSender{}
		if err := rows.Scan(&snd.ID, &snd.Name, &snd.Email, &snd.Created, &snd.Modified); err != nil {
			return nil, err
		}
		senders = append(senders, snd)
	}

	return senders, rows.Err()
}

func (d *DataService) ListEmailTemplates(typeKey string) ([]*common.EmailTemplate, error) {
	var where string
	var args []interface{}

	if typeKey != "" {
		where = "WHERE type = ?"
		args = append(args, typeKey)
	}

	rows, err := d.db.Query(`
		SELECT id, type, name, sender_id, subject_template,
			body_text_template, body_html_template, active, created, modified
		FROM email_template
		`+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*common.EmailTemplate
	for rows.Next() {
		tmpl := &common.EmailTemplate{}
		if err := rows.Scan(&tmpl.ID, &tmpl.Type, &tmpl.Name, &tmpl.SenderID,
			&tmpl.SubjectTemplate, &tmpl.BodyTextTemplate, &tmpl.BodyHTMLTemplate,
			&tmpl.Active, &tmpl.Created, &tmpl.Modified,
		); err != nil {
			return nil, err
		}
		templates = append(templates, tmpl)
	}

	return templates, rows.Err()
}

func (d *DataService) CreateEmailSender(name, email string) (int64, error) {
	res, err := d.db.Exec(`
		INSERT INTO email_sender (name, email)
		VALUES (?, ?)`, name, email)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DataService) CreateEmailTemplate(tmpl *common.EmailTemplate) (int64, error) {
	res, err := d.db.Exec(`
		INSERT INTO email_template (type, name, sender_id, subject_template,
			body_text_template, body_html_template, active)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		tmpl.Type, tmpl.Name, tmpl.SenderID, tmpl.SubjectTemplate,
		tmpl.BodyTextTemplate, tmpl.BodyHTMLTemplate, tmpl.Active)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DataService) GetEmailSender(id int64) (*common.EmailSender, error) {
	var snd common.EmailSender
	if err := d.db.QueryRow(`
		SELECT id, name, email, created, modified
		FROM email_sender
		WHERE id = ?`, id,
	).Scan(
		&snd.ID, &snd.Name, &snd.Email, &snd.Created, &snd.Modified,
	); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	return &snd, nil
}

func (d *DataService) GetEmailTemplate(id int64) (*common.EmailTemplate, error) {
	tmpl := &common.EmailTemplate{}
	if err := d.db.QueryRow(`
		SELECT id, type, name, sender_id, subject_template,
			body_text_template, body_html_template, active, created, modified
		FROM email_template
		WHERE id = ?`, id,
	).Scan(
		&tmpl.ID, &tmpl.Type, &tmpl.Name, &tmpl.SenderID,
		&tmpl.SubjectTemplate, &tmpl.BodyTextTemplate, &tmpl.BodyHTMLTemplate,
		&tmpl.Active, &tmpl.Created, &tmpl.Modified,
	); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	return tmpl, nil
}

func (d *DataService) GetActiveEmailTemplateForType(typeKey string) (*common.EmailTemplate, error) {
	// ORDER BY RAND() is generally very inefficient, but in this case since there should
	// only be at most a handful of rows it should be fine.
	row := d.db.QueryRow(`
		SELECT id, type, name, sender_id, subject_template,
			body_text_template, body_html_template, active, created, modified
		FROM email_template
		WHERE type = ? AND active = ?
		ORDER BY RAND()
		LIMIT 1`, typeKey, true)

	var tmpl common.EmailTemplate
	if err := row.Scan(&tmpl.ID, &tmpl.Type, &tmpl.Name, &tmpl.SenderID,
		&tmpl.SubjectTemplate, &tmpl.BodyTextTemplate, &tmpl.BodyHTMLTemplate,
		&tmpl.Active, &tmpl.Created, &tmpl.Modified,
	); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	return &tmpl, nil
}

type EmailTemplateUpdate struct {
	Name             *string `json:"name,omitempty"`
	SenderID         *int64  `json:"sender_id,omitempty"`
	SubjectTemplate  *string `json:"subject_template,omitempty"`
	BodyTextTemplate *string `json:"body_text_template,omitempty"`
	BodyHTMLTemplate *string `json:"body_html_template,omitempty"`
	Active           *bool   `json:"active,omitempty"`
}

func (d *DataService) UpdateEmailTemplate(id int64, update *EmailTemplateUpdate) error {
	var cols []string
	var vals []interface{}

	if update.Name != nil {
		cols = append(cols, "name = ?")
		vals = append(vals, *update.Name)
	}
	if update.SenderID != nil {
		cols = append(cols, "sender_id = ?")
		vals = append(vals, *update.SenderID)
	}
	if update.SubjectTemplate != nil {
		cols = append(cols, "subject_template = ?")
		vals = append(vals, *update.SubjectTemplate)
	}
	if update.BodyTextTemplate != nil {
		cols = append(cols, "body_text_template = ?")
		vals = append(vals, *update.BodyTextTemplate)
	}
	if update.BodyHTMLTemplate != nil {
		cols = append(cols, "body_html_template = ?")
		vals = append(vals, *update.BodyHTMLTemplate)
	}
	if update.Active != nil {
		cols = append(cols, "active = ?")
		vals = append(vals, *update.Active)
	}

	if len(cols) == 0 {
		return nil
	}
	vals = append(vals, id)

	colStr := strings.Join(cols, ", ")
	_, err := d.db.Exec(`UPDATE email_template SET `+colStr+` WHERE id = ?`, vals...)
	return err
}
