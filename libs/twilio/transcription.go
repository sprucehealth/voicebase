package twilio

type TranscriptionService struct {
	client *Client
}

type Transcriptioner interface {
	Delete(sid string) (*Response, error)
}

func (t *TranscriptionService) Delete(sid string) (*Response, error) {
	u := t.client.EndPoint("Transcriptions", sid)

	req, err := t.client.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := t.client.Do(req, nil)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
