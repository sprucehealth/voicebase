package harness

import (
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/blackbox/internal/dal"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// RegistrationConfig represents the configuration to use when registering test suites
type RegistrationConfig struct {
	SuitesToRegister map[string]struct{}
	TestsToRegister  map[string]struct{}
}

// ExecutionConfig represents the configuration to use when beginning test execution
type ExecutionConfig struct {
	SuiteStagger    time.Duration
	SuiteRepeat     time.Duration
	TestStagger     time.Duration
	MaxTestParallel int
	RunOnce         bool
}

type suiteContext struct {
	suite        reflect.Value
	payload      interface{}
	testContexts []*testContext
}

type testContext struct {
	testName string
	test     reflect.Method
}

var (
	rwMutex    = sync.RWMutex{}
	testSuites = make(map[string]*suiteContext)
)

// TestSuite defined the interface required by structs that want to be used as a test suite
type TestSuite interface {
	GeneratePayload() interface{}
	SuiteName() string
}

const bbTestPrefix = "BBTest"

// DAL represents the methods required to provide data access layer functionality
type DAL interface {
	InsertSuiteRun(model *dal.SuiteRun) (dal.SuiteRunID, error)
	SuiteRun(id dal.SuiteRunID) (*dal.SuiteRun, error)
	IncrementSuiteRunTestPassed(id dal.SuiteRunID) (int64, error)
	IncrementSuiteRunTestFailed(id dal.SuiteRunID) (int64, error)
	UpdateSuiteRun(id dal.SuiteRunID, update *dal.SuiteRunUpdate) (int64, error)
	DeleteSuiteRun(id dal.SuiteRunID) (int64, error)
	InsertSuiteTestRun(model *dal.SuiteTestRun) (dal.SuiteTestRunID, error)
	SuiteTestRun(id dal.SuiteTestRunID) (*dal.SuiteTestRun, error)
	UpdateSuiteTestRun(id dal.SuiteTestRunID, update *dal.SuiteTestRunUpdate) (int64, error)
	DeleteSuiteTestRun(id dal.SuiteTestRunID) (int64, error)
	InsertProfile(model *dal.Profile) (dal.ProfileID, error)
	Profile(id dal.ProfileID) (*dal.Profile, error)
	DeleteProfile(id dal.ProfileID) (int64, error)
	Transact(trans func(dal dal.DAL) error) (err error)
}

var (
	dl         dal.DAL
	dalSetLock = sync.Mutex{}
)

// SetDAL configures the system to record resultant information in the blackbox data store
func SetDAL(dal DAL) {
	dalSetLock.Lock()
	if dl == nil {
		dl = dal
	} else {
		golog.Errorf("DAL has already been set for this configuration of blackbox. It cannot be set twice.")
	}
	dalSetLock.Unlock()
}

// Register configures a test suite to run with the provided settings
func Register(testSuite TestSuite, config *RegistrationConfig) {
	suiteName := testSuite.SuiteName()
	if len(config.SuitesToRegister) != 0 {
		if _, ok := config.SuitesToRegister[suiteName]; !ok {
			golog.Debugf("Ignoring suite %s", suiteName)
			return
		}
	}
	testT := reflect.TypeOf(testSuite)
	testV := reflect.ValueOf(testSuite)
	golog.Debugf("Registering test suite %s with %d methods", suiteName, testT.NumMethod())
	payload := testSuite.GeneratePayload()
	golog.Debugf("Suite %s generated payload %+v", suiteName, payload)
	for i := 0; i < testT.NumMethod(); i++ {
		method := testT.Method(i)
		golog.Debugf("Inspecting method %s", method.Name)
		if len(config.TestsToRegister) != 0 {
			if _, ok := config.TestsToRegister[method.Name]; !ok {
				golog.Debugf("Ignoring test %s", method.Name)
				continue
			}
		}
		// Test methods will begin with the prefix and match the sig (interface{}) error
		if strings.HasPrefix(method.Name, bbTestPrefix) &&
			method.Type.NumIn() == 2 &&
			method.Type.In(1).Kind() == reflect.Interface &&
			method.Type.NumOut() == 0 {
			golog.Debugf("Registering test %s:%s", suiteName, method.Name)
			func() {
				defer rwMutex.Unlock()
				rwMutex.Lock()
				if suite, ok := testSuites[suiteName]; ok {
					suite.testContexts = append(suite.testContexts, &testContext{testName: method.Name, test: method})
				} else {
					testSuites[suiteName] = &suiteContext{
						suite:   testV,
						payload: payload,
						testContexts: []*testContext{
							&testContext{
								testName: method.Name,
								test:     method,
							},
						},
					}
				}
			}()
		} else {
			golog.Debugf("Ignoring method %s on test struct - NumIn: %d, NumOut: %d", method.Name, method.Type.NumIn(), method.Type.NumOut())
		}
	}
}

// SuiteRunReport represents the result of a suite run
type SuiteRunReport struct {
	SuiteName   string
	SuiteRunID  dal.SuiteRunID
	Start       time.Time
	Finish      time.Time
	TestsPassed int
	TestsFailed int
	TestReports map[string]*SuiteTestRunReport
	processed   bool
}

// ProcessTestResults inspects the contained test results for top level totals
func (r *SuiteRunReport) ProcessTestResults() {
	for _, v := range r.TestReports {
		if v.Status != dal.SuiteTestRunStatusPassed {
			r.TestsFailed++
		} else {
			r.TestsPassed++
		}
	}
	r.processed = true
}

func (r *SuiteRunReport) String() string {
	if !r.processed {
		r.ProcessTestResults()
	}
	return fmt.Sprintf(
		"%s Blackbox Suite Report:\n"+
			"Start: %v\n"+
			"Finish: %v\n"+
			"Elapsed: %v\n"+
			"Passed: %d\n"+
			"Failed: %d", r.SuiteName, r.Start, r.Finish, r.Finish.Sub(r.Start), r.TestsPassed, r.TestsFailed)
}

var (
	reportProcessors     [](func(r *SuiteRunReport) error)
	reportProcessorsLock = sync.RWMutex{}
)

// RegisterReportProcessor registers the provided function into the system to process reports
func RegisterReportProcessor(p func(r *SuiteRunReport) error) {
	reportProcessorsLock.Lock()
	defer reportProcessorsLock.Unlock()
	reportProcessors = append(reportProcessors, p)
}

func processReport(r *SuiteRunReport) {
	reportProcessorsLock.RLock()
	defer reportProcessorsLock.RUnlock()
	for _, p := range reportProcessors {
		if err := p(r); err != nil {
			golog.Errorf("Error while processing report %s: %s", r, err)
		}
	}
}

// SuiteTestRunReport represents the result of a test run
type SuiteTestRunReport struct {
	TestName string
	Status   dal.SuiteTestRunStatus
	Message  string
	Start    time.Time
	Finish   time.Time
}

// NewTicker wraps time.NewTicker and translates 0 duration into 1 nanosecond
func NewTicker(d time.Duration) *time.Ticker {
	if d.Nanoseconds() == 0 {
		d = time.Nanosecond * 1
	}
	return time.NewTicker(d)
}

// Execute runs the currently registered test suites
// TODO: In a world where suites run longer than the repeat time we will need to fail the later run and let the initial once finish.
func Execute(config *ExecutionConfig) {
	if config.MaxTestParallel <= 0 {
		config.MaxTestParallel = 1
	}
	golog.Debugf("Executing with config %+v", config)
	suiteRepeatTicker := NewTicker(config.SuiteRepeat)
	suiteStaggerTicker := NewTicker(config.SuiteStagger)
	parallel := conc.NewParallel()
	for {
		for suiteName, ctx := range testSuites {
			gctx := ctx
			parallel.Go(func() error {
				// TODO: Need to seperate the recording from the execution in a convenient DAL check mechanism
				var err error
				var suiteRunID dal.SuiteRunID
				startTime := time.Now()
				if dl != nil {
					suiteRunID, err = dl.InsertSuiteRun(&dal.SuiteRun{
						SuiteName: suiteName,
						Status:    dal.SuiteRunStatusRunning,
						Start:     startTime,
					})
					if err != nil {
						golog.Errorf("Error while inserting new suite run: %s", err.Error())
					}
				}

				golog.Infof("Starting test suite: %s", suiteName)
				golog.Debugf("Starting test suite: %s with context %+v", suiteName, ctx)
				report := &SuiteRunReport{
					SuiteName:   suiteName,
					SuiteRunID:  suiteRunID,
					Start:       startTime,
					TestReports: executeTestSuite(gctx, config, suiteRunID),
				}
				report.Finish = time.Now()

				if dl != nil && suiteRunID.IsValid {
					status := dal.SuiteRunStatusComplete
					_, err = dl.UpdateSuiteRun(suiteRunID, &dal.SuiteRunUpdate{
						Status: &status,
						Finish: ptr.Time(report.Finish),
					})
					if err != nil {
						golog.Errorf("Error while inserting new suite run: %s", err.Error())
					}
				}
				golog.Debugf("\n" + report.String())
				processReport(report)
				return nil
			})
			// For every full run stagger the suite execution
			<-suiteStaggerTicker.C
		}
		if err := parallel.Wait(); err != nil {
			golog.Errorf("Error from suite execution: %s", err.Error())
		}
		if config.RunOnce {
			break
		}
		// Execute every suite on the repeat schedule
		<-suiteRepeatTicker.C
	}
}

// TODO: There is likely a more clever way using channels rather than locking this count comparison
var testsExecuting int
var testExecutionComparisonMutex = sync.Mutex{}

// Note: As stated above there is likely a more clever way to do this. The below multithread locking system needs analysis
func executeTestSuite(ctx *suiteContext, config *ExecutionConfig, suiteRunID dal.SuiteRunID) map[string]*SuiteTestRunReport {
	testStaggerTicker := NewTicker(config.TestStagger)
	parallel := conc.NewParallel()
	resultsChan := make(chan *SuiteTestRunReport, len(ctx.testContexts))
	results := make(map[string]*SuiteTestRunReport)
	for i := 0; i < len(ctx.testContexts); {
		gtctx := ctx.testContexts[i]
		testExecutionComparisonMutex.Lock()
		func() {
			defer testExecutionComparisonMutex.Unlock()
			if testsExecuting < config.MaxTestParallel {
				testsExecuting++
				golog.Debugf("Starting exection of %s. %d/%d of parallel capacity in use", gtctx.testName, testsExecuting, config.MaxTestParallel)
				parallel.Go(func() error {
					result := executeTest(gtctx, ctx.payload, ctx.suite, suiteRunID)
					testExecutionComparisonMutex.Lock()
					func() {
						defer testExecutionComparisonMutex.Unlock()
						testsExecuting--
					}()
					resultsChan <- result
					return nil
				})
				i++
			} else {
				delay := config.TestStagger
				if delay.Nanoseconds() == 0 {
					delay = 2 * time.Second
				}
				golog.Warningf("Currently at maximum parallel test count %d/%d. Retrying %s in %v", testsExecuting, config.MaxTestParallel, gtctx.testName, config.TestStagger)
			}
		}()
		<-testStaggerTicker.C
	}
	if err := parallel.Wait(); err != nil {
		golog.Errorf("Error from test execution: %s", err.Error())
	}
	// TODO: Optimize this to empty the channel as results are created rather than waiting till the end
	for len(results) < len(ctx.testContexts) {
		select {
		case result, ok := <-resultsChan:
			if !ok {
				golog.Errorf("Unexpected channel closure when collecting results")
				break
			}
			results[result.TestName] = result
		}
	}
	return results
}

func executeTest(ctx *testContext, payload interface{}, suite reflect.Value, suiteRunID dal.SuiteRunID) (report *SuiteTestRunReport) {
	golog.Debugf("Starting test: %s with context %+v, payload: %+v", ctx.testName, ctx, payload)
	var err error
	var suiteTestRunID dal.SuiteTestRunID
	startTime := time.Now()
	defer func() {
		finishTime := time.Now()
		var msg *string
		var passed bool
		var status dal.SuiteTestRunStatus
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				golog.Errorf("%s - FAIL \n%+v", ctx.testName, r)
				msg = ptr.String(fmt.Sprintf("%+v", r))
			} else {
				golog.Errorf("%s - FAIL \n%s", ctx.testName, err.Error())
				msg = ptr.String(fmt.Sprintf(err.Error()))
			}
			status = dal.SuiteTestRunStatusFailed
		} else {
			passed = true
			golog.Infof("%s - PASS", ctx.testName)
			status = dal.SuiteTestRunStatusPassed
		}
		if dl != nil && suiteRunID.IsValid {
			if _, err = dl.UpdateSuiteTestRun(suiteTestRunID, &dal.SuiteTestRunUpdate{
				Message: msg,
				Status:  &status,
				Finish:  ptr.Time(finishTime),
			}); err != nil {
				golog.Errorf("Error while inserting new suite test run: %s", err.Error())
			}
			if passed {
				if _, err := dl.IncrementSuiteRunTestPassed(suiteRunID); err != nil {
					golog.Errorf("Error while incrementing test pass counter for run %s: %s", suiteRunID, err)
				}
			} else {
				if _, err := dl.IncrementSuiteRunTestFailed(suiteRunID); err != nil {
					golog.Errorf("Error while incrementing test fail counter for run %s: %s", suiteRunID, err)
				}
			}
		}
		var rMsg string
		if msg != nil {
			rMsg = *msg
		}
		report = &SuiteTestRunReport{
			TestName: ctx.testName,
			Status:   status,
			Message:  rMsg,
			Start:    startTime,
			Finish:   finishTime,
		}
	}()
	if dl != nil && suiteRunID.IsValid {
		suiteTestRunID, err = dl.InsertSuiteTestRun(&dal.SuiteTestRun{
			SuiteRunID: suiteRunID,
			TestName:   ctx.testName,
			Status:     dal.SuiteTestRunStatusRunning,
			Start:      startTime,
		})
		if err != nil {
			golog.Errorf("Error while inserting new suite test run: %s", err.Error())
		}
	}
	golog.Debugf("Starting %s with payload %+v", ctx.testName, payload)
	ctx.test.Func.Call([]reflect.Value{suite, reflect.ValueOf(payload)})
	return nil
}

