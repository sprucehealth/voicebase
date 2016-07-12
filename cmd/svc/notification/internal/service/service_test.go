package service

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"context"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/sprucehealth/backend/cmd/svc/notification/internal/dal"
	testdal "github.com/sprucehealth/backend/cmd/svc/notification/internal/dal/test"
	nsettings "github.com/sprucehealth/backend/cmd/svc/notification/internal/settings"
	"github.com/sprucehealth/backend/libs/caremessenger/deeplink"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	dmock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/settings"
	smock "github.com/sprucehealth/backend/svc/settings/mock"
)

func init() {
	conc.Testing = true
}

func TestExternalIDToAccountIDTransformation(t *testing.T) {
	externalIDs := []*directory.ExternalID{
		{
			ID: auth.AccountIDPrefix + "215610700746457088",
		},
		{
			ID: auth.AccountIDPrefix + "215610700746457090",
		},
		{
			ID: "other_1235123423522",
		},
	}
	accountIDs := accountIDsFromExternalIDs(externalIDs)
	test.Equals(t, []string{auth.AccountIDPrefix + "215610700746457088", auth.AccountIDPrefix + "215610700746457090"}, accountIDs)
}

const (
	deviceRegistrationSQSURL        = "deviceRegistrationSQSURL"
	notificationSQSURL              = "notificationSQSURL"
	appleDeviceRegistrationSNSARN   = "appleDeviceRegistrationSNSARN"
	andriodDeviceRegistrationSNSARN = "andriodDeviceRegistrationSNSARN"
	receiptHandle                   = "receiptHandle"
)

func TestProcessNewDeviceRegistrationIOS(t *testing.T) {
	dl := testdal.NewMockDAL(t)
	dc := dmock.New(t)
	snsAPI := mock.NewSNSAPI(t)
	sqsAPI := mock.NewSQSAPI(t)
	sc := smock.New(t)
	defer mock.FinishAll(dl, dc, snsAPI, sqsAPI, sc)
	svc := New(dl, dc, sc, &Config{
		DeviceRegistrationSQSURL:        deviceRegistrationSQSURL,
		AppleDeviceRegistrationSNSARN:   appleDeviceRegistrationSNSARN,
		AndriodDeviceRegistrationSNSARN: andriodDeviceRegistrationSNSARN,
		SQSAPI: sqsAPI,
		SNSAPI: snsAPI,
	})
	cSvc := svc.(*service)

	driData, err := json.Marshal(&notification.DeviceRegistrationInfo{
		ExternalGroupID: "ExternalGroupID",
		DeviceToken:     "DeviceToken",
		Platform:        "iOS",
		PlatformVersion: "PlatformVersion",
		AppVersion:      "AppVersion",
		Device:          "Device",
		DeviceModel:     "DeviceModel",
		DeviceID:        "DeviceID",
	})
	test.OK(t, err)

	// Lookup the device and don't find it
	dl.Expect(mock.NewExpectation(dl.PushConfigForDeviceToken, "DeviceToken").WithReturns((*dal.PushConfig)(nil), dal.ErrNotFound))

	// Generate an endpoint for the device
	snsAPI.Expect(mock.NewExpectation(snsAPI.CreatePlatformEndpoint, &sns.CreatePlatformEndpointInput{
		PlatformApplicationArn: ptr.String(appleDeviceRegistrationSNSARN),
		Token: ptr.String("DeviceToken"),
	}).WithReturns(&sns.CreatePlatformEndpointOutput{
		EndpointArn: ptr.String("iOSEnpointARN"),
	}, nil))

	// Insert a new record for the device
	dl.Expect(mock.NewExpectation(dl.InsertPushConfig, &dal.PushConfig{
		ExternalGroupID: "ExternalGroupID",
		Platform:        "iOS",
		PlatformVersion: "PlatformVersion",
		AppVersion:      "AppVersion",
		DeviceID:        "DeviceID",
		DeviceToken:     []byte("DeviceToken"),
		PushEndpoint:    "iOSEnpointARN",
		Device:          "Device",
		DeviceModel:     "DeviceModel",
	}))

	cSvc.processDeviceRegistration(context.Background(), string(driData))
}

