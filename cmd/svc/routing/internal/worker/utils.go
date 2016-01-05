package worker

type Worker interface {
	Start()
	Started() bool
}
