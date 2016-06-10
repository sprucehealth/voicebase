package dal

import (
	"database/sql"
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/layout/internal/models"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
	"golang.org/x/net/context"
)

var ErrNotFound = errors.New("not found")

type VisitLayoutUpdate struct {
	Deleted    *bool
	Name       *string
	CategoryID *models.VisitCategoryID
}

type VisitCategoryUpdate struct {
	Name    *string
	Deleted *bool
}

type DAL interface {
	Transact(context.Context, func(context.Context, DAL) error) error

	CreateVisitLayout(context.Context, *models.VisitLayout) (models.VisitLayoutID, error)
	CreateVisitLayoutVersion(context.Context, *models.VisitLayoutVersion) (models.VisitLayoutVersionID, error)
	CreateVisitCategory(context.Context, *models.VisitCategory) (models.VisitCategoryID, error)
	VisitLayout(context.Context, models.VisitLayoutID) (*models.VisitLayout, error)
	ActiveVisitLayoutVersion(context.Context, models.VisitLayoutID) (*models.VisitLayoutVersion, error)
	VisitCategory(context.Context, models.VisitCategoryID) (*models.VisitCategory, error)
	UpdateVisitCategory(context.Context, models.VisitCategoryID, *VisitCategoryUpdate) (int64, error)
	UpdateVisitLayout(context.Context, models.VisitLayoutID, *VisitLayoutUpdate) (int64, error)
	VisitLayoutVersion(context.Context, models.VisitLayoutVersionID) (*models.VisitLayoutVersion, error)
	VisitCategories() ([]*models.VisitCategory, error)
	VisitLayouts(visitCategoryID models.VisitCategoryID) ([]*models.VisitLayout, error)
}

type dal struct {
	db tsql.DB
}

func NewDAL(db *sql.DB) DAL {
	return &dal{
		db: tsql.AsDB(db),
	}
}

// Transact encapsulates the provided function in a transaction and handles rollback and commit actions
func (d *dal) Transact(ctx context.Context, trans func(context.Context, DAL) error) (err error) {
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
			errString := fmt.Sprintf("Encountered panic during transaction execution: %v", r)
			golog.Errorf(errString)
			err = errors.Trace(errors.New(errString))
		}
	}()
	if err := trans(ctx, tdal); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}
	return errors.Trace(tx.Commit())
}

func (d *dal) CreateVisitLayout(ctx context.Context, visitLayout *models.VisitLayout) (models.VisitLayoutID, error) {
	var err error
	visitLayout.ID, err = models.NewVisitLayoutID()
	if err != nil {
		return models.EmptyVisitLayoutID(), errors.Trace(err)
	}
	_, err = d.db.Exec(`INSERT INTO visit_layout (id, name, internal_name, visit_category_id) VALUES (?,?,?)`, visitLayout.ID, visitLayout.Name, visitLayout.InternalName, visitLayout.CategoryID)

	return visitLayout.ID, errors.Trace(err)
}

func (d *dal) CreateVisitLayoutVersion(ctx context.Context, visitLayoutVersion *models.VisitLayoutVersion) (models.VisitLayoutVersionID, error) {
	// validate
	if !visitLayoutVersion.VisitLayoutID.IsValid {
		return models.EmptyVisitLayoutVersionID(), errors.Trace(fmt.Errorf("visit_layout_id required"))
	} else if visitLayoutVersion.IntakeLayoutLocation == "" {
		return models.EmptyVisitLayoutVersionID(), errors.Trace(fmt.Errorf("visit_layout_location required"))
	} else if visitLayoutVersion.ReviewLayoutLocation == "" {
		return models.EmptyVisitLayoutVersionID(), errors.Trace(fmt.Errorf("visit_review_layout_location required"))
	}

	var err error
	visitLayoutVersion.ID, err = models.NewVisitLayoutVersionID()
	if err != nil {
		return models.EmptyVisitLayoutVersionID(), errors.Trace(err)
	}

	// inactivate any previous saml documents for the same visit layout
	tx, err := d.db.Begin()
	if err != nil {
		return models.EmptyVisitLayoutVersionID(), errors.Trace(err)
	}

	_, err = tx.Exec(`
		UPDATE visit_layout_version
		SET active = 0
		WHERE visit_layout_id = ?`, visitLayoutVersion.VisitLayoutID)
	if err != nil {
		tx.Rollback()
		return models.EmptyVisitLayoutVersionID(), errors.Trace(err)
	}

	_, err = d.db.Exec(`INSERT INTO visit_layout_version (id, visit_layout_id, saml_location, intake_layout_location, review_layout_location, active) VALUES (?,?,?,?,?,?)`,
		visitLayoutVersion.ID,
		visitLayoutVersion.VisitLayoutID,
		visitLayoutVersion.SAMLLocation,
		visitLayoutVersion.IntakeLayoutLocation,
		visitLayoutVersion.ReviewLayoutLocation,
		true)
	if err != nil {
		tx.Rollback()
		return models.EmptyVisitLayoutVersionID(), errors.Trace(err)
	}

	return visitLayoutVersion.ID, errors.Trace(tx.Commit())
}

func (d *dal) CreateVisitCategory(ctx context.Context, visitCategory *models.VisitCategory) (models.VisitCategoryID, error) {
	// validate
	if visitCategory.Name == "" {
		return models.EmptyVisitCategoryID(), errors.Trace(fmt.Errorf("visit_category name required"))
	}

	var err error
	visitCategory.ID, err = models.NewVisitCategoryID()
	if err != nil {
		return models.EmptyVisitCategoryID(), errors.Trace(err)
	}

	_, err = d.db.Exec(`INSERT INTO visit_category (id, name) VALUES (?,?) `, visitCategory.ID, visitCategory.Name)
	if err != nil {
		return models.EmptyVisitCategoryID(), errors.Trace(err)
	}

	return visitCategory.ID, nil
}

