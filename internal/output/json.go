package output

import (
	"encoding/json"
	"io"
	"time"

	"github.com/mcyrrer/mcfwtest/internal/runner"
)

// JSONFormatter formats test results as JSON
type JSONFormatter struct {
	writer io.Writer
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(w io.Writer) *JSONFormatter {
	return &JSONFormatter{
		writer: w,
	}
}

// JSONOutput represents the JSON output structure
type JSONOutput struct {
	Version    string      `json:"version"`
	Timestamp  string      `json:"timestamp"`
	ConfigFile string      `json:"config_file"`
	Summary    JSONSummary `json:"summary"`
	Results    []JSONResult `json:"results"`
}

// JSONSummary represents the summary section
type JSONSummary struct {
	Total      int `json:"total"`
	Passed     int `json:"passed"`
	Failed     int `json:"failed"`
	Errors     int `json:"errors"`
	DurationMS int64 `json:"duration_ms"`
}

// JSONResult represents a single test result
type JSONResult struct {
	Endpoint  string `json:"endpoint"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"`
	Expected  string `json:"expected"`
	Actual    string `json:"actual"`
	Status    string `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
	Message   string `json:"message"`
}

// Format outputs the test results in JSON format
func (f *JSONFormatter) Format(result *runner.RunResult, configFile, version string) error {
	output := JSONOutput{
		Version:    version,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		ConfigFile: configFile,
		Summary: JSONSummary{
			Total:      result.Total,
			Passed:     result.Passed,
			Failed:     result.Failed,
			Errors:     result.Errors,
			DurationMS: result.Duration.Milliseconds(),
		},
		Results: make([]JSONResult, 0, len(result.Results)),
	}

	for _, tr := range result.Results {
		status := "pass"
		if tr.Error != nil {
			status = "error"
		} else if !tr.Pass {
			status = "fail"
		}

		actual := string(tr.Result.Status)
		if tr.Error != nil {
			actual = "error"
		}

		jsonResult := JSONResult{
			Endpoint:  tr.TestCase.EndpointName,
			Host:      tr.TestCase.Host,
			Port:      tr.TestCase.Port,
			Protocol:  tr.TestCase.Protocol,
			Expected:  tr.TestCase.Expect,
			Actual:    actual,
			Status:    status,
			LatencyMS: tr.Result.Latency.Milliseconds(),
			Message:   tr.Result.Message,
		}

		if tr.Error != nil {
			jsonResult.Message = tr.Error.Error()
		}

		output.Results = append(output.Results, jsonResult)
	}

	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
