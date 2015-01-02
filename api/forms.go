package api

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/libs/dbutil"
)

func (d *DataService) RecordForm(form Form, source string, requestID int64) error {
	tableName, columns, values := form.TableColumnValues()
	columns = append(columns, "source", "request_id")
	values = append(values, source, requestID)
	query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`, tableName, strings.Join(columns, ", "), dbutil.MySQLArgs(len(columns)))
	_, err := d.db.Exec(query, values...)
	return err
}
