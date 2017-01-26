package transcription

type JobStatus string

const (
	JobStatusSubmitted  JobStatus = "SUBMITTED"
	JobStatusProcessing JobStatus = "PROCESSING"
	JobStatusCompleted  JobStatus = "COMPLETED"
	JobStatusFailed     JobStatus = "FAILED"
	JobStatusUnknown    JobStatus = "UNKNOWN"
)

type Job struct {
	ID                string
	Status            JobStatus
	TranscriptionText string
}

type Provider interface {
	SubmitTranscriptionJob(url string) (*Job, error)
	LookupTranscriptionJob(id string) (*Job, error)
}