func TestProcessNewDeviceRegistrationAndroid(t *testing.T) {
	dl := testdal.NewMockDAL(t)
	dc := dmock.New(t)
	snsAPI := mock.NewSNSAPI(t)
	sqsAPI := mock.NewSQSAPI(t)
	sc := smock.New(t)
	defer mock.FinishAll(dl, dc, snsAPI, sqsAPI, sc)
	svc := New(dl, dc, sc, &Config{
		DeviceRegistrationSQSURL:        deviceRegistrationSQSURL,
		AppleDeviceRegistrationSNSARN:   appleDeviceRegistrationSNSARN,
		AndriodDeviceRegistrationSNSARN: andriodDeviceRegistrationSNSARN,
		SQSAPI: sqsAPI,
		SNSAPI: snsAPI,
	})
	cSvc := svc.(*service)

	driData, err := json.Marshal(&notification.DeviceRegistrationInfo{
		ExternalGroupID: "ExternalGroupID",
		DeviceToken:     "DeviceToken",
		Platform:        "android",
		PlatformVersion: "PlatformVersion",
		AppVersion:      "AppVersion",
		Device:          "Device",
		DeviceModel:     "DeviceModel",
		DeviceID:        "DeviceID",
	})
	test.OK(t, err)

	// Lookup the device and don't find it
	dl.Expect(mock.NewExpectation(dl.PushConfigForDeviceToken, "DeviceToken").WithReturns((*dal.PushConfig)(nil), dal.ErrNotFound))

	// Generate an endpoint for the device
	snsAPI.Expect(mock.NewExpectation(snsAPI.CreatePlatformEndpoint, &sns.CreatePlatformEndpointInput{
		PlatformApplicationArn: ptr.String(andriodDeviceRegistrationSNSARN),
		Token: ptr.String("DeviceToken"),
	}).WithReturns(&sns.CreatePlatformEndpointOutput{
		EndpointArn: ptr.String("androidEnpointARN"),
	}, nil))

	// Insert a new record for the device
	dl.Expect(mock.NewExpectation(dl.InsertPushConfig, &dal.PushConfig{
		ExternalGroupID: "ExternalGroupID",
		Platform:        "android",
		PlatformVersion: "PlatformVersion",
		AppVersion:      "AppVersion",
		DeviceID:        "DeviceID",
		DeviceToken:     []byte("DeviceToken"),
		PushEndpoint:    "androidEnpointARN",
		Device:          "Device",
		DeviceModel:     "DeviceModel",
	}))

	cSvc.processDeviceRegistration(context.Background(), string(driData))
}

func TestProcessExistingDeviceRegistrationAndroid(t *testing.T) {
	dl := testdal.NewMockDAL(t)
	dc := dmock.New(t)
	snsAPI := mock.NewSNSAPI(t)
	sqsAPI := mock.NewSQSAPI(t)
	sc := smock.New(t)
	defer mock.FinishAll(dl, dc, snsAPI, sqsAPI, sc)
	svc := New(dl, dc, sc, &Config{
		DeviceRegistrationSQSURL:        deviceRegistrationSQSURL,
		AppleDeviceRegistrationSNSARN:   appleDeviceRegistrationSNSARN,
		AndriodDeviceRegistrationSNSARN: andriodDeviceRegistrationSNSARN,
		SQSAPI: sqsAPI,
		SNSAPI: snsAPI,
	})
	cSvc := svc.(*service)

	driData, err := json.Marshal(&notification.DeviceRegistrationInfo{
		ExternalGroupID: "ExternalGroupID",
		DeviceToken:     "DeviceToken",
		Platform:        "android",
		PlatformVersion: "PlatformVersion",
		AppVersion:      "AppVersion",
		Device:          "Device",
		DeviceModel:     "DeviceModel",
		DeviceID:        "DeviceID",
	})
	test.OK(t, err)

	// Lookup the device and find it
	dl.Expect(mock.NewExpectation(dl.PushConfigForDeviceToken, "DeviceToken").WithReturns(&dal.PushConfig{
		ID: dal.PushConfigID{
			ObjectID: model.ObjectID{
				Prefix:  notification.PushConfigIDPrefix,
				Val:     1,
				IsValid: true,
			},
		},
		DeviceToken: []byte("DeviceToken"),
	}, nil))

	// Insert a new record for the device
	dl.Expect(mock.NewExpectation(dl.UpdatePushConfig, dal.PushConfigID{
		ObjectID: model.ObjectID{
			Prefix:  notification.PushConfigIDPrefix,
			Val:     1,
			IsValid: true,
		},
	}, &dal.PushConfigUpdate{
		DeviceID:        ptr.String("DeviceID"),
		ExternalGroupID: ptr.String("ExternalGroupID"),
		Platform:        ptr.String("android"),
		PlatformVersion: ptr.String("PlatformVersion"),
		AppVersion:      ptr.String("AppVersion"),
		DeviceToken:     []byte("DeviceToken"),
	}))

	cSvc.processDeviceRegistration(context.Background(), string(driData))
}

func TestProcessExistingDeviceDeregistration(t *testing.T) {
	dl := testdal.NewMockDAL(t)
	dc := dmock.New(t)
	snsAPI := mock.NewSNSAPI(t)
	sqsAPI := mock.NewSQSAPI(t)
	sc := smock.New(t)
	defer mock.FinishAll(dl, dc, snsAPI, sqsAPI, sc)
	svc := New(dl, dc, sc, &Config{
		DeviceRegistrationSQSURL:        deviceRegistrationSQSURL,
		AppleDeviceRegistrationSNSARN:   appleDeviceRegistrationSNSARN,
		AndriodDeviceRegistrationSNSARN: andriodDeviceRegistrationSNSARN,
		SQSAPI: sqsAPI,
		SNSAPI: snsAPI,
	})
	cSvc := svc.(*service)

	ddriData, err := json.Marshal(&notification.DeviceDeregistrationInfo{
		DeviceID: "DeviceID",
	})
	test.OK(t, err)

	// Lookup the device and find it
	dl.Expect(mock.NewExpectation(dl.DeletePushConfigForDeviceID, "DeviceID"))

	cSvc.processDeviceDeregistration(context.Background(), string(ddriData))
}

