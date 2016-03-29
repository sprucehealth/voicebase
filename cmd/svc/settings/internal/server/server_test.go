package server

import (
	dalmock "github.com/sprucehealth/backend/cmd/svc/settings/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/settings/internal/models"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"testing"
)

func TestRegisterConfig_MultiSelect(t *testing.T) {
	md := dalmock.New(t)
	md.Expect(mock.NewExpectation(md.SetConfigs, []*models.Config{
		{
			Title:          "hello",
			Description:    "hi",
			Key:            "testingkey",
			Type:           models.ConfigType_MULTI_SELECT,
			AllowSubkeys:   true,
			PossibleOwners: []models.OwnerType{},
			Config: &models.Config_MultiSelect{
				MultiSelect: &models.MultiSelectConfig{
					Items: []*models.Item{
						{
							ID:    "option1",
							Label: "Option 1",
						},
						{
							ID:    "option2",
							Label: "Option 2",
						},
					},
					Default: &models.MultiSelectValue{
						Items: []*models.ItemValue{
							{
								ID: "option1",
							},
						},
					},
				},
			},
		},
	}))

	server := New(md)
	_, err := server.RegisterConfigs(context.Background(), &settings.RegisterConfigsRequest{
		Configs: []*settings.Config{
			{
				Title:          "hello",
				Description:    "hi",
				Key:            "testingkey",
				Type:           settings.ConfigType_MULTI_SELECT,
				AllowSubkeys:   true,
				PossibleOwners: []settings.OwnerType{},
				Config: &settings.Config_MultiSelect{
					MultiSelect: &settings.MultiSelectConfig{
						Items: []*settings.Item{
							{
								ID:    "option1",
								Label: "Option 1",
							},
							{
								ID:    "option2",
								Label: "Option 2",
							},
						},
						Default: &settings.MultiSelectValue{
							Items: []*settings.ItemValue{
								{
									ID: "option1",
								},
							},
						},
					},
				},
			},
		},
	})
	test.OK(t, err)

	mock.FinishAll(md)

}

func TestRegisterConfig_SingleSelect(t *testing.T) {
	md := dalmock.New(t)
	md.Expect(mock.NewExpectation(md.SetConfigs, []*models.Config{
		{
			Title:          "hello",
			Description:    "hi",
			Key:            "testingkey",
			Type:           models.ConfigType_SINGLE_SELECT,
			AllowSubkeys:   true,
			PossibleOwners: []models.OwnerType{},
			Config: &models.Config_SingleSelect{
				SingleSelect: &models.SingleSelectConfig{
					Items: []*models.Item{
						{
							ID:    "option1",
							Label: "Option 1",
						},
						{
							ID:    "option2",
							Label: "Option 2",
						},
					},
					Default: &models.SingleSelectValue{
						Item: &models.ItemValue{
							ID: "option1",
						},
					},
				},
			},
		},
	}))

	server := New(md)
	_, err := server.RegisterConfigs(context.Background(), &settings.RegisterConfigsRequest{
		Configs: []*settings.Config{
			{
				Title:          "hello",
				Description:    "hi",
				Key:            "testingkey",
				Type:           settings.ConfigType_SINGLE_SELECT,
				AllowSubkeys:   true,
				PossibleOwners: []settings.OwnerType{},
				Config: &settings.Config_SingleSelect{
					SingleSelect: &settings.SingleSelectConfig{
						Items: []*settings.Item{
							{
								ID:    "option1",
								Label: "Option 1",
							},
							{
								ID:    "option2",
								Label: "Option 2",
							},
						},
						Default: &settings.SingleSelectValue{
							Item: &settings.ItemValue{
								ID: "option1",
							},
						},
					},
				},
			},
		},
	})
	test.OK(t, err)

	mock.FinishAll(md)
}

