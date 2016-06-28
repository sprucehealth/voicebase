package manager

// NewVisitManager returns an instance of a visit manager that can be used by a mobile
// client to integrate the Spruce Derm implementation of the Manager interface.
func NewVisitManager() VisitManager {
	return &visitManager{}
}