func failf(file string, line int, fmtMessage string, fmtArgs ...interface{}) {
	panic(fmt.Errorf("%s\n%s:%d", fmt.Sprintf(fmtMessage, fmtArgs...), file, line))
}

// Failf is intended to be used as the failure mechanism for tests in the blackbox harness
func Failf(fmtMessage string, fmtArgs ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	failf(file, line, fmtMessage, fmtArgs...)
}

// FailErr is intended to be used as the failure mechanism for tests in the blackbox harness
func FailErr(err error, context ...interface{}) {
	if err == nil {
		return
	}
	_, file, line, _ := runtime.Caller(1)
	failf(file, line, withContext(err.Error(), context...))
}

// AssertEqual asserts the equality of the two values and
func AssertEqual(expected, actual interface{}, context ...interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		_, file, line, _ := runtime.Caller(1)
		failf(file, line, withContext(fmt.Sprintf("Equality assertion failed\nExpected: %+v\nActual: %+v", expected, actual), context...))
	}
}

// AssertNotNil asserts that the provided value is non nil
func AssertNotNil(actual interface{}, context ...interface{}) {
	if actual == nil {
		_, file, line, _ := runtime.Caller(1)
		failf(file, line, withContext(fmt.Sprintf("Expected a non nil value but got %+v", actual), context...))
	}
}

