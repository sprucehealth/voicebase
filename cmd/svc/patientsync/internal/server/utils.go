package server

import (
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	psettings "github.com/sprucehealth/backend/cmd/svc/patientsync/settings"
	"github.com/sprucehealth/backend/svc/patientsync"
)

func transformThreadType(threadType string) sync.Config_ThreadCreationType {
	switch threadType {
	case psettings.ThreadTypeOptionSecure:
		return sync.THREAD_CREATION_TYPE_SECURE
	case psettings.ThreadTypeOptionStandard:
		return sync.THREAD_CREATION_TYPE_STANDARD
	}
	return sync.THREAD_CREATION_TYPE_UKNOWN
}

func transformSyncConfigurationToResponse(syncConfig *sync.Config, syncBookmark *dal.SyncBookmark) *patientsync.Config {
	var threadType patientsync.ThreadCreationType
	switch syncConfig.ThreadCreationType {
	case sync.THREAD_CREATION_TYPE_SECURE:
		threadType = patientsync.THREAD_CREATION_TYPE_SECURE
	case sync.THREAD_CREATION_TYPE_STANDARD:
		threadType = patientsync.THREAD_CREATION_TYPE_STANDARD

	}

	mappings := make([]*patientsync.TagMappingItem, len(syncConfig.TagMappings))
	for i, item := range syncConfig.TagMappings {
		mappings[i] = &patientsync.TagMappingItem{
			Tag: item.Tag,
		}
		switch t := item.Key.(type) {
		case *sync.TagMappingItem_ProviderID:
			mappings[i].Key = &patientsync.TagMappingItem_ProviderID{
				ProviderID: t.ProviderID,
			}
		}
	}

	var source patientsync.Source
	switch syncConfig.Source {
	case sync.SOURCE_HINT:
		source = patientsync.SOURCE_HINT
	case sync.SOURCE_DRCHRONO:
		source = patientsync.SOURCE_DRCHRONO
	case sync.SOURCE_ELATION:
		source = patientsync.SOURCE_ELATION
	case sync.SOURCE_CSV:
		source = patientsync.SOURCE_CSV
	default:
		source = patientsync.SOURCE_UNKNOWN
	}

	return &patientsync.Config{
		ThreadCreationType: threadType,
		Source:             source,
		Connected:          syncBookmark != nil && syncBookmark.Status == dal.SyncStatusConnected,
		TagMappings:        mappings,
	}
}
