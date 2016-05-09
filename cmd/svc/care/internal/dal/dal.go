package dal

import (
	"database/sql"

	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
	"golang.org/x/net/context"
)

type DAL interface {
	CreateVisit(context.Context, *models.Visit) (models.VisitID, error)
	Visit(ctx context.Context, id models.VisitID) (*models.Visit, error)
}

var ErrNotFound = errors.New("care/dal: not found")

type dal struct {
	db tsql.DB
}

func New(db *sql.DB) DAL {
	return &dal{
		db: tsql.AsDB(db),
	}
}

func (d *dal) CreateVisit(ctx context.Context, visit *models.Visit) (models.VisitID, error) {
	id, err := models.NewVisitID()
	if err != nil {
		return models.EmptyVisitID(), errors.Trace(err)
	}

	_, err = d.db.Exec(`INSERT INTO visit (id, name, layout_version_id, entity_id) VALUES (?,?,?,?)`, id, visit.Name, visit.LayoutVersionID, visit.EntityID)
	if err != nil {
		return models.EmptyVisitID(), errors.Trace(err)
	}

	visit.ID = id
	return id, nil
}

func (d *dal) Visit(ctx context.Context, id models.VisitID) (*models.Visit, error) {
	var visit models.Visit
	if err := d.db.QueryRow(`
		SELECT id, name, layout_version_id, entity_id, submitted, created, submitted_timestamp
		FROM visit
		WHERE id = ?`, id).Scan(
		&visit.ID,
		&visit.Name,
		&visit.LayoutVersionID,
		&visit.EntityID,
		&visit.EntityID,
		&visit.Submitted,
		&visit.Created,
		&visit.SubmittedTimestamp); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &visit, nil
}