// AssertNil asserts that the provided value is nil
func AssertNil(actual interface{}, context ...interface{}) {
	av := reflect.ValueOf(actual)
	if !av.IsNil() {
		_, file, line, _ := runtime.Caller(1)
		failf(file, line, withContext(fmt.Sprintf("Expected a nil value but got %+v", actual), context...))
	}
}

// Assert asserts that the provided value is true
func Assert(actual bool, context ...interface{}) {
	if !actual {
		_, file, line, _ := runtime.Caller(1)
		failf(file, line, withContext("Assertion failure", context...))
	}
}

// withContext is a helper for wrapping format strings in context
func withContext(s string, context ...interface{}) string {
	if len(context) > 0 {
		s = s + "\n" + fmt.Sprintf("Context:"+strings.Repeat(" %+v", len(context)), context...)
	}
	return s
}

// RandInt64 - * Cough cough best effort uniqueness *
// https://golang.org/pkg/math/rand/
func RandInt64() int64 {
	r := time.Now().Unix()
	// Utilize an increasing timestamp suffixed by a random 5 digit number
	r = (r * 100000) + (rand.Int63n(90000) + 10000)
	return r
}

// RandInt64N just wraps Int64n
func RandInt64N(n int64) int64 {
	return rand.Int63n(n)
}

