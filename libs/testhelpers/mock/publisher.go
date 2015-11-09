package mock

// Publisher mocks out the implementation of the publishing aspects of the dispatch system
type Publisher struct {
	*Expector
	// Outputs should be set to stage return calls from the corresponding method
	PublishErrs []error
}

// Publish is a mocked implementation that returns the queued errors
func (d *Publisher) Publish(e interface{}) error {
	defer d.Record(e)
	var err error
	d.PublishErrs, err = NextError(d.PublishErrs)
	return err
}

// PublishAsync is a mocked implementation
func (d *Publisher) PublishAsync(e interface{}) {
	defer d.Record(e)
}
