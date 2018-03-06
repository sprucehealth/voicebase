package voicebase

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
)

type word struct {
	Position   int     `json:"p"`
	Confidence float64 `json:"c"`
	Word       string  `json:"w"`
	M          string  `json:"m"`
}

type transcript struct {
	ID    string  `json:"transcriptId"`
	Words []*word `json:"words"`
}

type Media struct {
	ID          string                 `json:"mediaId"`
	Status      string                 `json:"status"`
	Transcripts map[string]*transcript `json:"transcripts"`
}

func (m *Media) TranscriptionText() string {
	latestTranscription := m.Transcripts["latest"]
	if latestTranscription == nil {
		return ""
	}

	if len(latestTranscription.Words) == 0 {
		return ""
	}

	var transcriptionText string
	for _, w := range latestTranscription.Words {
		if w.M == "punc" {
			transcriptionText += w.Word
		} else {
			transcriptionText += " " + w.Word
		}

	}

	return transcriptionText[1:]
}

type mediaResponse struct {
	Media *Media `json:"media"`
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

type MediaClient interface {
	Upload(url string) (string, error)
	Get(id string) (*Media, error)
	Delete(id string) error
}

type mediaClient struct {
	b           Backend
	bearerToken string
}

func NewMediaClient(backend Backend, bearerToken string) MediaClient {
	return &mediaClient{
		b:           backend,
		bearerToken: bearerToken,
	}
}

func (m mediaClient) Upload(url string) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("media", url)

	configurationData, err := json.Marshal(&ConfigurationContainer{
		Configuration: voicemailOptimizedConfiguration,
	})
	if err != nil {
		return "", err
	}

	writer.WriteField("configuration", string(configurationData))

	if err := writer.Close(); err != nil {
		return "", err
	}

	var media Media
	if err := m.b.CallMultipart("POST", "media", m.bearerToken, writer.Boundary(), body, &media); err != nil {
		return "", err
	}

	return media.ID, nil
}

func (m mediaClient) Get(id string) (*Media, error) {
	var mediaResponse mediaResponse
	if err := m.b.Call("GET", "media/"+id, m.bearerToken, &mediaResponse); err != nil {
		return nil, err
	}

	return mediaResponse.Media, nil
}

func (m mediaClient) Delete(id string) error {
	return m.b.Call("DELETE", "media/"+id, m.bearerToken, nil)
}
