package transcription

type JobStatus string

const (
	JobStatusSubmitted JobStatus = "JOB_SUBMITTED"
	JobStatusCompleted JobStatus = "JOB_COMPLETED"
)

type Job struct {
	ID                string
	Status            JobStatus
	TranscriptionText string
}

type Provider interface {
	SubmitTranscriptionJob(url string) (*Job, error)
	LookupTranscriptionJob(id string) (*Job, error)
	DeleteMedia(mediaID string) error
}