func expectFilterNodesWithNotificationsDisabled(t *testing.T, sc *smock.Client, nodes []string, values []bool) {
	test.Assert(t, len(nodes) == len(values), "Expected the number of nodes and values to be equal for mocking")
	for i, n := range nodes {
		sc.Expect(mock.NewExpectation(sc.GetValues, &settings.GetValuesRequest{
			Keys:   []*settings.ConfigKey{{Key: notification.ReceiveNotificationsSettingsKey}},
			NodeID: n,
		}).WithReturns(&settings.GetValuesResponse{
			Values: []*settings.Value{
				{
					Type:  settings.ConfigType_BOOLEAN,
					Value: &settings.Value_Boolean{Boolean: &settings.BooleanValue{Value: values[i]}},
				},
			},
		}, nil))
	}
}

func expectFilterNodesForThreadActivityPreferences(t *testing.T, sc *smock.Client, key string, nodes, ids []string) {
	test.Assert(t, len(nodes) == len(ids), "Expected the number of nodes and values to be equal for mocking")
	for i, n := range nodes {
		sc.Expect(mock.NewExpectation(sc.GetValues, &settings.GetValuesRequest{
			Keys:   []*settings.ConfigKey{{Key: key}},
			NodeID: n,
		}).WithReturns(&settings.GetValuesResponse{
			Values: []*settings.Value{
				{
					Type:  settings.ConfigType_SINGLE_SELECT,
					Value: &settings.Value_SingleSelect{SingleSelect: &settings.SingleSelectValue{Item: &settings.ItemValue{ID: ids[i]}}},
				},
			},
		}, nil))
	}
}

