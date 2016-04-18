package dal

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
)

// ErrNotFound is returned when an item is not found
var ErrNotFound = errors.New("blackbox/dal: item not found")

// DAL represents the methods required to provide data access layer functionality
type DAL interface {
	InsertSuiteRun(model *SuiteRun) (SuiteRunID, error)
	SuiteRun(id SuiteRunID) (*SuiteRun, error)
	IncrementSuiteRunTestPassed(id SuiteRunID) (int64, error)
	IncrementSuiteRunTestFailed(id SuiteRunID) (int64, error)
	UpdateSuiteRun(id SuiteRunID, update *SuiteRunUpdate) (int64, error)
	DeleteSuiteRun(id SuiteRunID) (int64, error)
	InsertSuiteTestRun(model *SuiteTestRun) (SuiteTestRunID, error)
	SuiteTestRun(id SuiteTestRunID) (*SuiteTestRun, error)
	UpdateSuiteTestRun(id SuiteTestRunID, update *SuiteTestRunUpdate) (int64, error)
	DeleteSuiteTestRun(id SuiteTestRunID) (int64, error)
	InsertProfile(model *Profile) (ProfileID, error)
	Profile(id ProfileID) (*Profile, error)
	DeleteProfile(id ProfileID) (int64, error)
	Transact(trans func(dal DAL) error) (err error)
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

// NewSuiteRunID returns a new SuiteRunID using the provided value. If id is 0
// then the returned SuiteRunID is tagged as invalid.
func NewSuiteRunID(id uint64) SuiteRunID {
	return SuiteRunID{
		ObjectID: encoding.ObjectID{
			Uint64Value: id,
			IsValid:     id != 0,
		},
	}
}

// SuiteRunID is the ID for a suite_run object
type SuiteRunID struct {
	encoding.ObjectID
}

// NewSuiteTestRunID returns a new SuiteTestRunID using the provided value. If id is 0
// then the returned SuiteTestRunID is tagged as invalid.
func NewSuiteTestRunID(id uint64) SuiteTestRunID {
	return SuiteTestRunID{
		ObjectID: encoding.ObjectID{
			Uint64Value: id,
			IsValid:     id != 0,
		},
	}
}

// SuiteTestRunID is the ID for a suite_test_run object
type SuiteTestRunID struct {
	encoding.ObjectID
}

// NewProfileID returns a new ProfileID using the provided value. If id is 0
// then the returned ProfileID is tagged as invalid.
func NewProfileID(id uint64) ProfileID {
	return ProfileID{
		ObjectID: encoding.ObjectID{
			Uint64Value: id,
			IsValid:     id != 0,
		},
	}
}

// ProfileID is the ID for a profile object
type ProfileID struct {
	encoding.ObjectID
}

// SuiteRunStatus represents the type associated with the status column of the suite_run table
type SuiteRunStatus string

const (
	// SuiteRunStatusRunning represents the RUNNING state of the status field on a suite_run record
	SuiteRunStatusRunning SuiteRunStatus = "RUNNING"
	// SuiteRunStatusComplete represents the COMPLETE state of the status field on a suite_run record
	SuiteRunStatusComplete SuiteRunStatus = "COMPLETE"
)

// ParseSuiteRunStatus converts a string into the correcponding enum value
func ParseSuiteRunStatus(s string) (SuiteRunStatus, error) {
	switch t := SuiteRunStatus(strings.ToUpper(s)); t {
	case SuiteRunStatusRunning, SuiteRunStatusComplete:
		return t, nil
	}
	return SuiteRunStatus(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t SuiteRunStatus) String() string {
	return string(t)
}

// Scan allows for scanning of SuiteRunStatus from a database conforming to the sql.Scanner interface
func (t *SuiteRunStatus) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseSuiteRunStatus(ts)
	case []byte:
		*t, err = ParseSuiteRunStatus(string(ts))
	}
	return errors.Trace(err)
}

