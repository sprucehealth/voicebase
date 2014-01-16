package golog

import (
	"fmt"
	"io"
	"os"
	"time"
)

type IoOutput struct {
	Stdout, Stderr io.Writer
}

var DefaultOutput = &IoOutput{Stdout: os.Stdout, Stderr: os.Stderr}

func (o *IoOutput) Log(logType string, l Level, msg []byte) error {
	if l <= WARN {
		_, err := fmt.Fprintf(o.Stderr, "%s %s %s: %s\n", time.Now().Format(time.RFC3339), logType, l.String(), string(msg))
		return err
	} else {
		_, err := fmt.Fprintf(o.Stdout, "%s %s %s: %s\n", time.Now().Format(time.RFC3339), logType, l.String(), string(msg))
		return err
	}
}
