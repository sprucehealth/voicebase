package dal

import (
	"database/sql"
	"database/sql/driver"
	"strings"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
)

type SyncBookmark struct {
	Bookmark time.Time
	Status   SyncStatus
}

type DAL interface {
	CreateSyncConfig(cfg *sync.Config, externalID *string) error
	SyncConfigForOrg(orgID, source string) (*sync.Config, error)
	SyncConfigForExternalID(externalID string) (*sync.Config, error)
	UpdateSyncBookmarkForOrg(orgID string, bookmark time.Time, status SyncStatus) error
	SyncBookmarkForOrg(orgID string) (*SyncBookmark, error)
}

type dal struct {
	db tsql.DB
}

func New(db *sql.DB) DAL {
	return &dal{
		db: tsql.AsDB(db),
	}
}

// SyncStatus is used to communicate the status of an active sync
type SyncStatus string

const (
	// SyncStatusInitiated indicates that the sync process has been initiated
	// and the initial sync is still underway.
	SyncStatusInitiated SyncStatus = "INITIATED"

	// SyncStatusConnected indicates that the initial sync is completed and we
	// are now in a connected state.
	SyncStatusConnected SyncStatus = "CONNECTED"
)

// ParseSyncStatus converts a string into the corresponding enum value
func ParseSyncStatus(s string) (SyncStatus, error) {
	switch t := SyncStatus(strings.ToUpper(s)); t {
	case SyncStatusInitiated, SyncStatusConnected:
		return t, nil
	}
	return SyncStatus(""), errors.Errorf("Unknown status:%s", s)
}

func (t SyncStatus) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t SyncStatus) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of SyncStatus from a database conforming to the sql.Scanner interface
func (t *SyncStatus) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseSyncStatus(ts)
	case []byte:
		*t, err = ParseSyncStatus(string(ts))
	}
	return errors.Trace(err)
}

var NotFound = errors.New("patientsync/dal: resource not found")

func (d *dal) CreateSyncConfig(cfg *sync.Config, externalID *string) error {
	data, err := cfg.Marshal()
	if err != nil {
		return errors.Trace(err)
	}

	_, err = d.db.Exec(`REPLACE INTO sync_config (org_id, source, config, external_id) VALUES (?,?,?,?)`,
		cfg.OrganizationEntityID,
		cfg.Source.String(),
		data,
		externalID)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (d *dal) SyncConfigForOrg(orgID, source string) (*sync.Config, error) {
	var data []byte
	if err := d.db.QueryRow(`
		SELECT config 
		FROM sync_config 
		WHERE org_id = ? and source = ?`, orgID, source).Scan(&data); err == sql.ErrNoRows {
		return nil, errors.Trace(NotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	var cfg sync.Config
	if err := cfg.Unmarshal(data); err != nil {
		return nil, errors.Trace(err)
	}

	return &cfg, nil
}

func (d *dal) SyncConfigForExternalID(externalID string) (*sync.Config, error) {
	var data []byte
	if err := d.db.QueryRow(`
		SELECT config 
		FROM sync_config 
		WHERE external_id = ?`, externalID).Scan(&data); err == sql.ErrNoRows {
		return nil, errors.Trace(NotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	var cfg sync.Config
	if err := cfg.Unmarshal(data); err != nil {
		return nil, errors.Trace(err)
	}

	return &cfg, nil
}

func (d *dal) UpdateSyncBookmarkForOrg(orgID string, bookmark time.Time, status SyncStatus) error {
	_, err := d.db.Exec(`REPLACE INTO sync_bookmark (org_id, bookmark, status) VALUES (?, ?, ?)`, orgID, bookmark, status)
	return errors.Trace(err)
}

func (d *dal) SyncBookmarkForOrg(orgID string) (*SyncBookmark, error) {
	var sb SyncBookmark
	if err := d.db.QueryRow(`
		SELECT bookmark, status
		FROM sync_bookmark
		WHERE org_id = ?`, orgID).Scan(&sb.Bookmark, &sb.Status); err == sql.ErrNoRows {
		return nil, errors.Trace(NotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &sb, nil
}
