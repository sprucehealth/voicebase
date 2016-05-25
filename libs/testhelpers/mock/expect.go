package mock

// TODO: mraines: This package/file needs a refactor and some tests since it's becoming core to our testing

import (
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/kr/pretty"
	"github.com/sprucehealth/backend/libs/golog"
)

// Expectation represents an expectation that maps to a method name and list of parameters
type expectation struct {
	Func              *runtime.Func
	Params            []interface{}
	FnParams          []interface{}
	ParamValidationFn func(params ...interface{})
	Returns           []interface{}
	PostFn            func()
}

// NewExpectation returns an initialized instance of Expectation. This is sugar.
func NewExpectation(f interface{}, params ...interface{}) *expectation {
	return &expectation{
		Func:   runtime.FuncForPC(reflect.ValueOf(f).Pointer()),
		Params: params,
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
	T testing.TB

	// Keep unordered expectations in a map to simplify the deletion process
	unorderedExpects   map[int]*expectation
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

// ExpectUnordered sets an expectation that just needs to occur.
// Note: For a struct all expectations set in ExpectNoOrder are checked first
//   this is suboptimal for expectations in  a test that occur in both an unordered and order fashion
// Note: NoOrder expectations do not track the source of the expectation since there can be overlap
func (e *Expector) ExpectUnordered(exp *expectation) {
	if exp.ParamValidationFn != nil {
		e.T.Fatalf("Currently you cannot set unordered expectation withparam validation fn. This is TODO:")
	}
	if e.unorderedExpects == nil {
		e.unorderedExpects = make(map[int]*expectation)
	}
	e.unorderedExpects[len(e.unorderedExpects)] = exp
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
	// increment our call count
	if e.callCounts == nil {
		e.callCounts = make(map[string]int)
	}
	e.callCounts[caller.Name()]++

	// Check our unordered expectations first
	for k, ex := range e.unorderedExpects {
		failOnFatal := false
		if e.checkExpectation(ex, nil, failOnFatal, caller, file, line, params) {
			// execute any post fn
			if ex.PostFn != nil {
				ex.PostFn()
			}
			// If we matched an unordered expectation then remove it
			delete(e.unorderedExpects, k)
			return ex.Returns
		}
	}

	if len(e.expects) == 0 {
		e.T.Fatalf(
			"Recieved call to %s without any remaining ordered expectiations: params %+v\n"+
				"Source: %s:%d\n", caller.Name(), params, file, line)
	}

	// If we didn't match any unordered expectations do an in order assertion
	es := e.expectationSources[0]
	ex := e.expects[0]

	// If the inorder expectation fails then fail immediately
	failOnFatal := true
	e.checkExpectation(ex, es, failOnFatal, caller, file, line, params)

	// execute any post fn
	if ex.PostFn != nil {
		ex.PostFn()
	}

	e.expectationSources = e.expectationSources[1:]
	e.expects = e.expects[1:]
	return ex.Returns
}

const failureFormatString = "\nFailed Expectation:\n" +
	"File: %s:%d\n" +
	"Expected:\n" +
	"  Name: %s\n" +
	"Got:\n" +
	"  Name: %s\n" +
	"Params Diff:\n  %s\n" +
	"Expectation Source:\n" +
	"File: %s:%d\n"

// checkExpectation examines the provided expectation and fails the test if it does not pass if failOnFail is true
//   if failOnFail is false then the failure status of the expectation is returned
func (e *Expector) checkExpectation(ex *expectation, es *expectationSource, failOnFail bool, caller *runtime.Func, file string, line int, params []interface{}) bool {
	// Remove any returns from the expectation for proper comparison
	exp := &expectation{Func: ex.Func, Params: ex.Params}
	actual := &expectation{Func: caller, Params: params}
	if ex.ParamValidationFn != nil {
		ex.ParamValidationFn(actual.Params...)
	} else if !reflect.DeepEqual(exp, actual) {
		if failOnFail {
			e.T.Fatalf(
				failureFormatString,
				file, line,
				exp.Func.Name(),
				actual.Func.Name(),
				strings.Join(pretty.Diff(actual.Params, exp.Params), "\n  "),
				es.File, es.Line)
		}
		return false
	}
	return true
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

	// Make sure we're not in a deferred finish and there was a panic. If there is, bubble it up.
	if r := recover(); r != nil {
		panic(r)
	}

	// Check for outstanding expectations
	for _, ex := range e.expects {
		e.T.Fatalf("All expectations were not met. Next expectation - Name: %s, Params: %+v", ex.Func.Name(), ex.Params)
	}

	// Check for outstanding unordered expectations
	for _, ex := range e.unorderedExpects {
		e.T.Fatalf("All unordered expectations were not met. Next unordered expectation - Name: %s, Params: %+v", ex.Func.Name(), ex.Params)
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
