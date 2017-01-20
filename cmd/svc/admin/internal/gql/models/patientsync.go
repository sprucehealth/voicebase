package models

import "github.com/sprucehealth/backend/svc/patientsync"

type TagMappingItem struct {
	Tag        string `json:"tag"`
	ProviderID string `json:"providerID"`
}
type PatientSyncConfiguration struct {
	Source      string            `json:"source"`
	Connected   bool              `json:"connected"`
	ThreadType  string            `json:"threadType"`
	TagMappings []*TagMappingItem `json:"tagMappings"`
}

func TransformPatientSyncConfigurationToModel(config *patientsync.Config) *PatientSyncConfiguration {

	mappings := make([]*TagMappingItem, len(config.TagMappings))
	for i, item := range config.TagMappings {
		mappings[i] = &TagMappingItem{
			Tag:        item.Tag,
			ProviderID: "UNKNOWN",
		}
		switch t := item.Key.(type) {
		case *patientsync.TagMappingItem_ProviderID:
			mappings[i].ProviderID = t.ProviderID
		}
	}

	return &PatientSyncConfiguration{
		Connected:   config.Connected,
		Source:      config.Source.String(),
		ThreadType:  config.ThreadCreationType.String(),
		TagMappings: mappings,
	}
}
