package voicebase

type Priority string

const (
	// PriorityHigh moves the job to the front of the job queue for a premium.
	PriorityHigh Priority = "high"
	// PriorityLow moves jobs to the back of the job queue and allows for a discount to be offered.
	PriorityLow Priority = "low"
	// PriorityNormal is the default priority for a job.
	PriorityNormal Priority = "normal"
)

type TranscriptFormattingConfiguration struct {
	EnableNumberFormatting bool `json:"enableNumberFormatting"`
}
type TranscriptConfiguration struct {
	Formatting *TranscriptFormattingConfiguration `json:"formatting"`
}

type KnowledgeConfiguration struct {
	EnableDiscovery bool `json:"enableDiscovery"`
}

type SpeechModelConfiguration struct {
	Extensions []string `json:"extensions"`
}

// Configuration provides a way to configure how to transcribe a media object at
// the time of upload.
type Configuration struct {
	Priority   Priority                  `json:"priority"`
	Transcript *TranscriptConfiguration  `json:"transcript"`
	Knowledge  *KnowledgeConfiguration   `json:"knowledge"`
	SpeecModel *SpeechModelConfiguration `json:"speechModel"`
}