func TestProcessNotification(t *testing.T) {
	dl := testdal.NewMockDAL(t)
	dc := dmock.New(t)
	snsAPI := mock.NewSNSAPI(t)
	sqsAPI := mock.NewSQSAPI(t)
	sc := smock.New(t)
	defer mock.FinishAll(dl, dc, snsAPI, sqsAPI, sc)
	svc := New(dl, dc, sc, &Config{
		NotificationSQSURL:              notificationSQSURL,
		AppleDeviceRegistrationSNSARN:   appleDeviceRegistrationSNSARN,
		AndriodDeviceRegistrationSNSARN: andriodDeviceRegistrationSNSARN,
		SQSAPI:    sqsAPI,
		SNSAPI:    snsAPI,
		WebDomain: "testDomain",
	})
	cSvc := svc.(*service)

	entitiesToNotify := []string{"entity:1", "entity:2", "entity:4"}
	notificationData, err := json.Marshal(&notification.Notification{
		ShortMessages: map[string]string{
			"entity:1": "",
			"entity:2": "ShortMessage2",
			"entity:4": "ShortMessage4",
		},
		UnreadCounts: map[string]int{
			"entity:1": 1,
			"entity:2": 2,
			"entity:4": 4,
		},
		CollapseKey:      "collapse",
		DedupeKey:        "dedupe",
		ThreadID:         "ThreadID",
		OrganizationID:   "OrganizationID",
		MessageID:        "ItemID",
		SavedQueryID:     "SavedQueryID",
		EntitiesToNotify: entitiesToNotify,
	})
	test.OK(t, err)

	// Check the settings for each entity
	expectFilterNodesWithNotificationsDisabled(t, sc, []string{"entity:1", "entity:2", "entity:4"}, []bool{true, true, true})

	// Lookup account IDs for the entities via their external identifiers, we should have filtered 1
	dc.Expect(mock.NewExpectation(dc.ExternalIDs, &directory.ExternalIDsRequest{
		EntityIDs: []string{"entity:1", "entity:2", "entity:4"},
	}).WithReturns(&directory.ExternalIDsResponse{
		ExternalIDs: []*directory.ExternalID{
			{ID: "account_1", EntityID: "entity:1"},
			{ID: "account_2", EntityID: "entity:2"},
			{ID: "account_4", EntityID: "entity:4"},
		},
	}, nil))

	// Lookup the push configs for each external group id (account)
	dl.Expect(mock.NewExpectation(dl.PushConfigsForExternalGroupID, "account_1").WithReturns([]*dal.PushConfig{
		{PushEndpoint: "account1:pushEndpoint1", Platform: "iOS"},
		{PushEndpoint: "account1:pushEndpoint2", Platform: "android"},
	}, nil))

	// Build out expected notification structures
	iData, err := json.Marshal(&iOSPushNotification{
		PushData: &iOSPushData{
			ContentAvailable: 1,
		},
		URL:            deeplink.ThreadMessageURLShareable("testDomain", "OrganizationID", "ThreadID", "ItemID"),
		ThreadID:       "ThreadID",
		OrganizationID: "OrganizationID",
		MessageID:      "ItemID",
		SavedQueryID:   "SavedQueryID",
	})
	test.OK(t, err)
	aData, err := json.Marshal(&androidPushNotification{
		CollapseKey: "collapse",
		Priority:    "normal",
		PushData: &androidPushData{
			Message:        "",
			Background:     true,
			URL:            deeplink.ThreadMessageURLShareable("testDomain", "OrganizationID", "ThreadID", "ItemID"),
			ThreadID:       "ThreadID",
			OrganizationID: "OrganizationID",
			MessageID:      "ItemID",
			SavedQueryID:   "SavedQueryID",
			PushID:         "dedupe",
		},
	})
	test.OK(t, err)
	snsNote := &snsNotification{
		DefaultMessage: "",
		IOSSandBox:     string(iData),
		IOS:            string(iData),
		Android:        string(aData),
	}
	msg, err := json.Marshal(snsNote)
	test.OK(t, err)

	// Send out the push notifications
	snsAPI.Expect(mock.NewExpectation(snsAPI.Publish, &sns.PublishInput{
		Message:          ptr.String(string(msg)),
		MessageStructure: jsonStructure,
		TargetArn:        ptr.String("account1:pushEndpoint1"),
	}))
	snsAPI.Expect(mock.NewExpectation(snsAPI.Publish, &sns.PublishInput{
		Message:          ptr.String(string(msg)),
		MessageStructure: jsonStructure,
		TargetArn:        ptr.String("account1:pushEndpoint2"),
	}))

	// Repeat for the next thread member
	dl.Expect(mock.NewExpectation(dl.PushConfigsForExternalGroupID, "account_2").WithReturns([]*dal.PushConfig{
		{PushEndpoint: "account2:pushEndpoint1", Platform: "iOS"},
		{PushEndpoint: "account2:pushEndpoint2", Platform: "android"},
		{PushEndpoint: "account2:pushEndpoint2", Platform: "unknown"},
	}, nil))

	// Build out expected notification structures
	iData, err = json.Marshal(&iOSPushNotification{
		PushData: &iOSPushData{
			Alert:            "ShortMessage2",
			Sound:            "default",
			ContentAvailable: 1,
		},
		URL:            deeplink.ThreadMessageURLShareable("testDomain", "OrganizationID", "ThreadID", "ItemID"),
		ThreadID:       "ThreadID",
		OrganizationID: "OrganizationID",
		MessageID:      "ItemID",
		SavedQueryID:   "SavedQueryID",
	})
	test.OK(t, err)
	aData, err = json.Marshal(&androidPushNotification{
		CollapseKey: "collapse",
		Priority:    "normal",
		PushData: &androidPushData{
			Message:        "ShortMessage2",
			Background:     false,
			URL:            deeplink.ThreadMessageURLShareable("testDomain", "OrganizationID", "ThreadID", "ItemID"),
			ThreadID:       "ThreadID",
			OrganizationID: "OrganizationID",
			MessageID:      "ItemID",
			SavedQueryID:   "SavedQueryID",
			PushID:         "dedupe",
		},
	})
	test.OK(t, err)
	snsNote = &snsNotification{
		DefaultMessage: "ShortMessage2",
		IOSSandBox:     string(iData),
		IOS:            string(iData),
		Android:        string(aData),
	}
	msg, err = json.Marshal(snsNote)
	test.OK(t, err)

	// Send out the push notifications
	snsAPI.Expect(mock.NewExpectation(snsAPI.Publish, &sns.PublishInput{
		Message:          ptr.String(string(msg)),
		MessageStructure: jsonStructure,
		TargetArn:        ptr.String("account2:pushEndpoint1"),
	}))
	snsAPI.Expect(mock.NewExpectation(snsAPI.Publish, &sns.PublishInput{
		Message:          ptr.String(string(msg)),
		MessageStructure: jsonStructure,
		TargetArn:        ptr.String("account2:pushEndpoint2"),
	}))

	// Repeat for the next thread member
	dl.Expect(mock.NewExpectation(dl.PushConfigsForExternalGroupID, "account_4").WithReturns([]*dal.PushConfig{
		{PushEndpoint: "account4:pushEndpoint1", Platform: "iOS"},
	}, nil))

	// Build out expected notification structures
	iData, err = json.Marshal(&iOSPushNotification{
		PushData: &iOSPushData{
			Alert:            "ShortMessage4",
			Sound:            "default",
			ContentAvailable: 1,
		},
		URL:            deeplink.ThreadMessageURLShareable("testDomain", "OrganizationID", "ThreadID", "ItemID"),
		ThreadID:       "ThreadID",
		OrganizationID: "OrganizationID",
		MessageID:      "ItemID",
		SavedQueryID:   "SavedQueryID",
	})
	test.OK(t, err)
	aData, err = json.Marshal(&androidPushNotification{
		CollapseKey: "collapse",
		Priority:    "normal",
		PushData: &androidPushData{
			Message:        "ShortMessage4",
			Background:     false,
			URL:            deeplink.ThreadMessageURLShareable("testDomain", "OrganizationID", "ThreadID", "ItemID"),
			ThreadID:       "ThreadID",
			OrganizationID: "OrganizationID",
			MessageID:      "ItemID",
			SavedQueryID:   "SavedQueryID",
			PushID:         "dedupe",
		},
	})
	test.OK(t, err)
	snsNote = &snsNotification{
		DefaultMessage: "ShortMessage4",
		IOSSandBox:     string(iData),
		IOS:            string(iData),
		Android:        string(aData),
	}
	msg, err = json.Marshal(snsNote)
	test.OK(t, err)

	// Send out the push notifications
	snsAPI.Expect(mock.NewExpectation(snsAPI.Publish, &sns.PublishInput{
		Message:          ptr.String(string(msg)),
		MessageStructure: jsonStructure,
		TargetArn:        ptr.String("account4:pushEndpoint1"),
	}))

	cSvc.processNotification(context.Background(), string(notificationData))
}

