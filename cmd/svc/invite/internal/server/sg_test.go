package server

import (
	"testing"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

var _ SendGridClient = &sgMock{}

type sgMock struct {
	*mock.Expector
}

func newSGMock(t *testing.T) *sgMock {
	return &sgMock{&mock.Expector{T: t}}
}

func (m *sgMock) Send(sm *sendgrid.SGMail) error {
	r := m.Expector.Record(sm)
	return mock.SafeError(r[0])
}
