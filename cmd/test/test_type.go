package test

import (
	"fmt"
	"strings"
)

// This type setup uses a system that allows us to check test types at compile
// time. Modeled after https://github.com/ardanlabs/service/blob/master/business/core/user/role.go

var (
	SmokeTest = TestType{"smoke"}
	LoadTest  = TestType{"load"}
)

var testTypes = map[string]TestType{
	SmokeTest.name: SmokeTest,
	LoadTest.name:  LoadTest,
}

type TestType struct {
	name string
}

// Name returns the name of the testType
func (t TestType) Name() string {
	return t.name
}

// Checks the testTypes map for valid testType
func ParseTestType(value string) (TestType, error) {
	testType, exists := testTypes[strings.ToLower(value)]
	if !exists {
		return TestType{}, fmt.Errorf("invalid testType %q", value)
	}

	return testType, nil
}

// MustParseTestType parses the string value and returns a testType if one exists. If
// an error occurs the function panics.
func MustParseTestType(value string) TestType {
	testType, err := ParseTestType(value)
	if err != nil {
		panic(err)
	}

	return testType
}

// MarshalText implement the marshal interface for JSON conversions.
func (t TestType) MarshalText() ([]byte, error) {
	return []byte(t.name), nil
}

// Equal provides support for the go-cmp package and testing.
func (t TestType) Equal(t2 TestType) bool {
	return t.name == t2.name
}

type TestRunType string

const (
	// single iteration, fail slow, returns with exit code
	Smoke TestRunType = "smoke"
	// xxx iterations, don't fail, report to k6 cloud
	Load TestRunType = "load"
)

func (trt TestRunType) String() string {
	switch trt {
	case Smoke:
		return "smoke"
	case Load:
		return "load"
	default:
		panic("Unknown TestRunType")
	}
}

// Gets the TestRunType from a string
//func TestRunTypeFromString(trt string) (TestRunType, error) {
//  trt = strings.ToLower(trt)
//  switch trt {
//  case "smoke":
//    return Smoke, nil
//  case "load":
//    return Load, nil
//  default:
//    return nil, fmt.Sprintf("invalid test run type %s", trt)
//  }
//}
