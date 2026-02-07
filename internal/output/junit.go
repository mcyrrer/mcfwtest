package output

import (
	"encoding/xml"
	"fmt"
	"io"
	"time"

	"github.com/mcyrrer/mcfwtest/internal/runner"
)

// JUnitFormatter formats test results as JUnit XML
type JUnitFormatter struct {
	writer io.Writer
}

// NewJUnitFormatter creates a new JUnit formatter
func NewJUnitFormatter(w io.Writer) *JUnitFormatter {
	return &JUnitFormatter{
		writer: w,
	}
}

// JUnitTestSuites represents the root XML element
type JUnitTestSuites struct {
	XMLName    xml.Name          `xml:"testsuites"`
	Name       string            `xml:"name,attr"`
	Tests      int               `xml:"tests,attr"`
	Failures   int               `xml:"failures,attr"`
	Errors     int               `xml:"errors,attr"`
	Time       string            `xml:"time,attr"`
	TestSuites []JUnitTestSuite  `xml:"testsuite"`
}

// JUnitTestSuite represents a test suite (grouped by endpoint)
type JUnitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Time      string          `xml:"time,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a single test case
type JUnitTestCase struct {
	Name      string              `xml:"name,attr"`
	Classname string              `xml:"classname,attr"`
	Time      string              `xml:"time,attr"`
	Failure   *JUnitFailure       `xml:"failure,omitempty"`
	Error     *JUnitError         `xml:"error,omitempty"`
}

// JUnitFailure represents a test failure
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// JUnitError represents a test error
type JUnitError struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// Format outputs the test results in JUnit XML format
func (f *JUnitFormatter) Format(result *runner.RunResult, configFile, version string) error {
	// Group results by endpoint
	endpointMap := make(map[string][]runner.TestResult)
	for _, tr := range result.Results {
		name := tr.TestCase.EndpointName
		endpointMap[name] = append(endpointMap[name], tr)
	}

	// Create test suites
	testSuites := JUnitTestSuites{
		Name:     "FWProbe",
		Tests:    result.Total,
		Failures: result.Failed,
		Errors:   result.Errors,
		Time:     formatDuration(result.Duration),
	}

	for endpointName, tests := range endpointMap {
		suite := JUnitTestSuite{
			Name:      endpointName,
			Tests:     len(tests),
			Time:      "0",
		}

		var suiteDuration time.Duration
		for _, tr := range tests {
			suiteDuration += tr.Result.Latency

			// Create test case name
			testName := formatTestName(tr.TestCase.Host, tr.TestCase.Port, tr.TestCase.Protocol)
			testCase := JUnitTestCase{
				Name:      testName,
				Classname: endpointName,
				Time:      formatDuration(tr.Result.Latency),
			}

			// Add failure or error if applicable
			if tr.Error != nil {
				suite.Errors++
				testCase.Error = &JUnitError{
					Message: tr.Error.Error(),
					Type:    "error",
					Content: tr.Error.Error(),
				}
			} else if !tr.Pass {
				suite.Failures++
				message := formatFailureMessage(tr.TestCase.Expect, tr.Result.Status, tr.Result.Message)
				testCase.Failure = &JUnitFailure{
					Message: message,
					Type:    "assertion",
					Content: message,
				}
			}

			suite.TestCases = append(suite.TestCases, testCase)
		}

		suite.Time = formatDuration(suiteDuration)
		testSuites.TestSuites = append(testSuites.TestSuites, suite)
	}

	// Write XML with header
	output, err := xml.MarshalIndent(testSuites, "", "  ")
	if err != nil {
		return err
	}

	_, err = f.writer.Write([]byte(xml.Header))
	if err != nil {
		return err
	}

	_, err = f.writer.Write(output)
	if err != nil {
		return err
	}

	_, err = f.writer.Write([]byte("\n"))
	return err
}

// formatDuration formats a duration as seconds with decimal precision
func formatDuration(d time.Duration) string {
	return formatFloat(d.Seconds())
}

// formatFloat formats a float with reasonable precision
func formatFloat(f float64) string {
	return formatString("%.3f", f)
}

// formatString is a helper for string formatting
func formatString(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

// formatTestName creates a test case name
func formatTestName(host string, port int, protocol string) string {
	return formatString("%s:%d/%s", host, port, protocol)
}

// formatFailureMessage creates a failure message
func formatFailureMessage(expected string, actual interface{}, message string) string {
	if message != "" {
		return formatString("Expected %s, got %s (%s)", expected, actual, message)
	}
	return formatString("Expected %s, got %s", expected, actual)
}