func TestProcessNotificationDisabledEndpoint(t *testing.T) {
	dl := testdal.NewMockDAL(t)
	dc := dmock.New(t)
	snsAPI := mock.NewSNSAPI(t)
	sqsAPI := mock.NewSQSAPI(t)
	pcID, err := dal.NewPushConfigID()
	test.OK(t, err)
	sc := smock.New(t)
	defer mock.FinishAll(dl, dc, snsAPI, sqsAPI, sc)
	svc := New(dl, dc, sc, &Config{
		NotificationSQSURL:              notificationSQSURL,
		AppleDeviceRegistrationSNSARN:   appleDeviceRegistrationSNSARN,
		AndriodDeviceRegistrationSNSARN: andriodDeviceRegistrationSNSARN,
		SQSAPI:    sqsAPI,
		SNSAPI:    snsAPI,
		WebDomain: "testDomain",
	})
	cSvc := svc.(*service)

	notificationData, err := json.Marshal(&notification.Notification{
		ShortMessages: map[string]string{
			"entity:1": "ShortMessage",
		},
		UnreadCounts: map[string]int{
			"entity:1": 1,
		},
		ThreadID:         "ThreadID",
		OrganizationID:   "OrganizationID",
		MessageID:        "ItemID",
		SavedQueryID:     "SavedQueryID",
		EntitiesToNotify: []string{"entity:1", "entity:2"},
		CollapseKey:      "collapse",
		DedupeKey:        "dedupe",
	})
	test.OK(t, err)

	expectFilterNodesWithNotificationsDisabled(t, sc, []string{"entity:1", "entity:2"}, []bool{true, false})

	// Lookup account IDs for the entities via their external identifiers
	dc.Expect(mock.NewExpectation(dc.ExternalIDs, &directory.ExternalIDsRequest{
		EntityIDs: []string{"entity:1"},
	}).WithReturns(&directory.ExternalIDsResponse{
		ExternalIDs: []*directory.ExternalID{
			{ID: "account_1", EntityID: "entity:1"},
		},
	}, nil))

	// Lookup the push configs for each external group id (account)
	dl.Expect(mock.NewExpectation(dl.PushConfigsForExternalGroupID, "account_1").WithReturns([]*dal.PushConfig{
		{ID: pcID, PushEndpoint: "account1:pushEndpoint1", Platform: "iOS"},
		{PushEndpoint: "account1:pushEndpoint2", Platform: "android"},
	}, nil))

	// Build out expected notification structure
	iData, err := json.Marshal(&iOSPushNotification{
		PushData: &iOSPushData{
			Sound:            "default",
			Alert:            "ShortMessage",
			ContentAvailable: 1,
		},
		URL:            deeplink.ThreadMessageURLShareable("testDomain", "OrganizationID", "ThreadID", "ItemID"),
		ThreadID:       "ThreadID",
		OrganizationID: "OrganizationID",
		MessageID:      "ItemID",
		SavedQueryID:   "SavedQueryID",
	})
	test.OK(t, err)
	aData, err := json.Marshal(&androidPushNotification{
		CollapseKey: "collapse",
		Priority:    "normal",
		PushData: &androidPushData{
			Message:        "ShortMessage",
			URL:            deeplink.ThreadMessageURLShareable("testDomain", "OrganizationID", "ThreadID", "ItemID"),
			ThreadID:       "ThreadID",
			OrganizationID: "OrganizationID",
			MessageID:      "ItemID",
			SavedQueryID:   "SavedQueryID",
			PushID:         "dedupe",
		},
	})
	test.OK(t, err)
	snsNote := &snsNotification{
		DefaultMessage: "ShortMessage",
		IOSSandBox:     string(iData),
		IOS:            string(iData),
		Android:        string(aData),
	}
	msg, err := json.Marshal(snsNote)
	test.OK(t, err)

	// Send out the push notifications
	snsAPI.Expect(mock.NewExpectation(snsAPI.Publish, &sns.PublishInput{
		Message:          ptr.String(string(msg)),
		MessageStructure: jsonStructure,
		TargetArn:        ptr.String("account1:pushEndpoint1"),
	}).WithReturns((*sns.PublishOutput)(nil), awserr.New(endpointDisabledAWSErrCode, "So disabled", errors.New(":("))))

	dl.Expect(mock.NewExpectation(dl.DeletePushConfig, pcID))

	snsAPI.Expect(mock.NewExpectation(snsAPI.Publish, &sns.PublishInput{
		Message:          ptr.String(string(msg)),
		MessageStructure: jsonStructure,
		TargetArn:        ptr.String("account1:pushEndpoint2"),
	}))

	cSvc.processNotification(context.Background(), string(notificationData))
}

