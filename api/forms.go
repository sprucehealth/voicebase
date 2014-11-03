package api

import (
	"fmt"
	"strings"
)

func (d *DataService) RecordForm(form Form, source string, requestID int64) error {
	tableName, columns, values := form.TableColumnValues()
	columns = append(columns, "source", "request_id")
	values = append(values, source, requestID)
	query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`, tableName, strings.Join(columns, ", "), nReplacements(len(columns)))
	_, err := d.db.Exec(query, values...)
	return err
}
