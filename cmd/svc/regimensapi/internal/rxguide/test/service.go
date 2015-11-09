package test

// TODO: Where is the right place to keep these mocks for backend components?

import (
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

// RXGuideService mocks out the functionality of the rx guide service client for use in tests
type RXGuideService struct {
	*mock.Expector
	PutRXGuideErrs      []error
	QueryRXGuidesOutput []map[string]*responses.RXGuide
	QueryRXGuidesErrs   []error
	RXGuideOutput       []*responses.RXGuide
	RXGuideErrs         []error
}

// PutRXGuide is a mocked implementation that returns the queued data
func (d *RXGuideService) PutRXGuide(r *responses.RXGuide) error {
	defer d.Record(r)
	var err error
	d.PutRXGuideErrs, err = mock.NextError(d.PutRXGuideErrs)
	return err
}

// QueryRXGuides is a mocked implementation that returns the queued data
func (d *RXGuideService) QueryRXGuides(prefix string, limit int) (map[string]*responses.RXGuide, error) {
	defer d.Record(prefix, limit)
	out := d.QueryRXGuidesOutput[0]
	d.QueryRXGuidesOutput = d.QueryRXGuidesOutput[1:]

	var err error
	d.QueryRXGuidesErrs, err = mock.NextError(d.QueryRXGuidesErrs)
	return out, err
}

// RXGuide is a mocked implementation that returns the queued data
func (d *RXGuideService) RXGuide(id string) (*responses.RXGuide, error) {
	defer d.Record(id)
	out := d.RXGuideOutput[0]
	d.RXGuideOutput = d.RXGuideOutput[1:]

	var err error
	d.RXGuideErrs, err = mock.NextError(d.RXGuideErrs)
	return out, err
}