func TestRegisterConfig_Boolean(t *testing.T) {
	md := dalmock.New(t)
	md.Expect(mock.NewExpectation(md.SetConfigs, []*models.Config{
		{
			Title:          "hello",
			Description:    "hi",
			Key:            "testingkey",
			Type:           models.ConfigType_BOOLEAN,
			PossibleOwners: []models.OwnerType{},
			AllowSubkeys:   true,
			Config: &models.Config_Boolean{
				Boolean: &models.BooleanConfig{
					Default: &models.BooleanValue{
						Value: true,
					},
				},
			},
		},
	}))

	server := New(md)
	_, err := server.RegisterConfigs(context.Background(), &settings.RegisterConfigsRequest{
		Configs: []*settings.Config{
			{
				Title:          "hello",
				Description:    "hi",
				Key:            "testingkey",
				Type:           settings.ConfigType_BOOLEAN,
				AllowSubkeys:   true,
				PossibleOwners: []settings.OwnerType{},
				Config: &settings.Config_Boolean{
					Boolean: &settings.BooleanConfig{
						Default: &settings.BooleanValue{
							Value: true,
						},
					},
				},
			},
		},
	})
	test.OK(t, err)

	mock.FinishAll(md)
}

func TestRegisterConfig_StringList(t *testing.T) {
	md := dalmock.New(t)
	md.Expect(mock.NewExpectation(md.SetConfigs, []*models.Config{
		{
			Title:          "hello",
			Description:    "hi",
			Key:            "testingkey",
			Type:           models.ConfigType_STRING_LIST,
			AllowSubkeys:   true,
			PossibleOwners: []models.OwnerType{},
			Config: &models.Config_StringList{
				StringList: &models.StringListConfig{
					Default: &models.StringListValue{
						Values: []string{"test1", "test2"},
					},
				},
			},
		},
	}))

	server := New(md)
	_, err := server.RegisterConfigs(context.Background(), &settings.RegisterConfigsRequest{
		Configs: []*settings.Config{
			{
				Title:          "hello",
				Description:    "hi",
				Key:            "testingkey",
				PossibleOwners: []settings.OwnerType{},
				Type:           settings.ConfigType_STRING_LIST,
				AllowSubkeys:   true,
				Config: &settings.Config_StringList{
					StringList: &settings.StringListConfig{
						Default: &settings.StringListValue{
							Values: []string{"test1", "test2"},
						},
					},
				},
			},
		},
	})
	test.OK(t, err)

	mock.FinishAll(md)
}

func TestSetValue_MultiSelect(t *testing.T) {
	md := dalmock.New(t)
	md.Expect(mock.NewExpectation(md.GetConfigs, []string{"testingkey"}).WithReturns(
		[]*models.Config{
			{
				Title:        "hello",
				Description:  "hi",
				Key:          "testingkey",
				Type:         models.ConfigType_MULTI_SELECT,
				AllowSubkeys: true,
				Config: &models.Config_MultiSelect{
					MultiSelect: &models.MultiSelectConfig{
						Items: []*models.Item{
							{
								ID:    "option1",
								Label: "Option 1",
							},
							{
								ID:    "option2",
								Label: "Option 2",
							},
						},
						Default: &models.MultiSelectValue{
							Items: []*models.ItemValue{
								{
									ID: "option1",
								},
							},
						},
					},
				},
			},
		}, nil))

	md.Expect(mock.NewExpectation(md.SetValues, "12345", []*models.Value{
		{
			Key: &models.ConfigKey{
				Key:    "testingkey",
				Subkey: "22222",
			},
			Value: &models.Value_MultiSelect{
				MultiSelect: &models.MultiSelectValue{
					Items: []*models.ItemValue{
						{
							ID: "option1",
						},
					},
				},
			},
			Config: &models.Config{
				Title:        "hello",
				Description:  "hi",
				Key:          "testingkey",
				Type:         models.ConfigType_MULTI_SELECT,
				AllowSubkeys: true,
				Config: &models.Config_MultiSelect{
					MultiSelect: &models.MultiSelectConfig{
						Items: []*models.Item{
							{
								ID:    "option1",
								Label: "Option 1",
							},
							{
								ID:    "option2",
								Label: "Option 2",
							},
						},
						Default: &models.MultiSelectValue{
							Items: []*models.ItemValue{
								{
									ID: "option1",
								},
							},
						},
					},
				},
			},
		},
	}))

	server := New(md)
	_, err := server.SetValue(context.Background(), &settings.SetValueRequest{
		NodeID: "12345",
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key:    "testingkey",
				Subkey: "22222",
			},
			Type: settings.ConfigType_MULTI_SELECT,
			Value: &settings.Value_MultiSelect{
				MultiSelect: &settings.MultiSelectValue{
					Items: []*settings.ItemValue{
						{
							ID: "option1",
						},
					},
				},
			},
		},
	})
	test.OK(t, err)
	mock.FinishAll(md)
}

