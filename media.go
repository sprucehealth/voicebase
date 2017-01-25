package voicebase

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"strings"
)

type Word struct {
	Position   int     `json:"p"`
	Confidence float64 `json:"c"`
	Word       string  `json:"w"`
}

type Transcript struct {
	ID    string  `json:"transcriptId"`
	Words []*Word `json:"words"`
}

type Media struct {
	ID          string                 `json:"mediaId"`
	Status      string                 `json:"status"`
	Transcripts map[string]*Transcript `json:"transcripts"`
}

func (m *Media) TranscriptionText() string {

	latestTranscription := m.Transcripts["latest"]
	if latestTranscription == nil {
		return ""
	}

	if len(latestTranscription.Words) == 0 {
		return ""
	}

	words := make([]string, len(latestTranscription.Words))
	for i, w := range latestTranscription.Words {
		words[i] = w.Word
	}

	return strings.Join(words[:len(words)-1], " ") + words[len(words)-1]
}

type MediaResponse struct {
	Media *Media `json:"media"`
}

type MediaClient interface {
	Upload(url string) (string, error)
	Get(id string) (*Media, error)
	Delete(id string) error
}

type mediaClient struct {
	b Backend
}

func NewMediaClient(backend Backend) MediaClient {
	return &mediaClient{
		b: backend,
	}
}

func (m mediaClient) Upload(url string) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("media", url)

	configurationData, err := json.Marshal(&ConfigurationContainer{
		Configuration: defaultConfiguration,
	})
	if err != nil {
		return "", err
	}

	writer.WriteField("configuration", string(configurationData))

	if err := writer.Close(); err != nil {
		return "", err
	}

	var media Media
	if err := m.b.CallMultipart("POST", "media", BearerToken, writer.Boundary(), body, &media); err != nil {
		return "", err
	}

	return media.ID, nil
}

func (m mediaClient) Get(id string) (*Media, error) {
	var mediaResponse MediaResponse
	if err := m.b.Call("GET", "media/"+id, BearerToken, &mediaResponse); err != nil {
		return nil, err
	}

	return mediaResponse.Media, nil
}

func (m mediaClient) Delete(id string) error {
	return m.b.Call("DELETE", "media/"+id, BearerToken, nil)
}
