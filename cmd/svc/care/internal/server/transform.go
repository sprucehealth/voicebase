package server

import (
	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	"github.com/sprucehealth/backend/svc/care"
)

func transformVisitToResponse(v *models.Visit) *care.Visit {
	return &care.Visit{
		ID:              v.ID.String(),
		Name:            v.Name,
		Submitted:       v.Submitted,
		LayoutVersionID: v.LayoutVersionID,
		EntityID:        v.EntityID,
	}
}
