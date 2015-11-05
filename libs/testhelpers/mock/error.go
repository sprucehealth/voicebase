package mock

// NextError is a convenience method that returns the next error in the list if one exists and pops it from the list.
// If one is not present it will default to nil and return the empty list
func NextError(errs []error) ([]error, error) {
	if len(errs) == 0 {
		return nil, nil
	}
	e := errs[0]
	return errs[1:], e
}
