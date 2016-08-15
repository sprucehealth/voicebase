package dal

import (
	"database/sql"

	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
)

type DAL interface {
	CreateSyncConfig(cfg *sync.Config) error
	SyncConfigForOrg(orgID string) (*sync.Config, error)
}

type dal struct {
	db tsql.DB
}

func New(db *sql.DB) DAL {
	return &dal{
		db: tsql.AsDB(db),
	}
}

func (d *dal) CreateSyncConfig(cfg *sync.Config) error {
	data, err := cfg.Marshal()
	if err != nil {
		return errors.Trace(err)
	}

	_, err = d.db.Exec(`REPLACE INTO sync_config (org_id, config) VALUES (?,?)`, cfg.OrganizationEntityID, data)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (d *dal) SyncConfigForOrg(orgID string) (*sync.Config, error) {
	var data []byte
	if err := d.db.QueryRow(`
		SELECT config 
		FROM sync_config 
		WHERE org_id = ?`, orgID).Scan(&data); err != nil {
		return nil, errors.Trace(err)
	}

	var cfg sync.Config
	if err := cfg.Unmarshal(data); err != nil {
		return nil, errors.Trace(err)
	}

	return &cfg, nil
}