func TestSetValue_MultiSelect_Invalid_RequiredFreeText(t *testing.T) {
	md := dalmock.New(t)
	md.Expect(mock.NewExpectation(md.GetConfigs, []string{"testingkey"}).WithReturns(
		[]*models.Config{
			{
				Title:        "hello",
				Description:  "hi",
				Key:          "testingkey",
				Type:         models.ConfigType_MULTI_SELECT,
				AllowSubkeys: true,
				Config: &models.Config_MultiSelect{
					MultiSelect: &models.MultiSelectConfig{
						Items: []*models.Item{
							{
								ID:               "option1",
								Label:            "Option 1",
								AllowFreeText:    true,
								FreeTextRequired: true,
							},
							{
								ID:    "option2",
								Label: "Option 2",
							},
						},
						Default: &models.MultiSelectValue{
							Items: []*models.ItemValue{
								{
									ID: "option1",
								},
							},
						},
					},
				},
			},
		}, nil))

	server := New(md)
	_, err := server.SetValue(context.Background(), &settings.SetValueRequest{
		NodeID: "12345",
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key:    "testingkey",
				Subkey: "22222",
			},
			Type: settings.ConfigType_MULTI_SELECT,
			Value: &settings.Value_MultiSelect{
				MultiSelect: &settings.MultiSelectValue{
					Items: []*settings.ItemValue{
						{
							ID: "option1",
						},
					},
				},
			},
		},
	})
	test.Assert(t, err != nil, "expected validation error")
	mock.FinishAll(md)
}

func TestSetValue_SingleSelect_Invalid_RequiredFreeText(t *testing.T) {
	md := dalmock.New(t)
	md.Expect(mock.NewExpectation(md.GetConfigs, []string{"testingkey"}).WithReturns(
		[]*models.Config{
			{
				Title:        "hello",
				Description:  "hi",
				Key:          "testingkey",
				Type:         models.ConfigType_SINGLE_SELECT,
				AllowSubkeys: true,
				Config: &models.Config_SingleSelect{
					SingleSelect: &models.SingleSelectConfig{
						Items: []*models.Item{
							{
								ID:               "option1",
								Label:            "Option 1",
								AllowFreeText:    true,
								FreeTextRequired: true,
							},
							{
								ID:    "option2",
								Label: "Option 2",
							},
						},
						Default: &models.SingleSelectValue{
							Item: &models.ItemValue{
								ID: "option1",
							},
						},
					},
				},
			},
		}, nil))

	server := New(md)
	_, err := server.SetValue(context.Background(), &settings.SetValueRequest{
		NodeID: "12345",
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key:    "testingkey",
				Subkey: "22222",
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
	})
	test.Assert(t, err != nil, "expected validation error")
	mock.FinishAll(md)
}

func TestSetValue_MultiSelect_Invalid(t *testing.T) {
	md := dalmock.New(t)
	md.Expect(mock.NewExpectation(md.GetConfigs, []string{"testingkey"}).WithReturns(
		[]*models.Config{
			{
				Title:        "hello",
				Description:  "hi",
				Key:          "testingkey",
				Type:         models.ConfigType_MULTI_SELECT,
				AllowSubkeys: true,
				Config: &models.Config_MultiSelect{
					MultiSelect: &models.MultiSelectConfig{
						Items: []*models.Item{
							{
								ID:    "option1",
								Label: "Option 1",
							},
							{
								ID:    "option2",
								Label: "Option 2",
							},
						},
						Default: &models.MultiSelectValue{
							Items: []*models.ItemValue{
								{
									ID: "option1",
								},
							},
						},
					},
				},
			},
		}, nil))

	server := New(md)
	_, err := server.SetValue(context.Background(), &settings.SetValueRequest{
		NodeID: "12345",
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key:    "testingkey",
				Subkey: "22222",
			},
			Type: settings.ConfigType_MULTI_SELECT,
			Value: &settings.Value_MultiSelect{
				MultiSelect: &settings.MultiSelectValue{
					Items: []*settings.ItemValue{
						{
							ID: "option1124",
						},
					},
				},
			},
		},
	})
	test.Equals(t, true, err != nil)
	test.Equals(t, codes.InvalidArgument, grpc.Code(err))
	mock.FinishAll(md)
}

