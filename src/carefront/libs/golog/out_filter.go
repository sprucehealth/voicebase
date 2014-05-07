package golog

type OutputFunc func(logType string, l Level, msg []byte) error

func (o OutputFunc) Log(logType string, l Level, msg []byte) error {
	return o(logType, l, msg)
}
