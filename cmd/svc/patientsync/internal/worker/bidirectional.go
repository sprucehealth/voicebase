package worker

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/source/hint"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
)

// SyncEntityUpdate checks if the entity updated in Spruce is linked to a
// patient in an external system, and syncs the update from Spruce to the external system
func SyncEntityUpdate(dirCLI directory.DirectoryClient, dl dal.DAL, ev *directory.EntityUpdatedEvent) error {
	entity, err := directory.SingleEntity(context.Background(), dirCLI, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: ev.EntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_EXTERNAL_IDS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_PATIENT, directory.EntityType_EXTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	})
	if err != nil && errors.Cause(err) != directory.ErrEntityNotFound {
		return errors.Errorf("unable to lookup entity %s : %s", ev.EntityID, err)
	} else if errors.Cause(err) == directory.ErrEntityNotFound {
		golog.Warningf("entity %s not found", ev.EntityID)
		// nothing to do
		return nil
	}

	// check if any of the external ids represents an external system
	var source sync.Source
	var entityExternalID string
	for _, externalID := range entity.ExternalIDs {
		if ss, err := sync.SourceFromExternalID(externalID); err == nil && ss != sync.SOURCE_UNKNOWN {
			source = ss
			entityExternalID = externalID
			break
		}
	}

	if source == sync.SOURCE_UNKNOWN {
		golog.Warningf("unknown source for entity %s", ev.EntityID)
		// nothing to do since patient is not linked to an external system
		return nil
	}

	var orgID string
	for _, membership := range entity.Memberships {
		if membership.Type == directory.EntityType_ORGANIZATION {
			orgID = membership.ID
			break
		}
	}

	syncConfig, err := dl.SyncConfigForOrg(orgID, source.String())
	if err != nil {
		return errors.Errorf("Unable to get sync config for org %s : %s", orgID, err)
	}

	externalPatientID, err := sync.IDForSource(entityExternalID)
	if err != nil {
		return errors.Errorf("unable to parse external patient id from '%s' : %s", entityExternalID, err)
	}

	switch source {
	case sync.SOURCE_HINT:
		if err := hint.UpdatePatientIfDiffersFromEntity(externalPatientID, syncConfig, entity); err != nil {
			return errors.Errorf("unable to update patient '%s' in hint: %s", externalPatientID, err)
		}
	default:
		return errors.Errorf("unsupported source %s", source)
	}

	return nil
}
