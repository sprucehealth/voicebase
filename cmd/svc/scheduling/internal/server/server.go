package server

import (
	"github.com/sprucehealth/backend/cmd/svc/scheduling/internal/dal"
	"github.com/sprucehealth/backend/svc/scheduling"
)

type server struct {
	dal dal.DAL
}

// New returns an initialized instance of server after performing initial validation
func New(dl dal.DAL) (scheduling.SchedulingServer, error) {
	return &server{}, nil
}
