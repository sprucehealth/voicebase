package dal

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/idgen"
	modellib "github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
	"github.com/sprucehealth/backend/svc/notification"
)

// ErrNotFound is returned when an item is not found
var ErrNotFound = errors.New("notification/dal: item not found")

// DAL represents the methods required to provide data access layer functionality
type DAL interface {
	Transact(trans func(dal DAL) error) (err error)
	InsertPushConfig(model *PushConfig) (PushConfigID, error)
	PushConfig(id PushConfigID) (*PushConfig, error)
	PushConfigForDeviceID(deviceID string) (*PushConfig, error)
	PushConfigForDeviceToken(deviceToken string) (*PushConfig, error)
	PushConfigsForExternalGroupID(externalGroupID string) ([]*PushConfig, error)
	UpdatePushConfig(id PushConfigID, update *PushConfigUpdate) (int64, error)
	DeletePushConfig(id PushConfigID) (int64, error)
	DeletePushConfigForDeviceID(deviceID string) (int64, error)
}

type dal struct {
	db tsql.DB
}

// New returns an initialized instance of dal
func New(db *sql.DB) DAL {
	return &dal{db: tsql.AsDB(db)}
}

// Transact encapsulated the provided function in a transaction and handles rollback and commit actions
func (d *dal) Transact(trans func(dal DAL) error) (err error) {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}
	tdal := &dal{
		db: tsql.AsSafeTx(tx),
	}
	// Recover from any inner panics that happened and close the transaction
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			err = errors.Trace(fmt.Errorf("Encountered panic during transaction execution: %v", r))
		}
	}()
	if err := trans(tdal); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}
	return errors.Trace(tx.Commit())
}

// NewPushConfigID returns a new PushConfigID.
func NewPushConfigID() (PushConfigID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return PushConfigID{}, errors.Trace(err)
	}
	return PushConfigID{
		modellib.ObjectID{
			Prefix:  notification.PushConfigIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyPushConfigID returns an empty initialized instance of PushConfigID
func EmptyPushConfigID() PushConfigID {
	return PushConfigID{
		modellib.ObjectID{
			Prefix:  notification.PushConfigIDPrefix,
			IsValid: false,
		},
	}
}

// PushConfigID is the ID for a PushConfigID object
type PushConfigID struct {
	modellib.ObjectID
}

// PushConfig represents a push_config record
type PushConfig struct {
	ID              PushConfigID
	ExternalGroupID string
	Platform        string
	PlatformVersion string
	AppVersion      string
	DeviceID        string
	DeviceToken     []byte
	PushEndpoint    string
	Device          string
	DeviceModel     string
	Modified        time.Time
	Created         time.Time
}

// PushConfigUpdate represents the mutable aspects of a push_config record
type PushConfigUpdate struct {
	DeviceID        *string
	DeviceToken     []byte
	PushEndpoint    *string
	ExternalGroupID *string
	Platform        *string
	PlatformVersion *string
	AppVersion      *string
}

// InsertPushConfig inserts a push_config record
func (d *dal) InsertPushConfig(model *PushConfig) (PushConfigID, error) {
	if !model.ID.IsValid {
		id, err := NewPushConfigID()
		if err != nil {
			return PushConfigID{}, errors.Trace(err)
		}
		model.ID = id
	}

	_, err := d.db.Exec(
		`INSERT INTO push_config
          (platform, platform_version, app_version, device_id, id, external_group_id, device, device_model, device_token, push_endpoint)
          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, model.Platform, model.PlatformVersion, model.AppVersion, model.DeviceID, model.ID, model.ExternalGroupID, model.Device, model.DeviceModel, model.DeviceToken, model.PushEndpoint)
	if err != nil {
		return PushConfigID{}, errors.Trace(err)
	}

	return model.ID, nil
}

// PushConfig retrieves a push_config record
func (d *dal) PushConfig(id PushConfigID) (*PushConfig, error) {
	row := d.db.QueryRow(
		selectPushConfig+` WHERE id = ?`, id.Val)
	model, err := scanPushConfig(row)
	return model, errors.Trace(err)
}

// PushConfigForDeviceID retrieves a push_config record for a specific device id
func (d *dal) PushConfigForDeviceID(deviceID string) (*PushConfig, error) {
	row := d.db.QueryRow(selectPushConfig+` WHERE device_id = ?`, deviceID)
	pushConfig, err := scanPushConfig(row)
	return pushConfig, errors.Trace(err)
}

// PushConfigForDeviceToken retrieves a push_config record for a specific device token
func (d *dal) PushConfigForDeviceToken(deviceToken string) (*PushConfig, error) {
	row := d.db.QueryRow(selectPushConfig+` WHERE device_token = ?`, deviceToken)
	pushConfig, err := scanPushConfig(row)
	return pushConfig, errors.Trace(err)
}

// PushConfigsForExternalGroupID retrieves the set of push configs that map to the provided external group id
func (d *dal) PushConfigsForExternalGroupID(externalGroupID string) ([]*PushConfig, error) {
	rows, err := d.db.Query(selectPushConfig+` WHERE external_group_id = ?`, externalGroupID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var models []*PushConfig
	for rows.Next() {
		model, err := scanPushConfig(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		models = append(models, model)
	}

	return models, errors.Trace(rows.Err())
}

// UpdatePushConfig updates the mutable aspects of a push_config record
func (d *dal) UpdatePushConfig(id PushConfigID, update *PushConfigUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.DeviceID != nil {
		args.Append("device_id", update.DeviceID)
	}
	if len(update.DeviceToken) != 0 {
		args.Append("device_token", update.DeviceToken)
	}
	if update.PushEndpoint != nil {
		args.Append("push_endpoint", *update.PushEndpoint)
	}
	if update.ExternalGroupID != nil {
		args.Append("external_group_id", *update.ExternalGroupID)
	}
	if update.Platform != nil {
		args.Append("platform", *update.Platform)
	}
	if update.PlatformVersion != nil {
		args.Append("platform_version", *update.PlatformVersion)
	}
	if update.AppVersion != nil {
		args.Append("app_version", *update.AppVersion)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE push_config
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id.Val)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeletePushConfig deletes a push_config record
func (d *dal) DeletePushConfig(id PushConfigID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM push_config
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeletePushConfigForDeviceID deletes a push_config record for the specified device id
func (d *dal) DeletePushConfigForDeviceID(deviceID string) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM push_config
          WHERE device_id = ?`, deviceID)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

const selectPushConfig = `
    SELECT push_config.id, push_config.external_group_id, push_config.platform, push_config.device_id, push_config.device_model, push_config.created, push_config.modified, push_config.device_token, push_config.push_endpoint, push_config.platform_version, push_config.app_version, push_config.device
      FROM push_config`

func scanPushConfig(row dbutil.Scanner) (*PushConfig, error) {
	var m PushConfig
	m.ID = EmptyPushConfigID()

	err := row.Scan(&m.ID, &m.ExternalGroupID, &m.Platform, &m.DeviceID, &m.DeviceModel, &m.Created, &m.Modified, &m.DeviceToken, &m.PushEndpoint, &m.PlatformVersion, &m.AppVersion, &m.Device)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	}
	return &m, errors.Trace(err)
}
