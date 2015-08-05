package api

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/libs/dbutil"
)

func (d *dataService) RecordForm(form Form, source string, requestID uint64) error {
	tableName, columns, values := form.TableColumnValues()
	columns = append(columns, "source", "request_id")
	values = append(values, source, requestID)
	query := fmt.Sprintf(`REPLACE INTO %s (%s) VALUES (%s)`, dbutil.EscapeMySQLName(tableName), strings.Join(columns, ", "), dbutil.MySQLArgs(len(columns)))
	_, err := d.db.Exec(query, values...)
	return err
}

func (d *dataService) FormEntryExists(tableName, uniqueKey string) (bool, error) {
	var count int64
	err := d.db.QueryRow(`SELECT COUNT(*) FROM `+dbutil.EscapeMySQLName(tableName)+` WHERE unique_key = ?`, uniqueKey).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