func TestProcessNotificationInternalMessage(t *testing.T) {
	dl := testdal.NewMockDAL(t)
	dc := dmock.New(t)
	snsAPI := mock.NewSNSAPI(t)
	sqsAPI := mock.NewSQSAPI(t)
	sc := smock.New(t)
	defer mock.FinishAll(dl, dc, snsAPI, sqsAPI, sc)
	svc := New(dl, dc, sc, &Config{
		NotificationSQSURL:              notificationSQSURL,
		AppleDeviceRegistrationSNSARN:   appleDeviceRegistrationSNSARN,
		AndriodDeviceRegistrationSNSARN: andriodDeviceRegistrationSNSARN,
		SQSAPI:    sqsAPI,
		SNSAPI:    snsAPI,
		WebDomain: "testDomain",
	})
	cSvc := svc.(*service)

	entitiesToNotify := []string{"entity:1", "entity:2", "entity:3", "entity:4"}
	notificationData, err := json.Marshal(&notification.Notification{
		ShortMessages: map[string]string{
			"entity:1": "",
			"entity:2": "ShortMessage2",
			"entity:3": "ShortMessage3",
			"entity:4": "ShortMessage4",
		},
		UnreadCounts: map[string]int{
			"entity:1": 1,
			"entity:2": 2,
			"entity:3": 3,
			"entity:4": 4,
		},
		CollapseKey:          "collapse",
		DedupeKey:            "dedupe",
		ThreadID:             "ThreadID",
		OrganizationID:       "OrganizationID",
		MessageID:            "ItemID",
		SavedQueryID:         "SavedQueryID",
		EntitiesToNotify:     entitiesToNotify,
		EntitiesAtReferenced: map[string]struct{}{"entity:2": {}, "entity:3": {}},
		Type:                 notification.NewMessageOnInternalThread,
	})
	test.OK(t, err)

	// Check the settings for each account
	expectFilterNodesWithNotificationsDisabled(t, sc, []string{"entity:1", "entity:2", "entity:3", "entity:4"}, []bool{true, true, false, false})

	// Check the setting for the entities
	expectFilterNodesForThreadActivityPreferences(t, sc, notification.TeamNotificationPreferencesSettingsKey, []string{"entity:1", "entity:2"}, []string{
		nsettings.ThreadActivityNotificationPreferenceAllMessages,
		nsettings.ThreadActivityNotificationPreferenceReferencedOnly,
	})

	// Lookup account IDs for the entities via their external identifiers, we should have filtered 1
	dc.Expect(mock.NewExpectation(dc.ExternalIDs, &directory.ExternalIDsRequest{
		EntityIDs: []string{"entity:1", "entity:2"},
	}).WithReturns(&directory.ExternalIDsResponse{
		ExternalIDs: []*directory.ExternalID{
			{ID: "account_1", EntityID: "entity:1"},
			{ID: "account_2", EntityID: "entity:2"},
		},
	}, nil))

	// Lookup the push configs for each external group id (account)
	dl.Expect(mock.NewExpectation(dl.PushConfigsForExternalGroupID, "account_1").WithReturns([]*dal.PushConfig{
		{PushEndpoint: "account1:pushEndpoint1", Platform: "iOS"},
		{PushEndpoint: "account1:pushEndpoint2", Platform: "android"},
	}, nil))

	// Build out expected notification structures
	iData, err := json.Marshal(&iOSPushNotification{
		PushData: &iOSPushData{
			ContentAvailable: 1,
		},
		Type:           string(notification.NewMessageOnInternalThread),
		URL:            deeplink.ThreadMessageURLShareable("testDomain", "OrganizationID", "ThreadID", "ItemID"),
		ThreadID:       "ThreadID",
		OrganizationID: "OrganizationID",
		MessageID:      "ItemID",
		SavedQueryID:   "SavedQueryID",
	})
	test.OK(t, err)
	aData, err := json.Marshal(&androidPushNotification{
		CollapseKey: "collapse",
		Priority:    "normal",
		PushData: &androidPushData{
			Message:        "",
			Background:     true,
			Type:           string(notification.NewMessageOnInternalThread),
			URL:            deeplink.ThreadMessageURLShareable("testDomain", "OrganizationID", "ThreadID", "ItemID"),
			ThreadID:       "ThreadID",
			OrganizationID: "OrganizationID",
			MessageID:      "ItemID",
			SavedQueryID:   "SavedQueryID",
			PushID:         "dedupe",
		},
	})
	test.OK(t, err)
	snsNote := &snsNotification{
		DefaultMessage: "",
		IOSSandBox:     string(iData),
		IOS:            string(iData),
		Android:        string(aData),
	}
	msg, err := json.Marshal(snsNote)
	test.OK(t, err)

	// Send out the push notifications
	snsAPI.Expect(mock.NewExpectation(snsAPI.Publish, &sns.PublishInput{
		Message:          ptr.String(string(msg)),
		MessageStructure: jsonStructure,
		TargetArn:        ptr.String("account1:pushEndpoint1"),
	}))
	snsAPI.Expect(mock.NewExpectation(snsAPI.Publish, &sns.PublishInput{
		Message:          ptr.String(string(msg)),
		MessageStructure: jsonStructure,
		TargetArn:        ptr.String("account1:pushEndpoint2"),
	}))

	// Repeat for the next thread member
	dl.Expect(mock.NewExpectation(dl.PushConfigsForExternalGroupID, "account_2").WithReturns([]*dal.PushConfig{
		{PushEndpoint: "account2:pushEndpoint1", Platform: "iOS"},
		{PushEndpoint: "account2:pushEndpoint2", Platform: "android"},
	}, nil))

	// Build out expected notification structures
	iData, err = json.Marshal(&iOSPushNotification{
		PushData: &iOSPushData{
			Alert:            "ShortMessage2",
			Sound:            "default",
			ContentAvailable: 1,
		},
		Type:           string(notification.NewMessageOnInternalThread),
		URL:            deeplink.ThreadMessageURLShareable("testDomain", "OrganizationID", "ThreadID", "ItemID"),
		ThreadID:       "ThreadID",
		OrganizationID: "OrganizationID",
		MessageID:      "ItemID",
		SavedQueryID:   "SavedQueryID",
	})
	test.OK(t, err)
	aData, err = json.Marshal(&androidPushNotification{
		CollapseKey: "collapse",
		Priority:    "normal",
		PushData: &androidPushData{
			Type:           string(notification.NewMessageOnInternalThread),
			Message:        "ShortMessage2",
			Background:     false,
			URL:            deeplink.ThreadMessageURLShareable("testDomain", "OrganizationID", "ThreadID", "ItemID"),
			ThreadID:       "ThreadID",
			OrganizationID: "OrganizationID",
			MessageID:      "ItemID",
			SavedQueryID:   "SavedQueryID",
			PushID:         "dedupe",
		},
	})
	test.OK(t, err)
	snsNote = &snsNotification{
		DefaultMessage: "ShortMessage2",
		IOSSandBox:     string(iData),
		IOS:            string(iData),
		Android:        string(aData),
	}
	msg, err = json.Marshal(snsNote)
	test.OK(t, err)

	// Send out the push notifications
	snsAPI.Expect(mock.NewExpectation(snsAPI.Publish, &sns.PublishInput{
		Message:          ptr.String(string(msg)),
		MessageStructure: jsonStructure,
		TargetArn:        ptr.String("account2:pushEndpoint1"),
	}))
	snsAPI.Expect(mock.NewExpectation(snsAPI.Publish, &sns.PublishInput{
		Message:          ptr.String(string(msg)),
		MessageStructure: jsonStructure,
		TargetArn:        ptr.String("account2:pushEndpoint2"),
	}))

	cSvc.processNotification(context.Background(), string(notificationData))
}