// SuiteTestRunStatus represents the type associated with the status column of the suite_test_run table
type SuiteTestRunStatus string

const (
	// SuiteTestRunStatusRunning represents the RUNNING state of the status field on a suite_test_run record
	SuiteTestRunStatusRunning SuiteTestRunStatus = "RUNNING"
	// SuiteTestRunStatusPassed represents the PASSED state of the status field on a suite_test_run record
	SuiteTestRunStatusPassed SuiteTestRunStatus = "PASSED"
	// SuiteTestRunStatusFailed represents the FAILED state of the status field on a suite_test_run record
	SuiteTestRunStatusFailed SuiteTestRunStatus = "FAILED"
	// SuiteTestRunStatusErrored represents the ERRORED state of the status field on a suite_test_run record
	SuiteTestRunStatusErrored SuiteTestRunStatus = "ERRORED"
)

// ParseSuiteTestRunStatus converts a string into the correcponding enum value
func ParseSuiteTestRunStatus(s string) (SuiteTestRunStatus, error) {
	switch t := SuiteTestRunStatus(strings.ToUpper(s)); t {
	case SuiteTestRunStatusRunning, SuiteTestRunStatusPassed, SuiteTestRunStatusFailed, SuiteTestRunStatusErrored:
		return t, nil
	}
	return SuiteTestRunStatus(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t SuiteTestRunStatus) String() string {
	return string(t)
}

// Scan allows for scanning of SuiteTestRunStatus from a database conforming to the sql.Scanner interface
func (t *SuiteTestRunStatus) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseSuiteTestRunStatus(ts)
	case []byte:
		*t, err = ParseSuiteTestRunStatus(string(ts))
	}
	return errors.Trace(err)
}

// SuiteTestRun represents a suite_test_run record
type SuiteTestRun struct {
	ID         SuiteTestRunID
	SuiteRunID SuiteRunID
	TestName   string
	Status     SuiteTestRunStatus
	Message    string
	Start      time.Time
	Finish     *time.Time
}

// SuiteTestRunUpdate represents the mutable aspects of a suite_test_run record
type SuiteTestRunUpdate struct {
	Message *string
	Status  *SuiteTestRunStatus
	Finish  *time.Time
}

// SuiteRun represents a suite_run record
type SuiteRun struct {
	ID          SuiteRunID
	SuiteName   string
	Status      SuiteRunStatus
	TestsPassed int
	TestsFailed int
	Start       time.Time
	Finish      *time.Time
}

// SuiteRunUpdate represents the mutable aspects of a suite_run record
type SuiteRunUpdate struct {
	Status      *SuiteRunStatus
	TestsPassed *int
	TestsFailed *int
	Finish      *time.Time
}

// Profile represents a profile record
type Profile struct {
	ProfileKey string
	ResultMS   float64
	ID         ProfileID
	Created    time.Time
}