var maxIntLength int64 = 18

// RandInt64NDigits returns a random number N digits in length
func RandInt64NDigits(n int64) int64 {
	if n > maxIntLength {
		n = maxIntLength
	}

	var base int64 = 1
	for i := base; i < int64(n); i++ {
		base = base * 10
	}
	return base + rand.Int63n((base*10)-1)
}

// RandNumString - * Cough cough best effort uniqueness *
func RandNumString() string {
	return strconv.FormatInt(RandInt64(), 10)
}

var letters = []rune("abcdefghijklmnopqrstuvwxyz1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ`~!@#$%^&*()-_=+[{}];:<,'>.?/\\|")

// RandString generates a random string of length n
func RandString(n int64) string {
	r := make([]rune, n)
	for i := range r {
		r[i] = letters[rand.Intn(len(letters))]
	}
	return string(r)
}

// RandLengthString returns a random string of a random non empty length
func RandLengthString(n int64) string {
	r := RandInt64N(n)
	if r == 0 {
		r = 1
	}
	return RandString(r)
}

// RandPhoneNumber generates a random 10 digit number
func RandPhoneNumber() string {
	return strconv.FormatInt(rand.Int63n(9000000000)+1000000000, 10)
}

// RandEmail generates a random valid email address
func RandEmail() string {
	return fmt.Sprintf("e" + RandNumString() + "@randommail.com")
}

