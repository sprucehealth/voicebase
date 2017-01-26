package worker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/golang/mock/gomock"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	dmock "github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	isns "github.com/sprucehealth/backend/cmd/svc/excomms/internal/sns"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/worker/uploadermock"
	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/libs/transcription"
	"github.com/sprucehealth/backend/libs/transcription/transcriptionmock"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/settings/settingsmock"
)

func TestParseAddress(t *testing.T) {

	addr, err := parseAddress("Joe Schmoe (Joe) <joe@schmoe.com>")
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, "joe@schmoe.com", addr.Address)

	addr, err = parseAddress("Joe Schmoe <Joe> <joe@schmoe.com>")
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, "joe@schmoe.com", addr.Address)

	addr, err = parseAddress("Joe <3 Schmoe <joe@schmoe.com>")
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, "joe@schmoe.com", addr.Address)

	addr, err = parseAddress("Joe Schmoe <joe@schmoe.com>")
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, "joe@schmoe.com", addr.Address)

	addr, err = parseAddress("\"Joe Schmoe\" <joe@schmoe.com>")
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, "joe@schmoe.com", addr.Address)

	addr, err = parseAddress("<joe@schmoe.com>")
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, "joe@schmoe.com", addr.Address)

	addr, err = parseAddress("joe@schmoe.com")
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, "joe@schmoe.com", addr.Address)

	addr, err = parseAddress("I<joe@schmoe.com>")
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, "joe@schmoe.com", addr.Address)

	addr, err = parseAddress(" 		<joe@schmoe.com>")
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, "joe@schmoe.com", addr.Address)
}

func TestIncomingVoicemail_NoTranscription(t *testing.T) {
	environment.SetCurrent(environment.Test)
	rm := isns.IncomingRawMessageNotification{
		ID: 100,
	}

	mDAL := dmock.New(t)
	mSNS := mock.NewSNSAPI(t)
	mUploader := uploadermock.New(t)
	ctrl := gomock.NewController(t)
	mSettings := settingsmock.NewMockSettingsClient(ctrl)
	defer mDAL.Finish()
	defer mSNS.Finish()
	defer mUploader.Finish()
	defer ctrl.Finish()

	incomingMsg := &rawmsg.Incoming{
		Type: rawmsg.Incoming_TWILIO_VOICEMAIL,
		Message: &rawmsg.Incoming_Twilio{
			Twilio: &rawmsg.TwilioParams{
				From:              "+11111111111",
				To:                "+12222222222",
				RecordingURL:      "http://test.com",
				CallSID:           "callSID",
				RecordingSID:      "recordingSID",
				RecordingDuration: 1000,
			},
		},
		Timestamp: uint64(time.Now().Unix()),
	}

	mDAL.Expect(mock.NewExpectation(mDAL.IncomingRawMessage, rm.ID).WithReturns(incomingMsg, nil))

	mUploader.Expect(mock.NewExpectation(mUploader.Upload, "audio/mpeg", "http://test.com.mp3").WithReturns(&models.Media{
		ID:   "123-456",
		Type: "audio/mpeg",
	}, nil))

	mUploader.Expect(mock.NewExpectation(mUploader.Upload, "audio/wav", "http://test.com").WithReturns(&models.Media{
		ID:   "123-456.wav",
		Type: "audio/wav",
	}, nil))

	mDAL.Expect(mock.NewExpectation(mDAL.StoreMedia, []*models.Media{
		{
			ID:         "123-456",
			Type:       "audio/mpeg",
			ResourceID: "recordingSID",
			Duration:   1000000000000,
		},
		{
			ID:         "123-456.wav",
			Type:       "audio/wav",
			ResourceID: "recordingSID",
			Duration:   1000000000000,
		},
	}))

	mDAL.Expect(mock.NewExpectation(mDAL.StoreIncomingRawMessage, incomingMsg).WithReturns(uint64(123), nil))

	mDAL.Expect(mock.NewExpectation(mDAL.LookupIncomingCall, "callSID").WithReturns(&models.IncomingCall{
		CallSID:        "callSID",
		OrganizationID: "orgID",
	}, nil))

	mSettings.EXPECT().GetValues(context.Background(), &settings.GetValuesRequest{
		NodeID: "orgID",
		Keys: []*settings.ConfigKey{
			{
				Key: excommsSettings.ConfigKeyTranscribeVoicemail,
			},
			{
				Key: excommsSettings.ConfigKeyTranscriptionProvider,
			},
		},
	}).Return(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: false,
					},
				},
			},
			{
				Value: &settings.Value_SingleSelect{
					SingleSelect: &settings.SingleSelectValue{
						Item: &settings.ItemValue{
							ID: excommsSettings.TranscriptionProviderTwilio,
						},
					},
				},
			},
		},
	}, nil)

	externalMessage := &excomms.PublishedExternalMessage{
		FromChannelID: "+11111111111",
		ToChannelID:   "+12222222222",
		Timestamp:     incomingMsg.Timestamp,
		Direction:     excomms.PublishedExternalMessage_INBOUND,
		Type:          excomms.PublishedExternalMessage_INCOMING_CALL_EVENT,
		Item: &excomms.PublishedExternalMessage_Incoming{
			Incoming: &excomms.IncomingCallEventItem{
				Type:                excomms.IncomingCallEventItem_LEFT_VOICEMAIL,
				DurationInSeconds:   1000,
				VoicemailMediaID:    "123-456",
				VoicemailDurationNS: 1000000000000,
			},
		},
	}

	data, err := externalMessage.Marshal()
	test.OK(t, err)
	encodedData := base64.StdEncoding.EncodeToString(data)

	mSNS.Expect(mock.NewExpectation(mSNS.Publish, &sns.PublishInput{
		Message:  ptr.String(encodedData),
		TopicArn: ptr.String("topic"),
	}))

	w := &IncomingRawMessageWorker{
		snsAPI:               mSNS,
		settings:             mSettings,
		uploader:             mUploader,
		dal:                  mDAL,
		externalMessageTopic: "topic",
	}

	test.OK(t, w.process(&rm))

}

