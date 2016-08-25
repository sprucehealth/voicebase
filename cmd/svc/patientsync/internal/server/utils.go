package server

import (
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	"github.com/sprucehealth/backend/svc/patientsync"
)

func transformThreadType(threadType patientsync.ThreadType) sync.Config_ThreadCreationType {
	switch threadType {
	case patientsync.THREAD_TYPE_SECURE:
		return sync.THREAD_CREATION_TYPE_SECURE
	case patientsync.THREAD_TYPE_STANDARD:
		return sync.THREAD_CREATION_TYPE_STANDARD
	}
	return sync.THREAD_CREATION_TYPE_UKNOWN
}