// RandBool returns a random boolean value
func RandBool() bool {
	return (rand.Int() % 2) == 0
}

// NanosecondsToMillisecondsScalar represents a scalar value to be used in multiplication for conversions
const NanosecondsToMillisecondsScalar = 1.0 / 1000000.0

// Profile tracks the execution time of the provided function for profiling purposes
func Profile(key string, f func()) {
	start := time.Now().UnixNano()
	f()
	finish := time.Now().UnixNano()
	result := float64(finish-start) * NanosecondsToMillisecondsScalar
	_, file, line, _ := runtime.Caller(1)
	golog.Debugf("Profile: %s - %f ms - %s:%d", key, result, file, line)
	if dl != nil {
		profile := &dal.Profile{
			ProfileKey: key,
			ResultMS:   result,
		}
		if _, err := dl.InsertProfile(profile); err != nil {
			golog.Errorf("Error while inserting new profile record %+v: %s", profile, err.Error())
		}
	}
}

var (
	globalTestConfigs = make(map[string]string)
	configMutex       = sync.RWMutex{}
)

// GetConfig returns the config scoped to the suite and config name provided
func GetConfig(name string) string {
	defer configMutex.RUnlock()
	configMutex.RLock()
	return globalTestConfigs[name]
}

// SetConfig adds the config value to the global set
func SetConfig(name, value string) {
	defer configMutex.Unlock()
	configMutex.Lock()
	globalTestConfigs[name] = value
}
