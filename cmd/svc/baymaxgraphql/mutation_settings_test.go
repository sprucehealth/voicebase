package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
)

func TestModifySetting_Boolean(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "account_12345",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	key := "2fa"
	nodeID := "entity_e1"
	g.settingsC.Expect(mock.NewExpectation(g.settingsC.GetConfigs, &settings.GetConfigsRequest{
		Keys: []string{key},
	}).WithReturns(&settings.GetConfigsResponse{
		Configs: []*settings.Config{
			{
				Title:          "Hello",
				Description:    "Hi",
				Key:            key,
				AllowSubkeys:   false,
				Type:           settings.ConfigType_BOOLEAN,
				PossibleOwners: []settings.OwnerType{settings.OwnerType_INTERNAL_ENTITY},
				Config: &settings.Config_Boolean{
					Boolean: &settings.BooleanConfig{
						Default: &settings.BooleanValue{
							Value: false,
						},
					},
				},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: nodeID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_INTERNAL,
			ID:   nodeID,
			Info: &directory.EntityInfo{
				DisplayName: "HI",
			},
		},
	}, nil))

	g.settingsC.Expect(mock.NewExpectation(g.settingsC.SetValue, &settings.SetValueRequest{
		NodeID: nodeID,
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key: key,
			},
			Type: settings.ConfigType_BOOLEAN,
			Value: &settings.Value_Boolean{
				Boolean: &settings.BooleanValue{
					Value: true,
				},
			},
		},
	}).WithReturns(&settings.SetValueResponse{
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key: key,
			},
			Type: settings.ConfigType_BOOLEAN,
			Value: &settings.Value_Boolean{
				Boolean: &settings.BooleanValue{
					Value: true,
				},
			},
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ ($nodeID: ID!, $key: String!) {
			modifySetting(input: {
				clientMutationId: "a1b2c3",
				nodeID: $nodeID,
				key: $key,
				booleanValue: {
					set: true
				}
			}) {
				clientMutationId
				success
				setting {
					key
					subkey
					title
					description
					value {
						__typename
						... on BooleanSettingValue {
							set
						}
					}
				}
			}
		}`, map[string]interface{}{
		"nodeID": nodeID,
		"key":    key,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"modifySetting": {
			"clientMutationId": "a1b2c3",
			"setting": {
				"description": "Hi",
				"key": "2fa",
				"subkey": null,
				"title": "Hello",
				"value": {
					"__typename": "BooleanSettingValue",
					"set": true
				}
			},
			"success": true
		}
	}
}`, string(b))
}

