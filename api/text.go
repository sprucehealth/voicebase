package api

import (
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/dbutil"
)

func (d *DataService) LocalizedText(langID int64, tags []string) (map[string]string, error) {
	rows, err := d.db.Query(`
			SELECT at.app_text_tag, lt.ltext
			FROM app_text at
			INNER JOIN localized_text lt ON lt.app_text_id = at.id AND lt.language_id = ?
			WHERE at.app_text_tag IN (`+dbutil.MySQLArgs(len(tags))+`)
		`, dbutil.AppendStringsToInterfaceSlice([]interface{}{langID}, tags)...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()
	textMap := make(map[string]string, len(tags))
	for rows.Next() {
		var tag, text string
		if err := rows.Scan(&tag, &text); err != nil {
			return nil, err
		}
		textMap[tag] = text
	}
	return textMap, rows.Err()
}

func (d *DataService) UpdateLocalizedText(langID int64, tagText map[string]string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		UPDATE localized_text
		SET ltext = ?
		WHERE language_id = ?
			AND app_text_id = (SELECT id FROM app_text WHERE app_text_tag = ?)`)
	if err != nil {
		tx.Rollback()
		return err
	}

	for tag, text := range tagText {
		if _, err := stmt.Exec(text, langID, tag); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}