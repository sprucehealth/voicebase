package dal

import (
	"database/sql"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
)

type DAL interface {
	LogExternalMessage(data []byte, dataType, from, to string, status string) error
}

type dal struct {
	db tsql.DB
}

func NewDAL(db *sql.DB) DAL {
	return &dal{
		db: tsql.AsDB(db),
	}
}

func (d *dal) LogExternalMessage(data []byte, dataType, from, to string, status string) error {
	_, err := d.db.Exec(`INSERT INTO externalmsg (data, type, from_endpoint_id, to_endpoint_id, status) VALUES (?,?,?,?,?)`, data, dataType, from, to, status)
	return errors.Trace(err)
}