func TestModifySetting_StringList(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "account_12345",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	key := excommsSettings.ConfigKeyForwardingList
	nodeID := "entity_e1"
	g.settingsC.Expect(mock.NewExpectation(g.settingsC.GetConfigs, &settings.GetConfigsRequest{
		Keys: []string{key},
	}).WithReturns(&settings.GetConfigsResponse{
		Configs: []*settings.Config{
			{
				Title:          "Hello",
				Description:    "Hi",
				Key:            excommsSettings.ConfigKeyForwardingList,
				AllowSubkeys:   true,
				Type:           settings.ConfigType_STRING_LIST,
				PossibleOwners: []settings.OwnerType{settings.OwnerType_INTERNAL_ENTITY},
				Config: &settings.Config_StringList{
					StringList: &settings.StringListConfig{},
				},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: nodeID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_INTERNAL,
			ID:   nodeID,
			Info: &directory.EntityInfo{
				DisplayName: "HI",
			},
		},
	}, nil))

	g.settingsC.Expect(mock.NewExpectation(g.settingsC.SetValue, &settings.SetValueRequest{
		NodeID: nodeID,
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key:    key,
				Subkey: "+17348465522",
			},
			Type: settings.ConfigType_STRING_LIST,
			Value: &settings.Value_StringList{
				StringList: &settings.StringListValue{
					Values: []string{" 734 846-5522", "(206) 8773590", "1234565522"},
				},
			},
		},
	}).WithReturns(&settings.SetValueResponse{
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key:    key,
				Subkey: "+17348465522",
			},
			Type: settings.ConfigType_STRING_LIST,
			Value: &settings.Value_StringList{
				StringList: &settings.StringListValue{
					Values:        []string{" 734 8465522", "(206) 8773590", "1234565522"},
					DisplayValues: []string{"(734) 846-5522", "(206) 877-3590", "(123) 456-5522"},
				},
			},
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ ($nodeID: ID!, $key: String!, $subkey: String!) {
			modifySetting(input: {
				clientMutationId: "a1b2c3",
				nodeID: $nodeID,
				key: $key,
				subkey: $subkey,
				stringListValue: {
					list: [" 734 846-5522", "(206) 8773590", "1234565522"]
				}
			}) {
				clientMutationId
				success
				setting {
					key
					subkey
					title
					description
					value {
						__typename
						... on StringListSettingValue {
							list
						}
					}
				}
			}
		}`, map[string]interface{}{
		"nodeID": nodeID,
		"key":    key,
		"subkey": "(734) 846-5522",
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"modifySetting": {
			"clientMutationId": "a1b2c3",
			"setting": {
				"description": "Hi",
				"key": "forwarding_list",
				"subkey": "+17348465522",
				"title": "Hello",
				"value": {
					"__typename": "StringListSettingValue",
					"list": [
						"(734) 846-5522",
						"(206) 877-3590",
						"(123) 456-5522"
					]
				}
			},
			"success": true
		}
	}
}`, string(b))
}

