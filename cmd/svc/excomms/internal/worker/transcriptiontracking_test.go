package worker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/golang/mock/gomock"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	dmock "github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/libs/transcription"
	"github.com/sprucehealth/backend/libs/transcription/transcriptionmock"
	"github.com/sprucehealth/backend/svc/excomms"
)

func TestTranscriptionTracking_Completed(t *testing.T) {

	req := trackTranscriptionRequest{
		JobID:           "job1",
		MediaID:         "media1",
		MediaDurationNS: 1000,
		RawMessageID:    uint64(123),
	}

	mDAL := dmock.New(t)
	mSNS := mock.NewSNSAPI(t)
	defer mDAL.Finish()
	defer mSNS.Finish()

	mDAL.Expect(mock.NewExpectation(mDAL.LookupTranscriptionJob, req.MediaID).WithReturns(
		&models.TranscriptionJob{
			AvailableAfter: time.Now().Add(-time.Minute),
			Completed:      true,
		}, nil))

	w := &transcriptionTracker{
		dal:    mDAL,
		snsAPI: mSNS,
		clk:    clock.New(),
	}

	jsonData, err := json.Marshal(req)
	test.OK(t, err)

	data := base64.StdEncoding.EncodeToString(jsonData)

	test.OK(t, w.processTranscription(context.Background(), data))
}

func TestTranscriptionTracking_Processing(t *testing.T) {
	req := trackTranscriptionRequest{
		JobID:           "job1",
		MediaID:         "media1",
		MediaDurationNS: 1000,
		RawMessageID:    uint64(123),
	}

	mDAL := dmock.New(t)
	mSNS := mock.NewSNSAPI(t)
	defer mDAL.Finish()
	defer mSNS.Finish()

	mDAL.Expect(mock.NewExpectation(mDAL.LookupTranscriptionJob, req.MediaID).WithReturns(
		&models.TranscriptionJob{
			AvailableAfter: time.Now().Add(10 * time.Minute),
		}, nil))

	w := &transcriptionTracker{
		dal:    mDAL,
		snsAPI: mSNS,
		clk:    clock.New(),
	}

	jsonData, err := json.Marshal(req)
	test.OK(t, err)

	data := base64.StdEncoding.EncodeToString(jsonData)

	test.Equals(t, awsutil.ErrMsgNotProcessedYet, w.processTranscription(context.Background(), data))
}