func (d *dal) InsertSuiteRun(model *SuiteRun) (SuiteRunID, error) {
	res, err := d.db.Exec(
		`INSERT INTO suite_run
          (finish, id, suite_name, status, tests_passed, tests_failed, start)
          VALUES (?, ?, ?, ?, ?, ?, ?)`, model.Finish, model.ID.Uint64(), model.SuiteName, model.Status.String(), model.TestsPassed, model.TestsFailed, model.Start)
	if err != nil {
		return NewSuiteRunID(0), errors.Trace(err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return NewSuiteRunID(0), errors.Trace(err)
	}

	return NewSuiteRunID(uint64(id)), nil
}

func (d *dal) SuiteRun(id SuiteRunID) (*SuiteRun, error) {
	var idv uint64
	model := &SuiteRun{}
	if err := d.db.QueryRow(
		`SELECT tests_failed, start, finish, id, suite_name, status, tests_passed
          FROM suite_run
          WHERE id = ?`, id.Uint64()).Scan(&model.TestsFailed, &model.Start, &model.Finish, &idv, &model.SuiteName, &model.Status, &model.TestsPassed); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	model.ID = NewSuiteRunID(idv)

	return model, nil
}

func (d *dal) UpdateSuiteRun(id SuiteRunID, update *SuiteRunUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.Status != nil {
		args.Append("status", update.Status.String())
	}
	if update.TestsPassed != nil {
		args.Append("tests_passed", *update.TestsPassed)
	}
	if update.TestsFailed != nil {
		args.Append("tests_failed", *update.TestsFailed)
	}
	if update.Finish != nil {
		args.Append("finish", *update.Finish)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE suite_run
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id.Uint64())...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) DeleteSuiteRun(id SuiteRunID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM suite_run
          WHERE id = ?`, id.Uint64())
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) InsertSuiteTestRun(model *SuiteTestRun) (SuiteTestRunID, error) {
	res, err := d.db.Exec(
		`INSERT INTO suite_test_run
          (message, finish, id, suite_run_id, test_name, status, start)
          VALUES (?, ?, ?, ?, ?, ?, ?)`, model.Message, model.Finish, model.ID.Uint64(), model.SuiteRunID.Uint64(), model.TestName, model.Status.String(), model.Start)
	if err != nil {
		return NewSuiteTestRunID(0), errors.Trace(err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return NewSuiteTestRunID(0), errors.Trace(err)
	}

	return NewSuiteTestRunID(uint64(id)), nil
}

func (d *dal) SuiteTestRun(id SuiteTestRunID) (*SuiteTestRun, error) {
	var idv uint64
	var suiteRunIDv uint64
	model := &SuiteTestRun{}
	if err := d.db.QueryRow(
		`SELECT id, suite_run_id, test_name, status, message, start, finish
          FROM suite_test_run
          WHERE id = ?`, id.Uint64()).Scan(&idv, &suiteRunIDv, &model.TestName, &model.Status, &model.Message, &model.Start, &model.Finish); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	model.ID = NewSuiteTestRunID(idv)
	model.SuiteRunID = NewSuiteRunID(suiteRunIDv)

	return model, nil
}

func (d *dal) UpdateSuiteTestRun(id SuiteTestRunID, update *SuiteTestRunUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.Status != nil {
		args.Append("status", update.Status.String())
	}
	if update.Message != nil {
		args.Append("message", *update.Message)
	}
	if update.Finish != nil {
		args.Append("finish", *update.Finish)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE suite_test_run
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id.Uint64())...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) IncrementSuiteRunTestPassed(id SuiteRunID) (int64, error) {
	res, err := d.db.Exec(
		`UPDATE suite_run
          SET tests_passed = tests_passed + 1 WHERE id = ?`, id.Uint64())
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) IncrementSuiteRunTestFailed(id SuiteRunID) (int64, error) {
	res, err := d.db.Exec(
		`UPDATE suite_run
          SET tests_failed = tests_failed + 1 WHERE id = ?`, id.Uint64())
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) DeleteSuiteTestRun(id SuiteTestRunID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM suite_test_run
          WHERE id = ?`, id.Uint64())
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) InsertProfile(model *Profile) (ProfileID, error) {
	res, err := d.db.Exec(
		`INSERT INTO profile
          (profile_key, result_ms)
          VALUES (?, ?)`, model.ProfileKey, model.ResultMS)
	if err != nil {
		return NewProfileID(0), errors.Trace(err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return NewProfileID(0), errors.Trace(err)
	}

	return NewProfileID(uint64(id)), nil
}

func (d *dal) Profile(id ProfileID) (*Profile, error) {
	var idv uint64
	model := &Profile{}
	if err := d.db.QueryRow(
		`SELECT profile_key, result_ms, id, created
          FROM profile
          WHERE id = ?`, id.Uint64()).Scan(&model.ProfileKey, &model.ResultMS, &idv, &model.Created); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	model.ID = NewProfileID(idv)

	return model, nil
}

func (d *dal) DeleteProfile(id ProfileID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM profile
          WHERE id = ?`, id.Uint64())
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}