func (d *dal) VisitLayout(ctx context.Context, id models.VisitLayoutID) (*models.VisitLayout, error) {
	var layout models.VisitLayout
	layout.ID = models.EmptyVisitLayoutID()
	if err := d.db.QueryRow(`
		SELECT id, name, internal_name, visit_category_id, deleted
		FROM visit_layout
		WHERE id = ?
		AND deleted = 0`, id).Scan(
		&layout.ID,
		&layout.Name,
		&layout.InternalName,
		&layout.CategoryID,
		&layout.Deleted); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &layout, nil
}

func (d *dal) VisitCategory(ctx context.Context, id models.VisitCategoryID) (*models.VisitCategory, error) {
	var category models.VisitCategory
	category.ID = models.EmptyVisitCategoryID()
	if err := d.db.QueryRow(`
		SELECT id, name, deleted
		FROM visit_category
		WHERE id = ?
		AND deleted = 0`, id).Scan(
		&category.ID,
		&category.Name,
		&category.Deleted); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &category, nil
}

func (d *dal) UpdateVisitCategory(ctx context.Context, id models.VisitCategoryID, update *VisitCategoryUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.Deleted != nil {
		args.Append("deleted", *update.Deleted)
	}
	if update.Name != nil {
		args.Append("name", *update.Name)
	}

	if args == nil || args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(`
		UPDATE visit_category
		SET `+args.ColumnsForUpdate()+`
		WHERE id = ?`, append(args.Values(), id)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	rowsUpdated, err := res.RowsAffected()
	if err != nil {
		return 0, errors.Trace(err)
	}

	return rowsUpdated, nil
}

func (d *dal) UpdateVisitLayout(ctx context.Context, id models.VisitLayoutID, update *VisitLayoutUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.Deleted != nil {
		args.Append("deleted", *update.Deleted)
	}
	if update.Name != nil {
		args.Append("name", *update.Name)
	}
	if update.CategoryID != nil {
		args.Append("visit_category_id", *update.CategoryID)
	}

	if args == nil || args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(`
		UPDATE visit_layout
		SET `+args.ColumnsForUpdate()+`
		WHERE id = ?`, append(args.Values(), id)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	rowsUpdated, err := res.RowsAffected()
	if err != nil {
		return 0, errors.Trace(err)
	}

	return rowsUpdated, nil
}

func (d *dal) VisitLayoutVersion(ctx context.Context, id models.VisitLayoutVersionID) (*models.VisitLayoutVersion, error) {
	var visitLayoutVersion models.VisitLayoutVersion
	visitLayoutVersion.ID = models.EmptyVisitLayoutVersionID()
	if err := d.db.QueryRow(`
		SELECT id, visit_layout_id, saml_location, intake_layout_location, review_layout_location, active
		FROM visit_layout_version
		WHERE id = ?`, id).Scan(
		&visitLayoutVersion.ID,
		&visitLayoutVersion.VisitLayoutID,
		&visitLayoutVersion.SAMLLocation,
		&visitLayoutVersion.IntakeLayoutLocation,
		&visitLayoutVersion.ReviewLayoutLocation,
		&visitLayoutVersion.Active); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &visitLayoutVersion, nil
}

func (d *dal) ActiveVisitLayoutVersion(ctx context.Context, visitLayoutID models.VisitLayoutID) (*models.VisitLayoutVersion, error) {
	var visitLayoutVersion models.VisitLayoutVersion
	visitLayoutVersion.ID = models.EmptyVisitLayoutVersionID()
	if err := d.db.QueryRow(`
		SELECT id, visit_layout_id, saml_location, intake_layout_location, review_layout_location, active
		FROM visit_layout_version
		WHERE visit_layout_id = ?
		AND active = 1`, visitLayoutID).Scan(
		&visitLayoutVersion.ID,
		&visitLayoutVersion.VisitLayoutID,
		&visitLayoutVersion.SAMLLocation,
		&visitLayoutVersion.IntakeLayoutLocation,
		&visitLayoutVersion.ReviewLayoutLocation,
		&visitLayoutVersion.Active); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &visitLayoutVersion, nil
}

func (d *dal) VisitCategories() ([]*models.VisitCategory, error) {
	rows, err := d.db.Query(`
		SELECT id, name, deleted
		FROM visit_category
		WHERE deleted = false`)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var categories []*models.VisitCategory
	for rows.Next() {
		var category models.VisitCategory
		category.ID = models.EmptyVisitCategoryID()
		if err := rows.Scan(&category.ID, &category.Name, &category.Deleted); err != nil {
			return nil, errors.Trace(err)
		}

		categories = append(categories, &category)
	}

	return categories, errors.Trace(rows.Err())
}

func (d *dal) VisitLayouts(visitCategoryID models.VisitCategoryID) ([]*models.VisitLayout, error) {
	rows, err := d.db.Query(`
		SELECT id, name, internal_name, visit_category_id, deleted
		FROM visit_layout
		WHERE deleted = false
		AND visit_category_id = ?`, visitCategoryID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var layouts []*models.VisitLayout
	for rows.Next() {
		var layout models.VisitLayout
		layout.ID = models.EmptyVisitLayoutID()
		if err := rows.Scan(
			&layout.ID,
			&layout.Name,
			&layout.InternalName,
			&layout.CategoryID,
			&layout.Deleted); err != nil {
			return nil, errors.Trace(err)
		}

		layouts = append(layouts, &layout)
	}

	return layouts, errors.Trace(rows.Err())
}
