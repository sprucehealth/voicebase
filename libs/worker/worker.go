package worker

// Worker represents the interface that mechanisms performing periodic background tasks should conform to
type Worker interface {
	Start()
	Started() bool
}