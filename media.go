package voicebase

import (
	"bytes"
	"context"
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
	Transcript  *transcript            `json:"transcript"`
}

func (m *Media) TranscriptionText() string {
	latestTranscription := m.Transcripts["latest"]
	if latestTranscription == nil {
		if m.Transcript == nil {
			return ""
		}
		latestTranscription = m.Transcript
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

// voicemailOptimizedConfiguration defines a configuration
// as recommended by voicebase optimized for a fast transcription
// turnaround time.
// See: https://voicebase.readthedocs.io/en/v3/how-to-guides/voicemail.html
var voicemailOptimizedConfiguration = &Configuration{
	Priority: PriorityHigh,
	Transcript: &TranscriptConfiguration{
		Formatting: &TranscriptFormattingConfiguration{
			EnableNumberFormatting: true,
		},
	},
	Knowledge: &KnowledgeConfiguration{
		EnableDiscovery: false,
	},
	SpeecModel: &SpeechModelConfiguration{
		Extensions: []string{"voicemail"},
	},
}

func (c *Client) UploadMedia(ctx context.Context, url string) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("mediaUrl", url)

	configurationData, err := json.Marshal(voicemailOptimizedConfiguration)
	if err != nil {
		return "", err
	}

	writer.WriteField("configuration", string(configurationData))

	if err := writer.Close(); err != nil {
		return "", err
	}

	var media Media
	if err := c.callMultipart(ctx, "POST", "media", writer.Boundary(), body, &media); err != nil {
		return "", err
	}

	return media.ID, nil
}

func (c *Client) GetMedia(ctx context.Context, id string) (*Media, error) {
	var media Media
	if err := c.call(ctx, "GET", "media/"+id, &media); err != nil {
		return nil, err
	}
	return &media, nil
}

func (c *Client) DeleteMedia(ctx context.Context, id string) error {
	return c.call(ctx, "DELETE", "media/"+id, nil)
}