func TestModifySetting_StringList_InvalidInput(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "account_12345",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	key := excommsSettings.ConfigKeyForwardingList
	nodeID := "entity_e1"
	g.settingsC.Expect(mock.NewExpectation(g.settingsC.GetConfigs, &settings.GetConfigsRequest{
		Keys: []string{key},
	}).WithReturns(&settings.GetConfigsResponse{
		Configs: []*settings.Config{
			{
				Title:          "Hello",
				Description:    "Hi",
				Key:            excommsSettings.ConfigKeyForwardingList,
				AllowSubkeys:   true,
				Type:           settings.ConfigType_STRING_LIST,
				PossibleOwners: []settings.OwnerType{settings.OwnerType_INTERNAL_ENTITY},
				Config: &settings.Config_StringList{
					StringList: &settings.StringListConfig{},
				},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: nodeID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_INTERNAL,
			ID:   nodeID,
			Info: &directory.EntityInfo{
				DisplayName: "HI",
			},
		},
	}, nil))

	g.settingsC.Expect(mock.NewExpectation(g.settingsC.SetValue, &settings.SetValueRequest{
		NodeID: nodeID,
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key:    key,
				Subkey: "+17348465522",
			},
			Type: settings.ConfigType_STRING_LIST,
			Value: &settings.Value_StringList{
				StringList: &settings.StringListValue{
					Values: []string{" 734"},
				},
			},
		},
	}).WithReturns(&settings.SetValueResponse{}, grpc.Errorf(settings.InvalidUserValue, "Invalid US phone number")))

	res := g.query(ctx, `
		mutation _ ($nodeID: ID!, $key: String!, $subkey: String!) {
			modifySetting(input: {
				clientMutationId: "a1b2c3",
				nodeID: $nodeID,
				key: $key,
				subkey: $subkey,
				stringListValue: {
					list: [" 734"]
				}
			}) {
				clientMutationId
				success
				errorCode
				errorMessage
				setting {
					key
					subkey
					title
					description
					value {
						__typename
						... on StringListSettingValue {
							list
						}
					}
				}
			}
		}`, map[string]interface{}{
		"nodeID": nodeID,
		"key":    key,
		"subkey": "(734) 846-5522",
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"modifySetting": {
			"clientMutationId": "a1b2c3",
			"errorCode": "INVALID_INPUT",
			"errorMessage": "Invalid US phone number",
			"setting": null,
			"success": false
		}
	}
}`, string(b))
}

func TestModifySetting_MultiSelect(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "account_12345",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	key := "2fa"
	nodeID := "entity_e1"
	g.settingsC.Expect(mock.NewExpectation(g.settingsC.GetConfigs, &settings.GetConfigsRequest{
		Keys: []string{key},
	}).WithReturns(&settings.GetConfigsResponse{
		Configs: []*settings.Config{
			{
				Title:          "Hello",
				Description:    "Hi",
				Key:            key,
				AllowSubkeys:   false,
				Type:           settings.ConfigType_MULTI_SELECT,
				PossibleOwners: []settings.OwnerType{settings.OwnerType_INTERNAL_ENTITY},
				Config: &settings.Config_MultiSelect{
					MultiSelect: &settings.MultiSelectConfig{
						Items: []*settings.Item{
							{
								ID:    "option1",
								Label: "option1",
							},
							{
								ID:    "option2",
								Label: "option2",
							},
							{
								ID:    "option3",
								Label: "option3",
							},
						},
					},
				},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: nodeID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_INTERNAL,
			ID:   nodeID,
			Info: &directory.EntityInfo{
				DisplayName: "HI",
			},
		},
	}, nil))

	g.settingsC.Expect(mock.NewExpectation(g.settingsC.SetValue, &settings.SetValueRequest{
		NodeID: nodeID,
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key: key,
			},
			Type: settings.ConfigType_MULTI_SELECT,
			Value: &settings.Value_MultiSelect{
				MultiSelect: &settings.MultiSelectValue{
					Items: []*settings.ItemValue{
						{
							ID: "option1",
						},
						{
							ID: "option2",
						},
					},
				},
			},
		},
	}).WithReturns(&settings.SetValueResponse{
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key: key,
			},
			Type: settings.ConfigType_MULTI_SELECT,
			Value: &settings.Value_MultiSelect{
				MultiSelect: &settings.MultiSelectValue{
					Items: []*settings.ItemValue{
						{
							ID: "option1",
						},
						{
							ID: "option2",
						},
					},
				},
			},
		}}, nil))

	res := g.query(ctx, `
		mutation _ ($nodeID: ID!, $key: String!) {
			modifySetting(input: {
				clientMutationId: "a1b2c3",
				nodeID: $nodeID,
				key: $key,
				selectValue: {
					items: [{
							id: "option1"
						},{
							id: "option2"
						}
					]
				}
			}) {
				clientMutationId
				success
				setting {
					key
					subkey
					title
					description
					... on SelectSetting {
						allowsMultipleSelection
					}
					value {
						__typename
						... on SelectableSettingValue {
							items {
								id
							}
						}
					}
				}
			}
		}`, map[string]interface{}{
		"nodeID": nodeID,
		"key":    key,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"modifySetting": {
			"clientMutationId": "a1b2c3",
			"setting": {
				"allowsMultipleSelection": true,
				"description": "Hi",
				"key": "2fa",
				"subkey": null,
				"title": "Hello",
				"value": {
					"__typename": "SelectableSettingValue",
					"items": [
						{
							"id": "option1"
						},
						{
							"id": "option2"
						}
					]
				}
			},
			"success": true
		}
	}
}`, string(b))
}