func TestTranscriptionTracking_TimedOut(t *testing.T) {
	req := trackTranscriptionRequest{
		JobID:           "job1",
		MediaID:         "media1",
		MediaDurationNS: 1000,
		RawMessageID:    uint64(123),
	}

	mDAL := dmock.New(t)
	mSNS := mock.NewSNSAPI(t)
	defer mDAL.Finish()
	defer mSNS.Finish()
	clk := clock.NewManaged(time.Now())

	mDAL.Expect(mock.NewExpectation(mDAL.LookupTranscriptionJob, req.MediaID).WithReturns(
		&models.TranscriptionJob{
			Created: time.Now().Add(-20 * time.Minute),
		}, nil))

	mDAL.Expect(mock.NewExpectation(mDAL.UpdateTranscriptionJob, req.MediaID, &dal.TranscriptionJobUpdate{
		AvailableAfter: ptr.Time(clk.Now().Add(time.Minute)),
	}))

	mDAL.Expect(mock.NewExpectation(mDAL.IncomingRawMessage, req.RawMessageID).WithReturns(&rawmsg.Incoming{
		Timestamp: uint64(clk.Now().Unix()),
		Message: &rawmsg.Incoming_Twilio{
			Twilio: &rawmsg.TwilioParams{
				From:              "+11111111111",
				To:                "+12222222222",
				RecordingDuration: 50,
			},
		},
	}, nil))

	mDAL.Expect(mock.NewExpectation(mDAL.UpdateTranscriptionJob, req.MediaID, &dal.TranscriptionJobUpdate{
		Completed:          ptr.Bool(true),
		TimedOut:           ptr.Bool(true),
		CompletedTimestamp: ptr.Time(clk.Now()),
	}))

	externalMessage := &excomms.PublishedExternalMessage{
		FromChannelID: "+11111111111",
		ToChannelID:   "+12222222222",
		Timestamp:     uint64(clk.Now().Unix()),
		Direction:     excomms.PublishedExternalMessage_INBOUND,
		Type:          excomms.PublishedExternalMessage_INCOMING_CALL_EVENT,
		Item: &excomms.PublishedExternalMessage_Incoming{
			Incoming: &excomms.IncomingCallEventItem{
				Type:                excomms.IncomingCallEventItem_LEFT_VOICEMAIL,
				DurationInSeconds:   50,
				VoicemailMediaID:    req.MediaID,
				VoicemailDurationNS: req.MediaDurationNS,
				TranscriptionText:   "",
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

	w := &transcriptionTracker{
		dal:                  mDAL,
		snsAPI:               mSNS,
		clk:                  clk,
		externalMessageTopic: "topic",
	}

	jsonData, err := json.Marshal(req)
	test.OK(t, err)

	test.OK(t, w.processTranscription(context.Background(), base64.StdEncoding.EncodeToString(jsonData)))
}

func TestTranscriptionTracking_TranscriptionCompleted(t *testing.T) {
	environment.SetCurrent(environment.Test)
	req := trackTranscriptionRequest{
		JobID:           "job1",
		MediaID:         "media1",
		MediaDurationNS: 1000,
		RawMessageID:    uint64(123),
	}

	mDAL := dmock.New(t)
	mSNS := mock.NewSNSAPI(t)
	defer mDAL.Finish()
	defer mSNS.Finish()
	clk := clock.NewManaged(time.Now())

	ctrl := gomock.NewController(t)
	trmock := transcriptionmock.NewMockProvider(ctrl)
	defer ctrl.Finish()

	mDAL.Expect(mock.NewExpectation(mDAL.LookupTranscriptionJob, req.MediaID).WithReturns(
		&models.TranscriptionJob{
			Created: time.Now().Add(-time.Minute),
		}, nil))

	mDAL.Expect(mock.NewExpectation(mDAL.UpdateTranscriptionJob, req.MediaID, &dal.TranscriptionJobUpdate{
		AvailableAfter: ptr.Time(clk.Now().Add(time.Minute)),
	}))

	mDAL.Expect(mock.NewExpectation(mDAL.IncomingRawMessage, req.RawMessageID).WithReturns(&rawmsg.Incoming{
		Timestamp: uint64(clk.Now().Unix()),
		Message: &rawmsg.Incoming_Twilio{
			Twilio: &rawmsg.TwilioParams{
				From:              "+11111111111",
				To:                "+12222222222",
				RecordingDuration: 50,
			},
		},
	}, nil))

	mDAL.Expect(mock.NewExpectation(mDAL.UpdateTranscriptionJob, req.MediaID, &dal.TranscriptionJobUpdate{
		Completed:          ptr.Bool(true),
		CompletedTimestamp: ptr.Time(clk.Now()),
	}))

	trmock.EXPECT().LookupTranscriptionJob(req.JobID).Return(&transcription.Job{
		Status:            transcription.JobStatusCompleted,
		TranscriptionText: "Hi this is a test",
	}, nil)

	externalMessage := &excomms.PublishedExternalMessage{
		FromChannelID: "+11111111111",
		ToChannelID:   "+12222222222",
		Timestamp:     uint64(clk.Now().Unix()),
		Direction:     excomms.PublishedExternalMessage_INBOUND,
		Type:          excomms.PublishedExternalMessage_INCOMING_CALL_EVENT,
		Item: &excomms.PublishedExternalMessage_Incoming{
			Incoming: &excomms.IncomingCallEventItem{
				Type:                excomms.IncomingCallEventItem_LEFT_VOICEMAIL,
				DurationInSeconds:   50,
				VoicemailMediaID:    req.MediaID,
				VoicemailDurationNS: req.MediaDurationNS,
				TranscriptionText:   "Hi this is a test",
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

	w := &transcriptionTracker{
		dal:                   mDAL,
		snsAPI:                mSNS,
		clk:                   clk,
		externalMessageTopic:  "topic",
		transcriptionProvider: trmock,
	}

	jsonData, err := json.Marshal(req)
	test.OK(t, err)

	test.OK(t, w.processTranscription(context.Background(), base64.StdEncoding.EncodeToString(jsonData)))
}

func TestTranscriptionTracking_TranscriptionFailed(t *testing.T) {
	environment.SetCurrent(environment.Test)
	req := trackTranscriptionRequest{
		JobID:           "job1",
		MediaID:         "media1",
		MediaDurationNS: 1000,
		RawMessageID:    uint64(123),
	}

	mDAL := dmock.New(t)
	mSNS := mock.NewSNSAPI(t)
	defer mDAL.Finish()
	defer mSNS.Finish()
	clk := clock.NewManaged(time.Now())

	ctrl := gomock.NewController(t)
	trmock := transcriptionmock.NewMockProvider(ctrl)
	defer ctrl.Finish()

	mDAL.Expect(mock.NewExpectation(mDAL.LookupTranscriptionJob, req.MediaID).WithReturns(
		&models.TranscriptionJob{
			Created: time.Now().Add(-time.Minute),
		}, nil))

	mDAL.Expect(mock.NewExpectation(mDAL.UpdateTranscriptionJob, req.MediaID, &dal.TranscriptionJobUpdate{
		AvailableAfter: ptr.Time(clk.Now().Add(time.Minute)),
	}))

	mDAL.Expect(mock.NewExpectation(mDAL.IncomingRawMessage, req.RawMessageID).WithReturns(&rawmsg.Incoming{
		Timestamp: uint64(clk.Now().Unix()),
		Message: &rawmsg.Incoming_Twilio{
			Twilio: &rawmsg.TwilioParams{
				From:              "+11111111111",
				To:                "+12222222222",
				RecordingDuration: 50,
			},
		},
	}, nil))

	mDAL.Expect(mock.NewExpectation(mDAL.UpdateTranscriptionJob, req.MediaID, &dal.TranscriptionJobUpdate{
		Errored:            ptr.Bool(true),
		CompletedTimestamp: ptr.Time(clk.Now()),
	}))

	trmock.EXPECT().LookupTranscriptionJob(req.JobID).Return(&transcription.Job{
		Status: transcription.JobStatusFailed,
	}, nil)

	externalMessage := &excomms.PublishedExternalMessage{
		FromChannelID: "+11111111111",
		ToChannelID:   "+12222222222",
		Timestamp:     uint64(clk.Now().Unix()),
		Direction:     excomms.PublishedExternalMessage_INBOUND,
		Type:          excomms.PublishedExternalMessage_INCOMING_CALL_EVENT,
		Item: &excomms.PublishedExternalMessage_Incoming{
			Incoming: &excomms.IncomingCallEventItem{
				Type:                excomms.IncomingCallEventItem_LEFT_VOICEMAIL,
				DurationInSeconds:   50,
				VoicemailMediaID:    req.MediaID,
				VoicemailDurationNS: req.MediaDurationNS,
				TranscriptionText:   "",
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

	w := &transcriptionTracker{
		dal:                   mDAL,
		snsAPI:                mSNS,
		clk:                   clk,
		externalMessageTopic:  "topic",
		transcriptionProvider: trmock,
	}

	jsonData, err := json.Marshal(req)
	test.OK(t, err)

	test.OK(t, w.processTranscription(context.Background(), base64.StdEncoding.EncodeToString(jsonData)))
}

func TestTranscriptionTracking_TranscriptionNotCompleted(t *testing.T) {
	environment.SetCurrent(environment.Test)
	req := trackTranscriptionRequest{
		JobID:           "job1",
		MediaID:         "media1",
		MediaDurationNS: 1000,
		RawMessageID:    uint64(123),
	}

	mDAL := dmock.New(t)
	mSNS := mock.NewSNSAPI(t)
	defer mDAL.Finish()
	defer mSNS.Finish()
	clk := clock.NewManaged(time.Now())

	ctrl := gomock.NewController(t)
	trmock := transcriptionmock.NewMockProvider(ctrl)
	defer ctrl.Finish()

	mDAL.Expect(mock.NewExpectation(mDAL.LookupTranscriptionJob, req.MediaID).WithReturns(
		&models.TranscriptionJob{
			Created: time.Now().Add(-time.Minute),
		}, nil))

	mDAL.Expect(mock.NewExpectation(mDAL.UpdateTranscriptionJob, req.MediaID, &dal.TranscriptionJobUpdate{
		AvailableAfter: ptr.Time(clk.Now().Add(time.Minute)),
	}))

	trmock.EXPECT().LookupTranscriptionJob(req.JobID).Return(&transcription.Job{
		Status: transcription.JobStatusProcessing,
	}, nil)

	w := &transcriptionTracker{
		dal:                   mDAL,
		snsAPI:                mSNS,
		clk:                   clk,
		externalMessageTopic:  "topic",
		transcriptionProvider: trmock,
	}

	jsonData, err := json.Marshal(req)
	test.OK(t, err)

	test.Equals(t, awsutil.ErrMsgNotProcessedYet, w.processTranscription(context.Background(), base64.StdEncoding.EncodeToString(jsonData)))
}
