package transcription

import (
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/go-voicebase"
)

type voicebaseTranscriptionProvider struct{}

func NewVoicebaseProvider(bearerToken string) Provider {
	voicebase.BearerToken = bearerToken
	return &voicebaseTranscriptionProvider{}
}

func (v voicebaseTranscriptionProvider) SubmitTranscriptionJob(url string) (*Job, error) {
	id, err := voicebase.UploadMedia(url)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &Job{
		ID:     id,
		Status: JobStatusSubmitted,
	}, nil
}

func (v voicebaseTranscriptionProvider) LookupTranscriptionJob(id string) (*Job, error) {
	media, err := voicebase.GetMedia(id)
	if err != nil {
		return nil, errors.Trace(err)
	}

	status := JobStatusUnknown

	switch media.Status {
	case "finished":
		status = JobStatusCompleted
	case "running":
		status = JobStatusProcessing
	case "accepted":
		status = JobStatusSubmitted
	}

	return &Job{
		ID:                media.ID,
		Status:            status,
		TranscriptionText: media.TranscriptionText(),
	}, nil
}
