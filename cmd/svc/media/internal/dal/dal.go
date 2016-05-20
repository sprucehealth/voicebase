package dal

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"database/sql/driver"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
)

// DAL represents the methods required to provide data access layer functionality
type DAL interface {
	Transact(trans func(dal DAL) error) (err error)
	InsertMedia(model *Media) (MediaID, error)
	Media(id MediaID) (*Media, error)
	UpdateMedia(id MediaID, update *MediaUpdate) (int64, error)
	DeleteMedia(id MediaID) (int64, error)
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

// NewMediaID returns a new MediaID.
func NewMediaID() (MediaID, error) {
	id, err := media.NewID()
	if err != nil {
		return MediaID(""), errors.Trace(err)
	}
	return MediaID(id), nil
}

// EmptyMediaID returns an empty initialized ID
func EmptyMediaID() MediaID {
	return ""
}

// ParseMediaID transforms an MediaID from it's string representation into the actual ID value
func ParseMediaID(s string) (MediaID, error) {
	return MediaID(s), nil
}

// MediaID is the ID for a MediaID object
type MediaID string

// IsValid returns a flag representing if the media id is valid or not
func (m MediaID) IsValid() bool {
	return string(m) != ""
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (m MediaID) Value() (driver.Value, error) {
	return string(m), nil
}

func (m MediaID) String() string {
	return string(m)
}

// MediaOwnerType represents the type associated with the owner_type column of the media table
type MediaOwnerType string

const (
	// MediaOwnerTypeOrganization represents the ORGANIZATION state of the owner_type field on a media record
	MediaOwnerTypeOrganization MediaOwnerType = "ORGANIZATION"
	// MediaOwnerTypeThread represents the THREAD state of the owner_type field on a media record
	MediaOwnerTypeThread MediaOwnerType = "THREAD"
	// MediaOwnerTypeEntity represents the ENTITY state of the owner_type field on a media record
	MediaOwnerTypeEntity MediaOwnerType = "ENTITY"
)

// ParseMediaOwnerType converts a string into the correcponding enum value
func ParseMediaOwnerType(s string) (MediaOwnerType, error) {
	switch t := MediaOwnerType(strings.ToUpper(s)); t {
	case MediaOwnerTypeOrganization, MediaOwnerTypeThread, MediaOwnerTypeEntity:
		return t, nil
	}
	return MediaOwnerType(""), errors.Trace(fmt.Errorf("Unknown owner_type:%s", s))
}

func (t MediaOwnerType) String() string {
	return string(t)
}

// Value implements sql/driver.Valr to allow it to be used in an sql query
func (t MediaOwnerType) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan allows for scanning of MediaOwnerType from a database conforming to the sql.Scanner interface
func (t *MediaOwnerType) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseMediaOwnerType(ts)
	case []byte:
		*t, err = ParseMediaOwnerType(string(ts))
	}
	return errors.Trace(err)
}

// Media represents a media record
type Media struct {
	ID        MediaID
	MimeType  string
	OwnerType MediaOwnerType
	OwnerID   string
	Created   time.Time
}

// MediaUpdate represents the mutable aspects of a media record
type MediaUpdate struct {
	OwnerType *MediaOwnerType
	OwnerID   *string
}

// InsertMedia inserts a media record
func (d *dal) InsertMedia(model *Media) (MediaID, error) {
	if !model.ID.IsValid() {
		id, err := NewMediaID()
		if err != nil {
			return EmptyMediaID(), errors.Trace(err)
		}
		model.ID = id
	}
	_, err := d.db.Exec(
		`INSERT INTO media
          (mime_type, owner_type, owner_id, id)
          VALUES (?, ?, ?, ?)`, model.MimeType, model.OwnerType, model.OwnerID, model.ID)
	if err != nil {
		return EmptyMediaID(), errors.Trace(err)
	}

	return model.ID, nil
}

// Media retrieves a media record
func (d *dal) Media(id MediaID) (*Media, error) {
	row := d.db.QueryRow(
		selectMedia+` WHERE id = ?`, id)
	model, err := scanMedia(row, id)
	return model, errors.Trace(err)
}

// UpdateMedia updates the mutable aspects of a media record
func (d *dal) UpdateMedia(id MediaID, update *MediaUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.OwnerType != nil {
		args.Append("owner_type", *update.OwnerType)
	}
	if update.OwnerID != nil {
		args.Append("owner_id", *update.OwnerID)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE media
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteMedia deletes a media record
func (d *dal) DeleteMedia(id MediaID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM media
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

const selectMedia = `
    SELECT media.owner_type, media.owner_id, media.created, media.id, media.mime_type
      FROM media`

func scanMedia(row dbutil.Scanner, context interface{}) (*Media, error) {
	var m Media
	m.ID = EmptyMediaID()

	err := row.Scan(&m.OwnerType, &m.OwnerID, &m.Created, &m.ID, &m.MimeType)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(api.ErrNotFound(fmt.Sprintf("media - Media not found: %+v", context)))
	}
	return &m, errors.Trace(err)
}