func TestModifySetting_SingleSelect(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "account_12345",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	key := "2fa"
	nodeID := "entity_e1"
	g.settingsC.Expect(mock.NewExpectation(g.settingsC.GetConfigs, &settings.GetConfigsRequest{
		Keys: []string{key},
	}).WithReturns(&settings.GetConfigsResponse{
		Configs: []*settings.Config{
			{
				Title:          "Hello",
				Description:    "Hi",
				Key:            key,
				AllowSubkeys:   false,
				Type:           settings.ConfigType_SINGLE_SELECT,
				PossibleOwners: []settings.OwnerType{settings.OwnerType_INTERNAL_ENTITY},
				Config: &settings.Config_SingleSelect{
					SingleSelect: &settings.SingleSelectConfig{
						Items: []*settings.Item{
							{
								ID:    "option1",
								Label: "option1",
							},
							{
								ID:    "option2",
								Label: "option2",
							},
							{
								ID:    "option3",
								Label: "option3",
							},
						},
					},
				},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: nodeID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_INTERNAL,
			ID:   nodeID,
			Info: &directory.EntityInfo{
				DisplayName: "HI",
			},
		},
	}, nil))

	g.settingsC.Expect(mock.NewExpectation(g.settingsC.SetValue, &settings.SetValueRequest{
		NodeID: nodeID,
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key: key,
			},
			Type: settings.ConfigType_SINGLE_SELECT,
			Value: &settings.Value_SingleSelect{
				SingleSelect: &settings.SingleSelectValue{
					Item: &settings.ItemValue{
						ID: "option1",
					},
				},
			},
		},
	}).WithReturns(&settings.SetValueResponse{
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key: key,
			},
			Type: settings.ConfigType_SINGLE_SELECT,
			Value: &settings.Value_SingleSelect{
				SingleSelect: &settings.SingleSelectValue{
					Item: &settings.ItemValue{
						ID: "option1",
					},
				},
			},
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ ($nodeID: ID!, $key: String!) {
			modifySetting(input: {
				clientMutationId: "a1b2c3",
				nodeID: $nodeID,
				key: $key,
				selectValue: {
					items: [{
							id: "option1"
						}
					]
				}
			}) {
				clientMutationId
				success
				setting {
					key
					subkey
					title
					description
					... on SelectSetting {
						allowsMultipleSelection
					}
					value {
						__typename
						... on SelectableSettingValue {
							items {
								id
							}
						}
					}
				}
			}
		}`, map[string]interface{}{
		"nodeID": nodeID,
		"key":    key,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"modifySetting": {
			"clientMutationId": "a1b2c3",
			"setting": {
				"allowsMultipleSelection": false,
				"description": "Hi",
				"key": "2fa",
				"subkey": null,
				"title": "Hello",
				"value": {
					"__typename": "SelectableSettingValue",
					"items": [
						{
							"id": "option1"
						}
					]
				}
			},
			"success": true
		}
	}
}`, string(b))
}

