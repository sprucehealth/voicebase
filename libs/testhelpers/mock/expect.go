package mock

import (
	"log"
	"reflect"
	"runtime"
	"testing"
)

// Expectation represents an expectation that maps to a method name and list of parameters
type expectation struct {
	Func   *runtime.Func
	Params []interface{}
}

// NewExpectation returns an initialized instance of Expectation. This is sugar.
func NewExpectation(f interface{}, params ...interface{}) *expectation {
	return &expectation{
		Func:   runtime.FuncForPC(reflect.ValueOf(f).Pointer()),
		Params: params,
	}
}

type expectationSource struct {
	File string
	Line int
}

// Expector is to be used in composit mock structs for expectation setting
type Expector struct {
	T                  *testing.T
	Debug              bool
	expects            []*expectation
	expectationSources []*expectationSource
}

// Expect sets an in order expectation for this struct
func (e *Expector) Expect(exp *expectation) {
	_, file, line, _ := runtime.Caller(1)
	e.expects = append(e.expects, exp)
	e.expectationSources = append(e.expectationSources, &expectationSource{File: file, Line: line})
}

// Record uses the callers information to validate the call against the expected results
func (e *Expector) Record(params ...interface{}) {
	if e == nil {
		return
	}

	pc, file, line, _ := runtime.Caller(1)
	caller := runtime.FuncForPC(pc)
	if len(e.expects) == 0 {
		e.T.Fatalf("Recieved call to %s without any remaining expectiations: params %+v", caller.Name(), params)
	}
	// Grab out next expectation and then pop it off the list
	expect := e.expects[0]
	actual := &expectation{Func: caller, Params: params}
	e.expects = e.expects[1:]
	if !reflect.DeepEqual(expect, actual) {
		source := e.expectationSources[0]
		e.expectationSources = e.expectationSources[1:]
		e.T.Fatalf(
			"\nFailed Expectation:\n"+
				"File: %s\n"+
				"Line: %d\n"+
				"Expected:\n"+
				"  Name: %s\n"+
				"  Params: %+v\n"+
				"Got:\n"+
				"  Name: %s\n"+
				"  Params: %+v\n\n"+
				"Expectation Source:\n"+
				"File: %s\n"+
				"Line: %d\n", file, line,
			expect.Func.Name(), expect.Params,
			actual.Func.Name(), actual.Params,
			source.File, source.Line)
	}
	if e.Debug {
		log.Printf("Completed recording and validation of:\nFunction: %s\nParams:%+v", actual.Func.Name(), actual.Params)
	}
}

// Finisher is an interface for anything with a Finish method
type Finisher interface {
	Finish()
}

// Finish asserts that all expectations were met
func (e *Expector) Finish() {
	for _, ex := range e.expects {
		e.T.Fatalf("All expectations were not met. Next expectation - Name: %s, Params: %+v", ex.Func.Name(), ex.Params)
	}
}

// FinishAll is just a convenience method for finishing groups of mocks
func FinishAll(mocks ...Finisher) {
	for _, m := range mocks {
		m.Finish()
	}
}