func TestProcessNotificationExternalMessage(t *testing.T) {
	dl := testdal.NewMockDAL(t)
	dc := dmock.New(t)
	snsAPI := mock.NewSNSAPI(t)
	sqsAPI := mock.NewSQSAPI(t)
	sc := smock.New(t)
	defer mock.FinishAll(dl, dc, snsAPI, sqsAPI, sc)
	svc := New(dl, dc, sc, &Config{
		NotificationSQSURL:              notificationSQSURL,
		AppleDeviceRegistrationSNSARN:   appleDeviceRegistrationSNSARN,
		AndriodDeviceRegistrationSNSARN: andriodDeviceRegistrationSNSARN,
		SQSAPI:    sqsAPI,
		SNSAPI:    snsAPI,
		WebDomain: "testDomain",
	})
	cSvc := svc.(*service)

	entitiesToNotify := []string{"entity:1", "entity:2", "entity:3", "entity:4"}
	notificationData, err := json.Marshal(&notification.Notification{
		ShortMessages: map[string]string{
			"entity:1": "",
			"entity:2": "ShortMessage2",
			"entity:3": "ShortMessage3",
			"entity:4": "ShortMessage4",
		},
		UnreadCounts: map[string]int{
			"entity:1": 1,
			"entity:2": 2,
			"entity:3": 3,
			"entity:4": 4,
		},
		CollapseKey:          "collapse",
		DedupeKey:            "dedupe",
		ThreadID:             "ThreadID",
		OrganizationID:       "OrganizationID",
		MessageID:            "ItemID",
		SavedQueryID:         "SavedQueryID",
		EntitiesToNotify:     entitiesToNotify,
		EntitiesAtReferenced: map[string]struct{}{"entity:2": {}, "entity:3": {}},
		Type:                 notification.NewMessageOnExternalThread,
	})
	test.OK(t, err)

	// Check the settings for each account
	expectFilterNodesWithNotificationsDisabled(t, sc, []string{"entity:1", "entity:2", "entity:3", "entity:4"}, []bool{true, true, false, false})

	// Check the setting for the entities
	expectFilterNodesForThreadActivityPreferences(t, sc, notification.PatientNotificationPreferencesSettingsKey, []string{"entity:1", "entity:2"}, []string{
		nsettings.ThreadActivityNotificationPreferenceAllMessages,
		nsettings.ThreadActivityNotificationPreferenceReferencedOnly,
	})

	// Lookup account IDs for the entities via their external identifiers, we should have filtered 1
	dc.Expect(mock.NewExpectation(dc.ExternalIDs, &directory.ExternalIDsRequest{
		EntityIDs: []string{"entity:1", "entity:2"},
	}).WithReturns(&directory.ExternalIDsResponse{
		ExternalIDs: []*directory.ExternalID{
			{ID: "account_1", EntityID: "entity:1"},
			{ID: "account_2", EntityID: "entity:2"},
		},
	}, nil))

	// Lookup the push configs for each external group id (account)
	dl.Expect(mock.NewExpectation(dl.PushConfigsForExternalGroupID, "account_1").WithReturns([]*dal.PushConfig{
		{PushEndpoint: "account1:pushEndpoint1", Platform: "iOS"},
		{PushEndpoint: "account1:pushEndpoint2", Platform: "android"},
	}, nil))

	// Build out expected notification structures
	iData, err := json.Marshal(&iOSPushNotification{
		PushData: &iOSPushData{
			ContentAvailable: 1,
		},
		Type:           string(notification.NewMessageOnExternalThread),
		URL:            deeplink.ThreadMessageURLShareable("testDomain", "OrganizationID", "ThreadID", "ItemID"),
		ThreadID:       "ThreadID",
		OrganizationID: "OrganizationID",
		MessageID:      "ItemID",
		SavedQueryID:   "SavedQueryID",
	})
	test.OK(t, err)
	aData, err := json.Marshal(&androidPushNotification{
		CollapseKey: "collapse",
		Priority:    "normal",
		PushData: &androidPushData{
			Type:           string(notification.NewMessageOnExternalThread),
			Message:        "",
			Background:     true,
			URL:            deeplink.ThreadMessageURLShareable("testDomain", "OrganizationID", "ThreadID", "ItemID"),
			ThreadID:       "ThreadID",
			OrganizationID: "OrganizationID",
			MessageID:      "ItemID",
			SavedQueryID:   "SavedQueryID",
			PushID:         "dedupe",
		},
	})
	test.OK(t, err)
	snsNote := &snsNotification{
		DefaultMessage: "",
		IOSSandBox:     string(iData),
		IOS:            string(iData),
		Android:        string(aData),
	}
	msg, err := json.Marshal(snsNote)
	test.OK(t, err)

	// Send out the push notifications
	snsAPI.Expect(mock.NewExpectation(snsAPI.Publish, &sns.PublishInput{
		Message:          ptr.String(string(msg)),
		MessageStructure: jsonStructure,
		TargetArn:        ptr.String("account1:pushEndpoint1"),
	}))
	snsAPI.Expect(mock.NewExpectation(snsAPI.Publish, &sns.PublishInput{
		Message:          ptr.String(string(msg)),
		MessageStructure: jsonStructure,
		TargetArn:        ptr.String("account1:pushEndpoint2"),
	}))

	// Repeat for the next thread member
	dl.Expect(mock.NewExpectation(dl.PushConfigsForExternalGroupID, "account_2").WithReturns([]*dal.PushConfig{
		{PushEndpoint: "account2:pushEndpoint1", Platform: "iOS"},
		{PushEndpoint: "account2:pushEndpoint2", Platform: "android"},
	}, nil))

	// Build out expected notification structures
	iData, err = json.Marshal(&iOSPushNotification{
		PushData: &iOSPushData{
			Alert:            "ShortMessage2",
			Sound:            "default",
			ContentAvailable: 1,
		},
		Type:           string(notification.NewMessageOnExternalThread),
		URL:            deeplink.ThreadMessageURLShareable("testDomain", "OrganizationID", "ThreadID", "ItemID"),
		ThreadID:       "ThreadID",
		OrganizationID: "OrganizationID",
		MessageID:      "ItemID",
		SavedQueryID:   "SavedQueryID",
	})
	test.OK(t, err)
	aData, err = json.Marshal(&androidPushNotification{
		CollapseKey: "collapse",
		Priority:    "normal",
		PushData: &androidPushData{
			Type:           string(notification.NewMessageOnExternalThread),
			Message:        "ShortMessage2",
			Background:     false,
			URL:            deeplink.ThreadMessageURLShareable("testDomain", "OrganizationID", "ThreadID", "ItemID"),
			ThreadID:       "ThreadID",
			OrganizationID: "OrganizationID",
			MessageID:      "ItemID",
			SavedQueryID:   "SavedQueryID",
			PushID:         "dedupe",
		},
	})
	test.OK(t, err)
	snsNote = &snsNotification{
		DefaultMessage: "ShortMessage2",
		IOSSandBox:     string(iData),
		IOS:            string(iData),
		Android:        string(aData),
	}
	msg, err = json.Marshal(snsNote)
	test.OK(t, err)

	// Send out the push notifications
	snsAPI.Expect(mock.NewExpectation(snsAPI.Publish, &sns.PublishInput{
		Message:          ptr.String(string(msg)),
		MessageStructure: jsonStructure,
		TargetArn:        ptr.String("account2:pushEndpoint1"),
	}))
	snsAPI.Expect(mock.NewExpectation(snsAPI.Publish, &sns.PublishInput{
		Message:          ptr.String(string(msg)),
		MessageStructure: jsonStructure,
		TargetArn:        ptr.String("account2:pushEndpoint2"),
	}))

	cSvc.processNotification(context.Background(), string(notificationData))
}

func accountIDsFromExternalIDs(eIDs []*directory.ExternalID) []string {
	var accountIDs []string
	for _, eID := range eIDs {
		i := strings.IndexByte(eID.ID, '_')
		if i != -1 {
			prefix := eID.ID[:(i + 1)]
			switch prefix {
			case auth.AccountIDPrefix:
				accountIDs = append(accountIDs, eID.ID)
			}
		}
	}
	return accountIDs
}
