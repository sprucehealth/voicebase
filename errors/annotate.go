package errors

import "fmt"

// Annotate adds context to an error. It can be used to attach more information that is useful for debugging.
func Annotate(err error, msg string) error {
	if err == nil {
		return nil
	}
	e := wrap(err)
	e.annotations = append(e.annotations, msg)
	return e
}

// Annotatef adds context to an error. It can be used to attach more information that is useful for debugging.
func Annotatef(err error, f string, v ...interface{}) error {
	if err == nil {
		return nil
	}
	e := wrap(err)
	e.annotations = append(e.annotations, fmt.Sprintf(f, v...))
	return e
}

// Annotations returns all annotations attached to an error.
func Annotations(err error) []string {
	if e, ok := err.(aerr); ok {
		return e.annotations
	}
	return nil
}
