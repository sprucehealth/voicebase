package tools

import (
	"fmt"
	"runtime"
)

func Trace(prefix string) {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fmt.Printf("%s - %s:%d %s\n", prefix, file, line, f.Name())
}
