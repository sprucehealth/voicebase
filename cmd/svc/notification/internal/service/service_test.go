package service

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/cmd/svc/notification/internal/dal"
	testdal "github.com/sprucehealth/backend/cmd/svc/notification/internal/dal/test"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	dmock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/notification/deeplink"
	"github.com/sprucehealth/backend/svc/settings"
	smock "github.com/sprucehealth/backend/svc/settings/mock"
	"github.com/sprucehealth/backend/test"
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
	defer dl.Finish()
	dc := dmock.New(t)
	defer dc.Finish()
	snsAPI := mock.NewSNSAPI(t)
	defer snsAPI.Finish()
	sqsAPI := mock.NewSQSAPI(t)
	defer sqsAPI.Finish()
	sc := smock.New(t)
	defer sc.Finish()
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
	dl.Expect(mock.NewExpectation(dl.PushConfigForDeviceID, "DeviceID").WithReturns((*dal.PushConfig)(nil), api.ErrNotFound("not found")))

	// Generate an endpoint for the device
	snsAPI.Expect(mock.NewExpectation(snsAPI.CreatePlatformEndpoint, &sns.CreatePlatformEndpointInput{
		PlatformApplicationArn: ptr.String(appleDeviceRegistrationSNSARN),
		Token: ptr.String("DeviceToken"),
		Attributes: map[string]*string{
			snsEndpointEnabledAttributeKey: ptr.String("true"),
		},
		CustomUserData: ptr.String("ExternalGroupID"),
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

	cSvc.processDeviceRegistration(string(driData))
}

func TestProcessNewDeviceRegistrationAndroid(t *testing.T) {
	dl := testdal.NewMockDAL(t)
	defer dl.Finish()
	dc := dmock.New(t)
	defer dc.Finish()
	snsAPI := mock.NewSNSAPI(t)
	defer snsAPI.Finish()
	sqsAPI := mock.NewSQSAPI(t)
	defer sqsAPI.Finish()
	sc := smock.New(t)
	defer sc.Finish()
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
	dl.Expect(mock.NewExpectation(dl.PushConfigForDeviceID, "DeviceID").WithReturns((*dal.PushConfig)(nil), api.ErrNotFound("not found")))

	// Generate an endpoint for the device
	snsAPI.Expect(mock.NewExpectation(snsAPI.CreatePlatformEndpoint, &sns.CreatePlatformEndpointInput{
		PlatformApplicationArn: ptr.String(andriodDeviceRegistrationSNSARN),
		Token:          ptr.String("DeviceToken"),
		Attributes:     map[string]*string{snsEndpointEnabledAttributeKey: ptr.String("true")},
		CustomUserData: ptr.String("ExternalGroupID"),
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

	cSvc.processDeviceRegistration(string(driData))
}

func TestProcessExistingDeviceRegistrationIOSTokenChanged(t *testing.T) {
	dl := testdal.NewMockDAL(t)
	defer dl.Finish()
	dc := dmock.New(t)
	defer dc.Finish()
	snsAPI := mock.NewSNSAPI(t)
	defer snsAPI.Finish()
	sqsAPI := mock.NewSQSAPI(t)
	defer sqsAPI.Finish()
	sc := smock.New(t)
	defer sc.Finish()
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
		DeviceToken:     "NewDeviceToken",
		Platform:        "iOS",
		PlatformVersion: "PlatformVersion",
		AppVersion:      "AppVersion",
		Device:          "Device",
		DeviceModel:     "DeviceModel",
		DeviceID:        "DeviceID",
	})
	test.OK(t, err)

	// Lookup the device and find it
	dl.Expect(mock.NewExpectation(dl.PushConfigForDeviceID, "DeviceID").WithReturns(&dal.PushConfig{
		ID: dal.PushConfigID{
			ObjectID: model.ObjectID{
				Prefix:  notification.PushConfigIDPrefix,
				Val:     1,
				IsValid: true,
			},
		},
		DeviceToken:  []byte("DeviceToken"),
		PushEndpoint: "myEndpoint",
	}, nil))

	// Update the endpoint for the device
	snsAPI.Expect(mock.NewExpectation(snsAPI.SetEndpointAttributes, &sns.SetEndpointAttributesInput{
		EndpointArn: ptr.String("myEndpoint"),
		Attributes: map[string]*string{
			snsEndpointEnabledAttributeKey: ptr.String("true"),
			snsEndpointTokenAttributeKey:   ptr.String("NewDeviceToken"),
			snsEndpointCustomUserDataKey:   ptr.String("ExternalGroupID"),
		},
	}).WithReturns(&sns.SetEndpointAttributesOutput{}, nil))

	// Update the record for the device
	dl.Expect(mock.NewExpectation(dl.UpdatePushConfig, dal.PushConfigID{
		ObjectID: model.ObjectID{
			Prefix:  notification.PushConfigIDPrefix,
			Val:     1,
			IsValid: true,
		},
	}, &dal.PushConfigUpdate{
		DeviceID:        ptr.String("DeviceID"),
		ExternalGroupID: ptr.String("ExternalGroupID"),
		Platform:        ptr.String("iOS"),
		PlatformVersion: ptr.String("PlatformVersion"),
		AppVersion:      ptr.String("AppVersion"),
		DeviceToken:     []byte("NewDeviceToken"),
	}))

	cSvc.processDeviceRegistration(string(driData))
}

func TestProcessExistingDeviceRegistrationAndroid(t *testing.T) {
	dl := testdal.NewMockDAL(t)
	defer dl.Finish()
	dc := dmock.New(t)
	defer dc.Finish()
	snsAPI := mock.NewSNSAPI(t)
	defer snsAPI.Finish()
	sqsAPI := mock.NewSQSAPI(t)
	defer sqsAPI.Finish()
	sc := smock.New(t)
	defer sc.Finish()
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
	dl.Expect(mock.NewExpectation(dl.PushConfigForDeviceID, "DeviceID").WithReturns(&dal.PushConfig{
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

	cSvc.processDeviceRegistration(string(driData))
}

func TestProcessExistingDeviceDeregistration(t *testing.T) {
	dl := testdal.NewMockDAL(t)
	defer dl.Finish()
	dc := dmock.New(t)
	defer dc.Finish()
	snsAPI := mock.NewSNSAPI(t)
	defer snsAPI.Finish()
	sqsAPI := mock.NewSQSAPI(t)
	defer sqsAPI.Finish()
	sc := smock.New(t)
	defer sc.Finish()
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

	cSvc.processDeviceDeregistration(string(ddriData))
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

func TestProcessNotification(t *testing.T) {
	dl := testdal.NewMockDAL(t)
	defer dl.Finish()
	dc := dmock.New(t)
	defer dc.Finish()
	snsAPI := mock.NewSNSAPI(t)
	defer snsAPI.Finish()
	sqsAPI := mock.NewSQSAPI(t)
	defer sqsAPI.Finish()
	sc := smock.New(t)
	defer sc.Finish()
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
	expectFilterNodesWithNotificationsDisabled(t, sc, entitiesToNotify, []bool{true, true, false, true})

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

	// Check the settings for each account
	expectFilterNodesWithNotificationsDisabled(t, sc, []string{"account_1", "account_2", "account_4"}, []bool{true, true, true})

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
		ThreadID:       "ThreadID",
		OrganizationID: "OrganizationID",
		MessageID:      "ItemID",
		SavedQueryID:   "SavedQueryID",
	})
	test.OK(t, err)
	aData, err := json.Marshal(&androidPushNotification{
		CollapseKey: "collapse",
		PushData: &androidPushData{
			Message:        "",
			Background:     true,
			UnreadCount:    1,
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
			Alert: "ShortMessage2",
			//Badge: 2,
			URL:   deeplink.OrgURL("testDomain", "OrganizationID"),
			Sound: "default",
		},
		ThreadID:       "ThreadID",
		OrganizationID: "OrganizationID",
		MessageID:      "ItemID",
		SavedQueryID:   "SavedQueryID",
	})
	test.OK(t, err)
	aData, err = json.Marshal(&androidPushNotification{
		CollapseKey: "collapse",
		PushData: &androidPushData{
			Message:        "ShortMessage2",
			Background:     false,
			UnreadCount:    2,
			URL:            deeplink.OrgURL("testDomain", "OrganizationID"),
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
			Alert: "ShortMessage4",
			////Badge: 4,
			URL:   deeplink.OrgURL("testDomain", "OrganizationID"),
			Sound: "default",
		},
		ThreadID:       "ThreadID",
		OrganizationID: "OrganizationID",
		MessageID:      "ItemID",
		SavedQueryID:   "SavedQueryID",
	})
	test.OK(t, err)
	aData, err = json.Marshal(&androidPushNotification{
		CollapseKey: "collapse",
		PushData: &androidPushData{
			Message:        "ShortMessage4",
			Background:     false,
			UnreadCount:    4,
			URL:            deeplink.OrgURL("testDomain", "OrganizationID"),
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

	cSvc.processNotification(string(notificationData))
}

func TestProcessNotificationDisabledEndpoint(t *testing.T) {
	dl := testdal.NewMockDAL(t)
	defer dl.Finish()
	dc := dmock.New(t)
	defer dc.Finish()
	snsAPI := mock.NewSNSAPI(t)
	defer snsAPI.Finish()
	sqsAPI := mock.NewSQSAPI(t)
	defer sqsAPI.Finish()
	pcID, err := dal.NewPushConfigID()
	test.OK(t, err)
	sc := smock.New(t)
	defer sc.Finish()
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

	expectFilterNodesWithNotificationsDisabled(t, sc, []string{"entity:1", "entity:2"}, []bool{true, true})

	// Lookup account IDs for the entities via their external identifiers
	dc.Expect(mock.NewExpectation(dc.ExternalIDs, &directory.ExternalIDsRequest{
		EntityIDs: []string{"entity:1", "entity:2"},
	}).WithReturns(&directory.ExternalIDsResponse{
		ExternalIDs: []*directory.ExternalID{
			{ID: "account_1", EntityID: "entity:1"},
			{ID: "account_2", EntityID: "entity:2"},
		},
	}, nil))

	expectFilterNodesWithNotificationsDisabled(t, sc, []string{"account_1", "account_2"}, []bool{true, false})

	// Lookup the push configs for each external group id (account)
	dl.Expect(mock.NewExpectation(dl.PushConfigsForExternalGroupID, "account_1").WithReturns([]*dal.PushConfig{
		{ID: pcID, PushEndpoint: "account1:pushEndpoint1", Platform: "iOS"},
		{PushEndpoint: "account1:pushEndpoint2", Platform: "android"},
	}, nil))

	// Build out expected notification structure
	iData, err := json.Marshal(&iOSPushNotification{
		PushData: &iOSPushData{
			Sound: "default",
			Alert: "ShortMessage",
			URL:   deeplink.ThreadMessageURLShareable("testDomain", "OrganizationID", "ThreadID", "ItemID"),
			//Badge: 1,
		},
		ThreadID:       "ThreadID",
		OrganizationID: "OrganizationID",
		MessageID:      "ItemID",
		SavedQueryID:   "SavedQueryID",
	})
	test.OK(t, err)
	aData, err := json.Marshal(&androidPushNotification{
		CollapseKey: "collapse",
		PushData: &androidPushData{
			Message:        "ShortMessage",
			URL:            deeplink.ThreadMessageURLShareable("testDomain", "OrganizationID", "ThreadID", "ItemID"),
			ThreadID:       "ThreadID",
			OrganizationID: "OrganizationID",
			MessageID:      "ItemID",
			SavedQueryID:   "SavedQueryID",
			PushID:         "dedupe",
			UnreadCount:    1,
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

	cSvc.processNotification(string(notificationData))
}