func TestGetValues_Default(t *testing.T) {
	md := dalmock.New(t)
	md.Expect(mock.NewExpectation(md.GetValues, "12345", []*models.ConfigKey{
		{
			Key:    "test",
			Subkey: "22222",
		},
	}))
	md.Expect(mock.NewExpectation(md.GetConfigs, []string{"test"}).WithReturns(
		[]*models.Config{
			{
				Title:        "hello",
				Description:  "hi",
				Key:          "test",
				Type:         models.ConfigType_BOOLEAN,
				AllowSubkeys: true,
				Config: &models.Config_Boolean{
					Boolean: &models.BooleanConfig{
						Default: &models.BooleanValue{
							Value: true,
						},
					},
				},
			},
		}, nil))

	server := New(md)
	res, err := server.GetValues(context.Background(), &settings.GetValuesRequest{
		NodeID: "12345",
		Keys: []*settings.ConfigKey{
			{
				Key:    "test",
				Subkey: "22222",
			},
		},
	})
	test.OK(t, err)
	test.Equals(t, 1, len(res.Values))
	test.Equals(t, &settings.Value{
		Type: settings.ConfigType_BOOLEAN,
		Key: &settings.ConfigKey{
			Key:    "test",
			Subkey: "22222",
		},
		Value: &settings.Value_Boolean{
			Boolean: &settings.BooleanValue{
				Value: true,
			},
		},
	}, res.Values[0])
	mock.FinishAll(md)
}

func TestGetValues_MultiSelect(t *testing.T) {
	md := dalmock.New(t)
	md.Expect(mock.NewExpectation(md.GetValues, "12345", []*models.ConfigKey{
		{
			Key:    "test",
			Subkey: "22222",
		},
	}).WithReturns(
		[]*models.Value{
			{
				Key: &models.ConfigKey{
					Key:    "test",
					Subkey: "22222",
				},
				Config: &models.Config{
					Title:        "hello",
					Description:  "hi",
					Key:          "test",
					Type:         models.ConfigType_MULTI_SELECT,
					AllowSubkeys: true,
					Config: &models.Config_MultiSelect{
						MultiSelect: &models.MultiSelectConfig{
							Items: []*models.Item{
								{
									ID:    "option1",
									Label: "Option 1",
								},
								{
									ID:    "option2",
									Label: "Option 2",
								},
							},
							Default: &models.MultiSelectValue{
								Items: []*models.ItemValue{
									{
										ID: "option1",
									},
								},
							},
						},
					},
				},
				Value: &models.Value_MultiSelect{
					MultiSelect: &models.MultiSelectValue{
						Items: []*models.ItemValue{
							{
								ID: "option1",
							},
						},
					},
				},
			},
		}, nil))

	server := New(md)
	res, err := server.GetValues(context.Background(), &settings.GetValuesRequest{
		NodeID: "12345",
		Keys: []*settings.ConfigKey{
			{
				Key:    "test",
				Subkey: "22222",
			},
		},
	})
	test.OK(t, err)
	test.Equals(t, 1, len(res.Values))
	test.Equals(t, &settings.Value{
		Type: settings.ConfigType_MULTI_SELECT,
		Key: &settings.ConfigKey{
			Key:    "test",
			Subkey: "22222",
		},
		Value: &settings.Value_MultiSelect{
			MultiSelect: &settings.MultiSelectValue{
				Items: []*settings.ItemValue{
					{
						ID: "option1",
					},
				},
			},
		},
	}, res.Values[0])
	mock.FinishAll(md)
}
