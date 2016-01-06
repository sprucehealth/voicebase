package mock

import (
	"log"
	"reflect"
	"runtime"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/sprucehealth/backend/libs/golog"
)

// Expectation represents an expectation that maps to a method name and list of parameters
type expectation struct {
	Func              *runtime.Func
	Params            []interface{}
	ExactParamMatch   bool
	FnParams          []interface{}
	ParamValidationFn func(params ...interface{})
	Returns           []interface{}
	PostFn            func()
}

// NewExpectation returns an initialized instance of Expectation. This is sugar.
func NewExpectation(f interface{}, params ...interface{}) *expectation {
	return &expectation{
		Func:            runtime.FuncForPC(reflect.ValueOf(f).Pointer()),
		Params:          params,
		ExactParamMatch: true,
	}
}

// NewExpectationFn returns an initialized instance of Expectation set to a custom validation function. This is sugar.
func NewExpectationFn(f interface{}, fn func(params ...interface{})) *expectation {
	return &expectation{
		Func:              runtime.FuncForPC(reflect.ValueOf(f).Pointer()),
		ParamValidationFn: fn,
	}
}

// WithReturns is sugar to wrap expectations in returns
func WithReturns(e *expectation, returns ...interface{}) *expectation {
	e.Returns = returns
	return e
}

// WithReturns is sugar to wrap expectations with returns
func (e *expectation) WithReturns(returns ...interface{}) *expectation {
	e.Returns = returns
	return e
}

// WithPostFn adds a function to execute after this expectation is met
func WithPostFn(e *expectation, f func()) *expectation {
	e.PostFn = f
	return e
}

// WithPostFn adds a function to execute after this expectation is met
func (e *expectation) WithPostFn(f func()) *expectation {
	e.PostFn = f
	return e
}

// WithParamValidationFn is sugar to wrap expectations with validation functions
func WithParamValidationFn(e *expectation, fn func(params ...interface{})) *expectation {
	e.ParamValidationFn = fn
	return e
}

// WithParamValidationFn is sugar to wrap expectations with validation functions
func (e *expectation) WithParamValidationFn(fn func(params ...interface{})) *expectation {
	e.ParamValidationFn = fn
	return e
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
	callCounts         map[string]int
}

// Expect sets an in order expectation for this struct
func (e *Expector) Expect(exp *expectation) {
	_, file, line, _ := runtime.Caller(1)
	e.expects = append(e.expects, exp)
	e.expectationSources = append(e.expectationSources, &expectationSource{File: file, Line: line})
}

// callIndex returns the call count of the calling function - 1. Call counts are incremented using the Record method
func (e *Expector) callIndex() int {
	if e.callCounts == nil {
		e.callCounts = make(map[string]int)
	}
	pc, _, _, _ := runtime.Caller(1)
	caller := runtime.FuncForPC(pc)
	return e.callCounts[caller.Name()] - 1
}

// Record uses the callers information to validate the call against the expected results
func (e *Expector) Record(params ...interface{}) []interface{} {
	if e == nil {
		return nil
	} else if e.T == nil {
		golog.Fatalf("Calling Record on an expector with an uninitialized *testing.T is not allowed.")
	}

	pc, file, line, _ := runtime.Caller(1)
	caller := runtime.FuncForPC(pc)
	if len(e.expects) == 0 {
		e.T.Fatalf(
			"Recieved call to %s without any remaining expectiations: params %+v\n"+
				"Source:\n"+
				"File: %s\n"+
				"Line: %d\n", caller.Name(), params, file, line)
	}
	// increment our call count
	if e.callCounts == nil {
		e.callCounts = make(map[string]int)
	}
	e.callCounts[caller.Name()]++

	// Grab out next expectation and then pop it off the list
	expectWithReturns := e.expects[0]
	actual := &expectation{Func: caller, Params: params}
	source := e.expectationSources[0]
	if expectWithReturns.ExactParamMatch {
		expectWithoutReturns := &expectation{Func: expectWithReturns.Func, Params: expectWithReturns.Params}
		if !reflect.DeepEqual(expectWithoutReturns, actual) {
			e.T.Fatalf(
				"\nFailed Expectation:\n"+
					"File: %s\n"+
					"Line: %d\n"+
					"Expected:\n"+
					"  Name: %s\n"+
					"  Params: %s\n"+
					"Got:\n"+
					"  Name: %s\n"+
					"  Params: %s\n\n"+
					"Expectation Source:\n"+
					"File: %s\n"+
					"Line: %d\n", file, line,
				expectWithoutReturns.Func.Name(), spew.Sdump(expectWithoutReturns.Params),
				actual.Func.Name(), spew.Sdump(actual.Params),
				source.File, source.Line)
		}
	} else if !reflect.DeepEqual(expectWithReturns.Func, actual.Func) {
		e.T.Fatalf(
			"\nFailed Expectation:\n"+
				"File: %s\n"+
				"Line: %d\n"+
				"Expected:\n"+
				"  Name: %s\n"+
				"Got:\n"+
				"  Name: %s\n"+
				"Expectation Source:\n"+
				"File: %s\n"+
				"Line: %d\n", file, line,
			expectWithReturns.Func.Name(),
			actual.Func.Name(),
			source.File, source.Line)
	}
	if expectWithReturns.ParamValidationFn != nil {
		expectWithReturns.ParamValidationFn(actual.Params...)
	}
	if e.Debug {
		log.Printf("Completed recording and validation of:\nFunction: %s\nParams:%+v", actual.Func.Name(), actual.Params)
	}

	if expectWithReturns.PostFn != nil {
		expectWithReturns.PostFn()
	}

	e.expectationSources = e.expectationSources[1:]
	e.expects = e.expects[1:]
	return expectWithReturns.Returns
}

// Finisher is an interface for anything with a Finish method
type Finisher interface {
	Finish()
}

// Finish asserts that all expectations were met
func (e *Expector) Finish() {
	if e == nil {
		return
	}
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

// SafeError uses reflection to safely return an error from an interface
func SafeError(e interface{}) error {
	if err, ok := e.(error); ok {
		return err
	}
	return nil
}