func TestIncomingVoicemail_VoicebaseTranscription(t *testing.T) {
	environment.SetCurrent(environment.Test)
	rm := isns.IncomingRawMessageNotification{
		ID: 100,
	}

	mDAL := dmock.New(t)
	mSQS := mock.NewSQSAPI(t)
	mUploader := uploadermock.New(t)
	ctrl := gomock.NewController(t)
	mTranscription := transcriptionmock.NewMockProvider(ctrl)
	mSettings := settingsmock.NewMockSettingsClient(ctrl)
	defer mDAL.Finish()
	defer mSQS.Finish()
	defer mUploader.Finish()
	defer ctrl.Finish()

	mclk := clock.NewManaged(time.Now())

	teststore := storage.NewTestStore(nil)

	incomingMsg := &rawmsg.Incoming{
		Type: rawmsg.Incoming_TWILIO_VOICEMAIL,
		Message: &rawmsg.Incoming_Twilio{
			Twilio: &rawmsg.TwilioParams{
				From:              "+11111111111",
				To:                "+12222222222",
				RecordingURL:      "http://test.com",
				CallSID:           "callSID",
				RecordingSID:      "recordingSID",
				RecordingDuration: 1000,
			},
		},
		Timestamp: uint64(time.Now().Unix()),
	}

	mDAL.Expect(mock.NewExpectation(mDAL.IncomingRawMessage, rm.ID).WithReturns(incomingMsg, nil))

	mUploader.Expect(mock.NewExpectation(mUploader.Upload, "audio/mpeg", "http://test.com.mp3").WithReturns(&models.Media{
		ID:   "123-456",
		Type: "audio/mpeg",
	}, nil))

	mUploader.Expect(mock.NewExpectation(mUploader.Upload, "audio/wav", "http://test.com").WithReturns(&models.Media{
		ID:   "123-456.wav",
		Type: "audio/wav",
	}, nil))

	mDAL.Expect(mock.NewExpectation(mDAL.StoreMedia, []*models.Media{
		{
			ID:         "123-456",
			Type:       "audio/mpeg",
			ResourceID: "recordingSID",
			Duration:   1000000000000,
		},
		{
			ID:         "123-456.wav",
			Type:       "audio/wav",
			ResourceID: "recordingSID",
			Duration:   1000000000000,
		},
	}))

	mDAL.Expect(mock.NewExpectation(mDAL.StoreIncomingRawMessage, incomingMsg).WithReturns(uint64(123), nil))

	mDAL.Expect(mock.NewExpectation(mDAL.LookupIncomingCall, "callSID").WithReturns(&models.IncomingCall{
		CallSID:        "callSID",
		OrganizationID: "orgID",
	}, nil))

	mDAL.Expect(mock.NewExpectation(mDAL.LookupTranscriptionJob, "123-456").WithReturns((*models.TranscriptionJob)(nil), dal.ErrTranscriptionJobNotFound))

	mSettings.EXPECT().GetValues(context.Background(), &settings.GetValuesRequest{
		NodeID: "orgID",
		Keys: []*settings.ConfigKey{
			{
				Key: excommsSettings.ConfigKeyTranscribeVoicemail,
			},
			{
				Key: excommsSettings.ConfigKeyTranscriptionProvider,
			},
		},
	}).Return(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: true,
					},
				},
			},
			{
				Value: &settings.Value_SingleSelect{
					SingleSelect: &settings.SingleSelectValue{
						Item: &settings.ItemValue{
							ID: excommsSettings.TranscriptionProviderVoicebase,
						},
					},
				},
			},
		},
	}, nil)

	mTranscription.EXPECT().SubmitTranscriptionJob("123-456.wav").Return(&transcription.Job{
		ID: "jobID",
	}, nil)

	mDAL.Expect(mock.NewExpectation(mDAL.InsertTranscriptionJob, &models.TranscriptionJob{
		MediaID:        "123-456",
		JobID:          "jobID",
		AvailableAfter: mclk.Now(),
	}))

	req := &trackTranscriptionRequest{
		JobID:           "jobID",
		MediaID:         "123-456",
		RawMessageID:    rm.ID,
		UrgentVoicemail: false,
	}

	jsonData, err := json.Marshal(req)
	test.OK(t, err)

	msg := base64.StdEncoding.EncodeToString(jsonData)
	mSQS.Expect(mock.NewExpectation(mSQS.SendMessage, &sqs.SendMessageInput{
		QueueUrl:    ptr.String("transcriptionSQSURL"),
		MessageBody: &msg,
	}))

	w := &IncomingRawMessageWorker{
		sqsAPI:   mSQS,
		settings: mSettings,
		uploader: mUploader,
		dal:      mDAL,
		clk:      mclk,
		transcriptionTrackingSQSURL: "transcriptionSQSURL",
		store: teststore,
		transcriptionProvider: mTranscription,
	}

	test.OK(t, w.process(&rm))
}
