package server

import (
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	psettings "github.com/sprucehealth/backend/cmd/svc/patientsync/settings"
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
