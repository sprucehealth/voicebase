package manager

// New returns an instance of a visit manager that can be used by a mobile
// client to integrate the Spruce Derm implementation of the Manager interface.
func New() VisitManager {
	return &visitManager{}
}