func TestModifySetting_InvalidOwner(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "account_12345",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	key := "2fa"
	nodeID := "entity_e1"
	g.settingsC.Expect(mock.NewExpectation(g.settingsC.GetConfigs, &settings.GetConfigsRequest{
		Keys: []string{key},
	}).WithReturns(&settings.GetConfigsResponse{
		Configs: []*settings.Config{
			{
				Title:          "Hello",
				Description:    "Hi",
				Key:            key,
				AllowSubkeys:   false,
				Type:           settings.ConfigType_BOOLEAN,
				PossibleOwners: []settings.OwnerType{settings.OwnerType_ACCOUNT},
				Config: &settings.Config_Boolean{
					Boolean: &settings.BooleanConfig{
						Default: &settings.BooleanValue{
							Value: false,
						},
					},
				},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: nodeID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_INTERNAL,
			ID:   nodeID,
			Info: &directory.EntityInfo{
				DisplayName: "HI",
			},
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ ($nodeID: ID!, $key: String!) {
			modifySetting(input: {
				clientMutationId: "a1b2c3",
				nodeID: $nodeID,
				key: $key,
				booleanValue: {
					set: true
				}
			}) {
				clientMutationId
				success
				setting {
					key
					subkey
					title
					description
					value {
						__typename
						... on BooleanSettingValue {
							set
						}
					}
				}
			}
		}`, map[string]interface{}{
		"nodeID": nodeID,
		"key":    key,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, true, strings.Contains(string(b), "cannot modify"))
}

func TestModifySetting_PatientBackwardsNotifications(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "account_12345",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	key := notification.PatientNotificationPreferencesSettingsKey
	nodeID := "entity_e1"
	g.settingsC.Expect(mock.NewExpectation(g.settingsC.GetConfigs, &settings.GetConfigsRequest{
		Keys: []string{key},
	}).WithReturns(&settings.GetConfigsResponse{
		Configs: []*settings.Config{
			{
				Title:          "Hello",
				Description:    "Hi",
				Key:            key,
				AllowSubkeys:   false,
				Type:           settings.ConfigType_SINGLE_SELECT,
				PossibleOwners: []settings.OwnerType{settings.OwnerType_INTERNAL_ENTITY},
				Config: &settings.Config_SingleSelect{
					SingleSelect: &settings.SingleSelectConfig{
						Items: []*settings.Item{
							{
								ID:    notification.ThreadActivityNotificationPreferenceAllMessages,
								Label: notification.ThreadActivityNotificationPreferenceAllMessages,
							},
							{
								ID:    notification.ThreadActivityNotificationPreferenceReferencedOnly,
								Label: notification.ThreadActivityNotificationPreferenceReferencedOnly,
							},
							{
								ID:    notification.ThreadActivityNotificationPreferenceOff,
								Label: notification.ThreadActivityNotificationPreferenceOff,
							},
						},
					},
				},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: nodeID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_INTERNAL,
			ID:   nodeID,
			Info: &directory.EntityInfo{
				DisplayName: "HI",
			},
		},
	}, nil))

	g.settingsC.Expect(mock.NewExpectation(g.settingsC.SetValue, &settings.SetValueRequest{
		NodeID: nodeID,
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key: key,
			},
			Type: settings.ConfigType_SINGLE_SELECT,
			Value: &settings.Value_SingleSelect{
				SingleSelect: &settings.SingleSelectValue{
					Item: &settings.ItemValue{
						ID: notification.ThreadActivityNotificationPreferenceOff,
					},
				},
			},
		},
	}).WithReturns(&settings.SetValueResponse{
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key: key,
			},
			Type: settings.ConfigType_SINGLE_SELECT,
			Value: &settings.Value_SingleSelect{
				SingleSelect: &settings.SingleSelectValue{
					Item: &settings.ItemValue{
						ID: notification.ThreadActivityNotificationPreferenceOff,
					},
				},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.SavedQueries, nodeID).WithReturns([]*threading.SavedQuery{
		{
			ID:         "patientSQID",
			ShortTitle: "patient",
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.UpdateSavedQuery, &threading.UpdateSavedQueryRequest{
		SavedQueryID:         "patientSQID",
		NotificationsEnabled: threading.BOOL_FALSE,
	}))

	res := g.query(ctx, `
		mutation _ ($nodeID: ID!, $key: String!) {
			modifySetting(input: {
				clientMutationId: "a1b2c3",
				nodeID: $nodeID,
				key: $key,
				selectValue: {
					items: [{
							id: "`+notification.ThreadActivityNotificationPreferenceOff+`"
						}
					]
				}
			}) {
				clientMutationId
				success
				setting {
					key
					subkey
					title
					description
					... on SelectSetting {
						allowsMultipleSelection
					}
					value {
						__typename
						... on SelectableSettingValue {
							items {
								id
							}
						}
					}
				}
			}
		}`, map[string]interface{}{
		"nodeID": nodeID,
		"key":    key,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"modifySetting": {
			"clientMutationId": "a1b2c3",
			"setting": {
				"allowsMultipleSelection": false,
				"description": "Hi",
				"key": "`+key+`",
				"subkey": null,
				"title": "Hello",
				"value": {
					"__typename": "SelectableSettingValue",
					"items": [
						{
							"id": "`+notification.ThreadActivityNotificationPreferenceOff+`"
						}
					]
				}
			},
			"success": true
		}
	}
}`, string(b))
}

func TestModifySetting_TeamBackwardsNotifications(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "account_12345",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	key := notification.TeamNotificationPreferencesSettingsKey
	nodeID := "entity_e1"
	g.settingsC.Expect(mock.NewExpectation(g.settingsC.GetConfigs, &settings.GetConfigsRequest{
		Keys: []string{key},
	}).WithReturns(&settings.GetConfigsResponse{
		Configs: []*settings.Config{
			{
				Title:          "Hello",
				Description:    "Hi",
				Key:            key,
				AllowSubkeys:   false,
				Type:           settings.ConfigType_SINGLE_SELECT,
				PossibleOwners: []settings.OwnerType{settings.OwnerType_INTERNAL_ENTITY},
				Config: &settings.Config_SingleSelect{
					SingleSelect: &settings.SingleSelectConfig{
						Items: []*settings.Item{
							{
								ID:    notification.ThreadActivityNotificationPreferenceAllMessages,
								Label: notification.ThreadActivityNotificationPreferenceAllMessages,
							},
							{
								ID:    notification.ThreadActivityNotificationPreferenceReferencedOnly,
								Label: notification.ThreadActivityNotificationPreferenceReferencedOnly,
							},
							{
								ID:    notification.ThreadActivityNotificationPreferenceOff,
								Label: notification.ThreadActivityNotificationPreferenceOff,
							},
						},
					},
				},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: nodeID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_INTERNAL,
			ID:   nodeID,
			Info: &directory.EntityInfo{
				DisplayName: "HI",
			},
		},
	}, nil))

	g.settingsC.Expect(mock.NewExpectation(g.settingsC.SetValue, &settings.SetValueRequest{
		NodeID: nodeID,
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key: key,
			},
			Type: settings.ConfigType_SINGLE_SELECT,
			Value: &settings.Value_SingleSelect{
				SingleSelect: &settings.SingleSelectValue{
					Item: &settings.ItemValue{
						ID: notification.ThreadActivityNotificationPreferenceOff,
					},
				},
			},
		},
	}).WithReturns(&settings.SetValueResponse{
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key: key,
			},
			Type: settings.ConfigType_SINGLE_SELECT,
			Value: &settings.Value_SingleSelect{
				SingleSelect: &settings.SingleSelectValue{
					Item: &settings.ItemValue{
						ID: notification.ThreadActivityNotificationPreferenceOff,
					},
				},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.SavedQueries, nodeID).WithReturns([]*threading.SavedQuery{
		{
			ID:         "teamSQID",
			ShortTitle: "team",
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.UpdateSavedQuery, &threading.UpdateSavedQueryRequest{
		SavedQueryID:         "teamSQID",
		NotificationsEnabled: threading.BOOL_FALSE,
	}))

	res := g.query(ctx, `
		mutation _ ($nodeID: ID!, $key: String!) {
			modifySetting(input: {
				clientMutationId: "a1b2c3",
				nodeID: $nodeID,
				key: $key,
				selectValue: {
					items: [{
							id: "`+notification.ThreadActivityNotificationPreferenceOff+`"
						}
					]
				}
			}) {
				clientMutationId
				success
				setting {
					key
					subkey
					title
					description
					... on SelectSetting {
						allowsMultipleSelection
					}
					value {
						__typename
						... on SelectableSettingValue {
							items {
								id
							}
						}
					}
				}
			}
		}`, map[string]interface{}{
		"nodeID": nodeID,
		"key":    key,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"modifySetting": {
			"clientMutationId": "a1b2c3",
			"setting": {
				"allowsMultipleSelection": false,
				"description": "Hi",
				"key": "`+key+`",
				"subkey": null,
				"title": "Hello",
				"value": {
					"__typename": "SelectableSettingValue",
					"items": [
						{
							"id": "`+notification.ThreadActivityNotificationPreferenceOff+`"
						}
					]
				}
			},
			"success": true
		}
	}
}`, string(b))
}
