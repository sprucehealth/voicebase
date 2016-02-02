package main

import (
	"encoding/json"
	"strings"
	"testing"

	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func TestModifySetting_Boolean(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &account{
		ID: "account_12345",
	}
	ctx = ctxWithAccount(ctx, acc)

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

	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntities,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: nodeID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
				},
			},
		},
	).WithReturns(
		&directory.LookupEntitiesResponse{
			Entities: []*directory.Entity{
				{
					Type: directory.EntityType_INTERNAL,
					ID:   nodeID,
					Info: &directory.EntityInfo{
						DisplayName: "HI",
					},
				},
			},
		},
		nil))

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
	}).WithReturns(&settings.SetValueResponse{}, nil))

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
				result				
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
			"result": "SUCCESS",
			"setting": {
				"description": "Hi",
				"key": "2fa",
				"subkey": null,
				"title": "Hello",
				"value": {
					"__typename": "BooleanSettingValue",
					"set": true
				}
			}
		}
	}
}`, string(b))
}

func TestModifySetting_StringList(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &account{
		ID: "account_12345",
	}
	ctx = ctxWithAccount(ctx, acc)

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

	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntities,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: nodeID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
				},
			},
		},
	).WithReturns(
		&directory.LookupEntitiesResponse{
			Entities: []*directory.Entity{
				{
					Type: directory.EntityType_INTERNAL,
					ID:   nodeID,
					Info: &directory.EntityInfo{
						DisplayName: "HI",
					},
				},
			},
		},
		nil))

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
					Values: []string{"(734) 846-5522", "(206) 877-3590", "(123) 456-5522"},
				},
			},
		},
	}).WithReturns(&settings.SetValueResponse{}, nil))

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
				result				
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
			"result": "SUCCESS",
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
			}
		}
	}
}`, string(b))
}

func TestModifySetting_StringList_InvalidInput(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &account{
		ID: "account_12345",
	}
	ctx = ctxWithAccount(ctx, acc)

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

	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntities,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: nodeID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
				},
			},
		},
	).WithReturns(
		&directory.LookupEntitiesResponse{
			Entities: []*directory.Entity{
				{
					Type: directory.EntityType_INTERNAL,
					ID:   nodeID,
					Info: &directory.EntityInfo{
						DisplayName: "HI",
					},
				},
			},
		},
		nil))

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
				result
				userErrorMessage				
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
			"result": "INVALID_INPUT",
			"setting": null,
			"userErrorMessage": "Please enter a valid US phone number"
		}
	}
}`, string(b))
}

func TestModifySetting_MultiSelect(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &account{
		ID: "account_12345",
	}
	ctx = ctxWithAccount(ctx, acc)

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

	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntities,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: nodeID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
				},
			},
		},
	).WithReturns(
		&directory.LookupEntitiesResponse{
			Entities: []*directory.Entity{
				{
					Type: directory.EntityType_INTERNAL,
					ID:   nodeID,
					Info: &directory.EntityInfo{
						DisplayName: "HI",
					},
				},
			},
		},
		nil))

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
	}).WithReturns(&settings.SetValueResponse{}, nil))

	res := g.query(ctx, `
		mutation _ ($nodeID: ID!, $key: String!) {
			modifySetting(input: {
				clientMutationId: "a1b2c3",
				nodeID: $nodeID,
				key: $key,
				multiSelectValue: {
					items: [{
							id: "option1"
						},{
							id: "option2"
						}
					]
				}
			}) {
				clientMutationId
				result				
				setting {
					key
					subkey
					title
					description
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
			"result": "SUCCESS",
			"setting": {
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
			}
		}
	}
}`, string(b))
}

func TestModifySetting_SingleSelect(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &account{
		ID: "account_12345",
	}
	ctx = ctxWithAccount(ctx, acc)

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

	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntities,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: nodeID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
				},
			},
		},
	).WithReturns(
		&directory.LookupEntitiesResponse{
			Entities: []*directory.Entity{
				{
					Type: directory.EntityType_INTERNAL,
					ID:   nodeID,
					Info: &directory.EntityInfo{
						DisplayName: "HI",
					},
				},
			},
		},
		nil))

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
	}).WithReturns(&settings.SetValueResponse{}, nil))

	res := g.query(ctx, `
		mutation _ ($nodeID: ID!, $key: String!) {
			modifySetting(input: {
				clientMutationId: "a1b2c3",
				nodeID: $nodeID,
				key: $key,
				singleSelectValue: {
					items: [{
							id: "option1"
						}
					]
				}
			}) {
				clientMutationId
				result				
				setting {
					key
					subkey
					title
					description
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
			"result": "SUCCESS",
			"setting": {
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
			}
		}
	}
}`, string(b))
}

func TestModifySetting_InvalidOwner(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &account{
		ID: "account_12345",
	}
	ctx = ctxWithAccount(ctx, acc)

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

	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntities,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: nodeID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
				},
			},
		},
	).WithReturns(
		&directory.LookupEntitiesResponse{
			Entities: []*directory.Entity{
				{
					Type: directory.EntityType_INTERNAL,
					ID:   nodeID,
					Info: &directory.EntityInfo{
						DisplayName: "HI",
					},
				},
			},
		},
		nil))

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
				result
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
