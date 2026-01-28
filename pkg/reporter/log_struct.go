package reporter

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/go-playground/validator/v10"
)

var (
	ErrInvalidLogFormat = errors.New("invalid log format")
	ErrInvalidLogStream = errors.New("invalid log stream")
	ErrParsingLog       = errors.New("error parsing log")
)
const (
	Passed = Status("passed")
	Failed = Status("failed")
	Error  = Status("error")
)

// Duration is a custom type used to allow time.Duration to be unmarshaled from JSON strings
type Duration time.Duration


func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}
    
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	var err error
	var duration time.Duration
	switch value := v.(type) {
	case string:
		duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(duration)
		return nil
	default:
		return errors.New("invalid duration")
	}
}

type Status string

type LogAttributes struct {
	Time  time.Time `json:"time"    validate:"required"`
	Level string    `json:"level"   validate:"required"`
}

type BenchRunAttributes struct {
	Trigger        string `json:"testTrigger"`
	Executor       string `json:"testExecutor"`
	BenchRevision  string `json:"benchRevision"`
	ServiceURL     string `json:"serviceUrl"`
	ServiceVersion string `json:"serviceVersion"`
}

type SuiteRunAttributes struct {
	RunId    string `json:"runId"`
	SuiteRun string `json:"suiteRun"`
}

type SuiteAttributes struct {
	SuiteName     string `json:"suiteName"`
	SuiteRevision string `json:"suiteRevision"`
}

type TestRunLine struct {
	LogAttributes
	BenchRunAttributes
	SuiteAttributes
	SuiteRunAttributes
	Message          string   `json:"msg"               validate:"required,eq='testRun'"`
	//FIXME: this should be a int. The issue comes from k6 executor
	Order            string   `json:"order"             validate:"required"`
	Folder           string   `json:"folder"            validate:"required"`
	TestFile         string   `json:"testFile"          validate:"required"`
	//FIXME: this should be a int. The issue comes from k6 executor
	Iterations       string   `json:"iterations"        validate:"required"`
	SetupDuration    Duration `json:"setupDuration"     validate:"required"`
	ScenarioDuration Duration `json:"scenarioDuration"  validate:"required"`
	TeardownDuration Duration `json:"teardownDuration"  validate:"required"`
	TotalDuration    Duration `json:"totalDuration"     validate:"required"`
	Status           Status   `json:"status"            validate:"required"`
	ExitMessage      string   `json:"exitMessage"`
	CloudId          string   `json:"cloudId"`
	CloudURL         string   `json:"cloudURL"`
}

type TestSuiteLine struct {
	LogAttributes
	BenchRunAttributes
	SuiteAttributes
	SuiteRunAttributes
	Message                string  `json:"msg"           validate:"required,eq='testRun'"`
	TotalScenarioDurations float64 `json:"totalScenarioDurations" validate:"required"`
	Duration               int     `json:"duration"      validate:"required"`
	TestsExecuted          int     `json:"testsExecuted" validate:"required"`
	TestsPassed            int     `json:"testsPassed"   validate:"required"`
	TestsFailed            int     `json:"testsFailed"   validate:"required"`
	TestsError             int     `json:"testsError"    validate:"required"`
	AnyFailures            bool    `json:"anyFailures"   validate:"required"`
}

type LogLines struct {
	TestRunLines  []TestRunLine
	TestSuiteLine *TestSuiteLine
}

// ValidateLog reads a structured log and validates it complies with the expected format
func ValidateLog(log io.Reader) error {
	logLines := LogLines{}
	decoder := json.NewDecoder(log)
	decoder.DisallowUnknownFields()

	var err error
	for {
		line := map[string]interface{}{}

		err := decoder.Decode(&line)
		if errors.Is(err, io.EOF) {
			break
		}
		
		if err != nil {
			return fmt.Errorf("%w %w", ErrParsingLog, err)
		}

		switch line["msg"] {
		case "testRun":
			testRunLine := TestRunLine{}
			err = unmarshallMap(line, &testRunLine)
			if err != nil {
				return err
			}
			logLines.TestRunLines = append(logLines.TestRunLines, testRunLine)
		case "testSuite":
			testSuiteLine := TestSuiteLine{}
			err = unmarshallMap(line, &testSuiteLine)
			if err != nil {
				return err
			}
			if logLines.TestSuiteLine != nil {
				return fmt.Errorf("%w multiple testSuite lines found", ErrInvalidLogStream)
			}
			logLines.TestSuiteLine = &testSuiteLine
		}
	}

	validate := validator.New()
	err = validate.Struct(logLines)
	if err != nil {
		return fmt.Errorf("validating log %w", err)
	}

	return nil
}

// takes an object and unmarshalls it into a struct
func unmarshallMap(line map[string]interface{}, v interface{}) error {
	data, err := json.Marshal(line)
	if err != nil {
		return fmt.Errorf("%w %w", ErrParsingLog, err)
	}
	
	err = json.Unmarshal(data, v)
	if err != nil {
		return fmt.Errorf("%w %w", ErrInvalidLogFormat, err)
	}

	return nil
}