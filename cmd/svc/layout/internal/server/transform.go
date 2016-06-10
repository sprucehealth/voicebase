package server

import (
	"github.com/sprucehealth/backend/cmd/svc/layout/internal/models"
	"github.com/sprucehealth/backend/svc/layout"
)

func transformCategoryToResponse(category *models.VisitCategory) *layout.VisitCategory {
	return &layout.VisitCategory{
		ID:   category.ID.String(),
		Name: category.Name,
	}
}

func transformVisitLayoutToResponse(visitLayout *models.VisitLayout, layoutVersion *models.VisitLayoutVersion) *layout.VisitLayout {
	tVisitLayout := &layout.VisitLayout{
		ID:           visitLayout.ID.String(),
		Name:         visitLayout.Name,
		InternalName: visitLayout.InternalName,
		CategoryID:   visitLayout.CategoryID.String(),
	}

	if layoutVersion != nil {
		tVisitLayout.Version = &layout.VisitLayoutVersion{
			ID:                   layoutVersion.ID.String(),
			SAMLLocation:         layoutVersion.SAMLLocation,
			IntakeLayoutLocation: layoutVersion.IntakeLayoutLocation,
			ReviewLayoutLocation: layoutVersion.ReviewLayoutLocation,
		}
	}

	return tVisitLayout
}

func transformVisitLayoutVersionToResponse(visitLayoutVersion *models.VisitLayoutVersion) *layout.VisitLayoutVersion {
	return &layout.VisitLayoutVersion{
		ID:                   visitLayoutVersion.ID.String(),
		SAMLLocation:         visitLayoutVersion.SAMLLocation,
		ReviewLayoutLocation: visitLayoutVersion.ReviewLayoutLocation,
		IntakeLayoutLocation: visitLayoutVersion.IntakeLayoutLocation,
	}
}
