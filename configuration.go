package voicebase

type KeywordConfiguration struct {
	Semantic bool `json:"semantic"`
}

type TopicsConfiguration struct {
	Semantic bool `json:"semantic"`
}

type Priority string

const (
	// PriorityHigh moves the job to the front of the job queue for a premium.
	PriorityHigh Priority = "high"
	// PriorityLow moves jobs to the back of the job queue and allows for a discount to be offered.
	PriorityLow Priority = "low"
	// PriorityNormal is the default priority for a job.
	PriorityNormal Priority = "normal"
)

type IngestConfiguration struct {
	Priority Priority `json:"priority"`
}

type TranscriptionConfiguration struct {
	FormatNumbers []string `json:"formatNumbers"`
}

// Configuration provides a way to configure how to transcribe a media object at
// the time of upload.
type Configuration struct {
	Executor   string                      `json:"executor"`
	Keywords   *KeywordConfiguration       `json:"keywords,omitempty"`
	Topics     *TopicsConfiguration        `json:"topics,omitempty"`
	Ingest     *IngestConfiguration        `json:"ingest"`
	Transcript *TranscriptionConfiguration `json:"transcripts"`
}

type ConfigurationContainer struct {
	Configuration *Configuration `json:"configuration"`
}

// voicemailOptimizedConfiguration defines a configuration
// as recommended by voicebase optimized for a fast transcription
// turnaround time.
var voicemailOptimizedConfiguration = &Configuration{
	Executor: "v2",
	Keywords: &KeywordConfiguration{
		Semantic: false,
	},
	Topics: &TopicsConfiguration{
		Semantic: false,
	},
	Ingest: &IngestConfiguration{
		Priority: PriorityHigh,
	},
	Transcript: &TranscriptionConfiguration{
		FormatNumbers: []string{"digits", "dashed"},
	},
}
