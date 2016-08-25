package hint

import (
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
