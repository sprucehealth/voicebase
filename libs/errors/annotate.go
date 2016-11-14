package errors

import "fmt"

// Wrap adds context to an error. It can be used to attach more information that is useful for debugging.
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	e := wrap(trace(err, 1))
	e.annotations = append(e.annotations, msg)
	return e
}

// Wrapf adds context to an error. It can be used to attach more information that is useful for debugging.
func Wrapf(err error, f string, v ...interface{}) error {
	if err == nil {
		return nil
	}
	e := wrap(trace(err, 1))
	e.annotations = append(e.annotations, fmt.Sprintf(f, v...))
	return e
}
