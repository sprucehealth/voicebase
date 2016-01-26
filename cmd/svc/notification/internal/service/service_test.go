package service

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/cmd/svc/notification/internal/dal"
	testdal "github.com/sprucehealth/backend/cmd/svc/notification/internal/dal/test"
	"github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	dmock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/test"
)

func TestExternalIDToAccountIDTransformation(t *testing.T) {
	externalIDs := []*directory.ExternalID{
		&directory.ExternalID{
			ID: auth.AccountIDPrefix + "215610700746457088",
		},
		&directory.ExternalID{
			ID: auth.AccountIDPrefix + "215610700746457090",
		},
		&directory.ExternalID{
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
	snsAPI := mock.NewMockSNSAPI(t)
	defer snsAPI.Finish()
	sqsAPI := mock.NewMockSQSAPI(t)
	defer sqsAPI.Finish()
	svc := New(dl, dc, &Config{
		DeviceRegistrationSQSURL:        deviceRegistrationSQSURL,
		AppleDeviceRegistrationSNSARN:   appleDeviceRegistrationSNSARN,
		AndriodDeviceRegistrationSNSARN: andriodDeviceRegistrationSNSARN,
		SNSAPI: snsAPI,
		SQSAPI: sqsAPI,
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

	// Generate an endpoint for the device
	snsAPI.Expect(mock.NewExpectation(snsAPI.CreatePlatformEndpoint, &sns.CreatePlatformEndpointInput{
		PlatformApplicationArn: ptr.String(appleDeviceRegistrationSNSARN),
		Token: ptr.String("DeviceToken"),
	}).WithReturns(&sns.CreatePlatformEndpointOutput{
		EndpointArn: ptr.String("iOSEnpointARN"),
	}, nil))

	// Lookup the device and don't find it
	dl.Expect(mock.NewExpectation(dl.PushConfigForDeviceID, "DeviceID").WithReturns((*dal.PushConfig)(nil), api.ErrNotFound("not found")))

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

	cSvc.processDeviceRegistration(driData)
}

func TestProcessNewDeviceRegistrationAndroid(t *testing.T) {
	dl := testdal.NewMockDAL(t)
	defer dl.Finish()
	dc := dmock.New(t)
	defer dc.Finish()
	snsAPI := mock.NewMockSNSAPI(t)
	defer snsAPI.Finish()
	sqsAPI := mock.NewMockSQSAPI(t)
	defer sqsAPI.Finish()
	svc := New(dl, dc, &Config{
		DeviceRegistrationSQSURL:        deviceRegistrationSQSURL,
		AppleDeviceRegistrationSNSARN:   appleDeviceRegistrationSNSARN,
		AndriodDeviceRegistrationSNSARN: andriodDeviceRegistrationSNSARN,
		SNSAPI: snsAPI,
		SQSAPI: sqsAPI,
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

	// Generate an endpoint for the device
	snsAPI.Expect(mock.NewExpectation(snsAPI.CreatePlatformEndpoint, &sns.CreatePlatformEndpointInput{
		PlatformApplicationArn: ptr.String(andriodDeviceRegistrationSNSARN),
		Token: ptr.String("DeviceToken"),
	}).WithReturns(&sns.CreatePlatformEndpointOutput{
		EndpointArn: ptr.String("androidEnpointARN"),
	}, nil))

	// Lookup the device and don't find it
	dl.Expect(mock.NewExpectation(dl.PushConfigForDeviceID, "DeviceID").WithReturns((*dal.PushConfig)(nil), api.ErrNotFound("not found")))

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

	cSvc.processDeviceRegistration(driData)
}

func TestProcessExistingDeviceRegistrationIOS(t *testing.T) {
	dl := testdal.NewMockDAL(t)
	defer dl.Finish()
	dc := dmock.New(t)
	defer dc.Finish()
	snsAPI := mock.NewMockSNSAPI(t)
	defer snsAPI.Finish()
	sqsAPI := mock.NewMockSQSAPI(t)
	defer sqsAPI.Finish()
	svc := New(dl, dc, &Config{
		DeviceRegistrationSQSURL:        deviceRegistrationSQSURL,
		AppleDeviceRegistrationSNSARN:   appleDeviceRegistrationSNSARN,
		AndriodDeviceRegistrationSNSARN: andriodDeviceRegistrationSNSARN,
		SNSAPI: snsAPI,
		SQSAPI: sqsAPI,
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

	// Generate an endpoint for the device
	snsAPI.Expect(mock.NewExpectation(snsAPI.CreatePlatformEndpoint, &sns.CreatePlatformEndpointInput{
		PlatformApplicationArn: ptr.String(appleDeviceRegistrationSNSARN),
		Token: ptr.String("DeviceToken"),
	}).WithReturns(&sns.CreatePlatformEndpointOutput{
		EndpointArn: ptr.String("iOSEnpointARN"),
	}, nil))

	// Lookup the device and find it
	dl.Expect(mock.NewExpectation(dl.PushConfigForDeviceID, "DeviceID").WithReturns(&dal.PushConfig{
		ID: dal.PushConfigID{
			ObjectID: model.ObjectID{
				Prefix:  notification.PushConfigIDPrefix,
				Val:     1,
				IsValid: true,
			},
		},
	}, nil))

	// Insert a new record for the device
	dl.Expect(mock.NewExpectation(dl.UpdatePushConfig, dal.PushConfigID{
		ObjectID: model.ObjectID{
			Prefix:  notification.PushConfigIDPrefix,
			Val:     1,
			IsValid: true,
		},
	}, &dal.PushConfigUpdate{
		DeviceToken:     []byte("DeviceToken"),
		ExternalGroupID: ptr.String("ExternalGroupID"),
		Platform:        ptr.String("iOS"),
		PlatformVersion: ptr.String("PlatformVersion"),
		AppVersion:      ptr.String("AppVersion"),
		PushEndpoint:    ptr.String("iOSEnpointARN"),
	}))

	cSvc.processDeviceRegistration(driData)
}

func TestProcessExistingDeviceRegistrationAndroid(t *testing.T) {
	dl := testdal.NewMockDAL(t)
	defer dl.Finish()
	dc := dmock.New(t)
	defer dc.Finish()
	snsAPI := mock.NewMockSNSAPI(t)
	defer snsAPI.Finish()
	sqsAPI := mock.NewMockSQSAPI(t)
	defer sqsAPI.Finish()
	svc := New(dl, dc, &Config{
		DeviceRegistrationSQSURL:        deviceRegistrationSQSURL,
		AppleDeviceRegistrationSNSARN:   appleDeviceRegistrationSNSARN,
		AndriodDeviceRegistrationSNSARN: andriodDeviceRegistrationSNSARN,
		SNSAPI: snsAPI,
		SQSAPI: sqsAPI,
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

	// Generate an endpoint for the device
	snsAPI.Expect(mock.NewExpectation(snsAPI.CreatePlatformEndpoint, &sns.CreatePlatformEndpointInput{
		PlatformApplicationArn: ptr.String(andriodDeviceRegistrationSNSARN),
		Token: ptr.String("DeviceToken"),
	}).WithReturns(&sns.CreatePlatformEndpointOutput{
		EndpointArn: ptr.String("androidEnpointARN"),
	}, nil))

	// Lookup the device and find it
	dl.Expect(mock.NewExpectation(dl.PushConfigForDeviceID, "DeviceID").WithReturns(&dal.PushConfig{
		ID: dal.PushConfigID{
			ObjectID: model.ObjectID{
				Prefix:  notification.PushConfigIDPrefix,
				Val:     1,
				IsValid: true,
			},
		},
	}, nil))

	// Insert a new record for the device
	dl.Expect(mock.NewExpectation(dl.UpdatePushConfig, dal.PushConfigID{
		ObjectID: model.ObjectID{
			Prefix:  notification.PushConfigIDPrefix,
			Val:     1,
			IsValid: true,
		},
	}, &dal.PushConfigUpdate{
		DeviceToken:     []byte("DeviceToken"),
		ExternalGroupID: ptr.String("ExternalGroupID"),
		Platform:        ptr.String("android"),
		PlatformVersion: ptr.String("PlatformVersion"),
		AppVersion:      ptr.String("AppVersion"),
		PushEndpoint:    ptr.String("androidEnpointARN"),
	}))

	cSvc.processDeviceRegistration(driData)
}

func TestProcessNotification(t *testing.T) {
	dl := testdal.NewMockDAL(t)
	defer dl.Finish()
	dc := dmock.New(t)
	defer dc.Finish()
	snsAPI := mock.NewMockSNSAPI(t)
	defer snsAPI.Finish()
	sqsAPI := mock.NewMockSQSAPI(t)
	defer sqsAPI.Finish()
	svc := New(dl, dc, &Config{
		NotificationSQSURL:              notificationSQSURL,
		AppleDeviceRegistrationSNSARN:   appleDeviceRegistrationSNSARN,
		AndriodDeviceRegistrationSNSARN: andriodDeviceRegistrationSNSARN,
		SNSAPI: snsAPI,
		SQSAPI: sqsAPI,
	})
	cSvc := svc.(*service)

	notificationData, err := json.Marshal(&notification.Notification{
		ShortMessage:     "ShortMessage",
		ThreadID:         "ThreadID",
		OrganizationID:   "OrganizationID",
		EntitiesToNotify: []string{"entity:1", "entity:2"},
	})
	test.OK(t, err)

	// Lookup account IDs for the entities via their external identifiers
	dc.Expect(mock.NewExpectation(dc.ExternalIDs, &directory.ExternalIDsRequest{
		EntityIDs: []string{"entity:1", "entity:2"},
	}).WithReturns(&directory.ExternalIDsResponse{
		ExternalIDs: []*directory.ExternalID{
			&directory.ExternalID{ID: "account_1"},
			&directory.ExternalID{ID: "account_2"},
		},
	}, nil))

	// Lookup the push configs for each external group id (account)
	dl.Expect(mock.NewExpectation(dl.PushConfigsForExternalGroupID, "account_1").WithReturns([]*dal.PushConfig{
		&dal.PushConfig{PushEndpoint: "account1:pushEndpoint1", Platform: "iOS"},
		&dal.PushConfig{PushEndpoint: "account1:pushEndpoint2", Platform: "android"},
	}, nil))

	// Build out expected notification structure
	iOSNotif := &iOSPushNotification{
		PushData: &iOSPushData{
			Alert: "ShortMessage",
			URL:   threadActivityURL("OrganizationID", "ThreadID"),
		},
		ThreadID: "ThreadID",
	}
	snsNote := &snsNotification{
		DefaultMessage: "ShortMessage",
		IOSSandBox:     iOSNotif,
		IOS:            iOSNotif,
		Android: &androidPushNotification{
			PushData: &androidPushData{
				Message:  "ShortMessage",
				URL:      threadActivityURL("OrganizationID", "ThreadID"),
				ThreadID: "ThreadID",
			},
		},
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
		&dal.PushConfig{PushEndpoint: "account2:pushEndpoint1", Platform: "iOS"},
		&dal.PushConfig{PushEndpoint: "account2:pushEndpoint2", Platform: "android"},
		&dal.PushConfig{PushEndpoint: "account2:pushEndpoint2", Platform: "unknown"},
	}, nil))

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

	cSvc.processNotification(notificationData)
}
