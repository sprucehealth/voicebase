package dal

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"database/sql/driver"

	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
)

// ErrNotFound represents when an object cannot be found at the data layer
var ErrNotFound = errors.New("media/dal: object not found")

// DAL represents the methods required to provide data access layer functionality
type DAL interface {
	Transact(trans func(dal DAL) error) (err error)
	InsertMedia(model *Media) (MediaID, error)
	Media(id MediaID) (*Media, error)
	Medias(ids []MediaID) ([]*Media, error)
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
		return EmptyMediaID(), errors.Trace(err)
	}
	return MediaID(id), nil
}

// EmptyMediaID returns an empty initialized ID
func EmptyMediaID() MediaID {
	return ""
}

// Scan implements sql.Scanner and expects src to be nil or of type []byte, or string
func (m *MediaID) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch v := src.(type) {
	case []byte:
		*m = MediaID(string(v))
	case string:
		*m = MediaID(v)
	default:
		return errors.Trace(fmt.Errorf("unsupported type for MediaID.Scan: %T", src))
	}
	return nil
}

// ParseMediaID transforms an MediaID from it's string representation into the actual ID value
func ParseMediaID(s string) (MediaID, error) {
	if s == "" {
		return EmptyMediaID(), fmt.Errorf("Cannot parse media id: %q", s)
	}
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

// String returns a string representation of the media id
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
	// MediaOwnerTypeAccount represents the ACCOUNT state of the owner_type field on a media record
	MediaOwnerTypeAccount MediaOwnerType = "ACCOUNT"
	// MediaOwnerTypeVisit represents the VISIT state of the owner_type field on a media record
	MediaOwnerTypeVisit MediaOwnerType = "VISIT"
)

// ParseMediaOwnerType converts a string into the correcponding enum value
func ParseMediaOwnerType(s string) (MediaOwnerType, error) {
	switch t := MediaOwnerType(strings.ToUpper(s)); t {
	case MediaOwnerTypeOrganization, MediaOwnerTypeThread, MediaOwnerTypeEntity, MediaOwnerTypeAccount, MediaOwnerTypeVisit:
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
	ID         MediaID
	URL        string
	MimeType   string
	OwnerType  MediaOwnerType
	OwnerID    string
	SizeBytes  uint64
	DurationNS uint64
	Created    time.Time
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
          (mime_type, owner_type, owner_id, id, url, size_bytes, duration_ns)
          VALUES (?, ?, ?, ?, ?, ?, ?)`, model.MimeType, model.OwnerType, model.OwnerID, model.ID, model.URL, model.SizeBytes, model.DurationNS)
	if err != nil {
		return EmptyMediaID(), errors.Trace(err)
	}

	return model.ID, nil
}

// Media retrieves a media record
func (d *dal) Media(id MediaID) (*Media, error) {
	row := d.db.QueryRow(
		selectMedia+` WHERE id = ?`, id)
	model, err := scanMedia(row)
	return model, errors.Trace(err)
}

// Media retrieves a multiple media records
// TODO: I know the name 'Medias' is dumb, but want a multi read and want to preserve the
// NotFound functionality of the single read. Need to establish a pattern to merge this
func (d *dal) Medias(ids []MediaID) ([]*Media, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	vals := make([]interface{}, len(ids))
	for i, v := range ids {
		vals[i] = v
	}
	rows, err := d.db.Query(
		selectMedia+` WHERE id IN (`+dbutil.MySQLArgs(len(ids))+`)`, vals...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var media []*Media
	for rows.Next() {
		m, err := scanMedia(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		media = append(media, m)
	}
	return media, errors.Trace(rows.Err())
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
    SELECT media.owner_type, media.owner_id, media.created, media.id, media.url, media.mime_type, media.size_bytes, media.duration_ns
      FROM media`

func scanMedia(row dbutil.Scanner) (*Media, error) {
	var m Media
	m.ID = EmptyMediaID()

	err := row.Scan(&m.OwnerType, &m.OwnerID, &m.Created, &m.ID, &m.URL, &m.MimeType, &m.SizeBytes, &m.DurationNS)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &m, errors.Trace(err)
}
