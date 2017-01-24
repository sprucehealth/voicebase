package hint

import (
	"errors"
	"time"
)

// Practitioner represents a provider registered in Hint.
type Practitioner struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
}

type PractitionerClient interface {
	List(practiceKey string) ([]*Practitioner, error)
}

type practitionerClient struct {
	B   Backend
	Key string
}

func NewPractitionerClient(backend Backend, key string) PractitionerClient {
	return &practitionerClient{
		B:   backend,
		Key: key,
	}
}

func (c practitionerClient) List(practiceKey string) ([]*Practitioner, error) {
	if practiceKey == "" {
		return nil, errors.New("practice_key required")
	}

	var practitioners []*Practitioner
	if _, err := c.B.Call("GET", "/provider/practitioners", practiceKey, nil, &practitioners); err != nil {
		return nil, err
	}

	return practitioners, nil
}
