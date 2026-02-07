package output

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mcyrrer/mcfwtest/internal/runner"
)

// PrettyFormatter formats test results for terminal output
type PrettyFormatter struct {
	NoColor bool
	Quiet   bool
	writer  io.Writer
}

// NewPrettyFormatter creates a new pretty formatter
func NewPrettyFormatter(w io.Writer, noColor, quiet bool) *PrettyFormatter {
	return &PrettyFormatter{
		NoColor: noColor,
		Quiet:   quiet,
		writer:  w,
	}
}

// Format outputs the test results in a human-readable format
func (f *PrettyFormatter) Format(result *runner.RunResult, configFile string) error {
	// Define styles - apply NoColor conditionally
	var headerStyle, passStyle, failStyle, dimStyle, boldStyle lipgloss.Style

	if f.NoColor {
		headerStyle = lipgloss.NewStyle().Bold(true).Border(lipgloss.RoundedBorder()).Padding(0, 1)
		passStyle = lipgloss.NewStyle()
		failStyle = lipgloss.NewStyle()
		dimStyle = lipgloss.NewStyle()
		boldStyle = lipgloss.NewStyle().Bold(true)
	} else {
		headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")). // Cyan
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

		passStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green
		failStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // Red
		dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))   // Gray
		boldStyle = lipgloss.NewStyle().Bold(true)
	}

	// Print header
	header := fmt.Sprintf("FWProbe — Firewall Rule Testing\nConfig: %s\nTests: %d | Concurrency: N/A",
		configFile, result.Total)
	fmt.Fprintln(f.writer, headerStyle.Render(header))
	fmt.Fprintln(f.writer)

	// Group results by endpoint
	groupedResults := f.groupByEndpoint(result.Results)

	// Print results for each endpoint
	for _, endpointName := range f.getEndpointOrder(groupedResults) {
		endpointResults := groupedResults[endpointName]

		// Print endpoint name
		fmt.Fprintln(f.writer, boldStyle.Render("  "+endpointName))

		// Print individual test results
		for _, tr := range endpointResults {
			// Skip passed tests in quiet mode
			if f.Quiet && tr.Pass {
				continue
			}

			f.printTestResult(tr, passStyle, failStyle, dimStyle)
		}

		fmt.Fprintln(f.writer)
	}

	// Print summary
	separator := strings.Repeat("─", 60)
	fmt.Fprintln(f.writer, dimStyle.Render(separator))

	summary := fmt.Sprintf("  Results: %s · %s · %d total",
		passStyle.Render(fmt.Sprintf("%d passed", result.Passed)),
		failStyle.Render(fmt.Sprintf("%d failed", result.Failed)),
		result.Total,
	)
	if result.Errors > 0 {
		summary = fmt.Sprintf("  Results: %s · %s · %s · %d total",
			passStyle.Render(fmt.Sprintf("%d passed", result.Passed)),
			failStyle.Render(fmt.Sprintf("%d failed", result.Failed)),
			failStyle.Render(fmt.Sprintf("%d errors", result.Errors)),
			result.Total,
		)
	}

	fmt.Fprintln(f.writer, summary)
	fmt.Fprintln(f.writer, fmt.Sprintf("  Duration: %v", result.Duration))
	fmt.Fprintln(f.writer, dimStyle.Render(separator))

	return nil
}

// printTestResult prints a single test result
func (f *PrettyFormatter) printTestResult(tr runner.TestResult, passStyle, failStyle, dimStyle lipgloss.Style) {
	tc := tr.TestCase
	address := fmt.Sprintf("%s:%d/%s", tc.Host, tc.Port, tc.Protocol)

	var symbol, status, details string
	var style lipgloss.Style

	if tr.Error != nil {
		symbol = "✘"
		status = "ERROR"
		details = sanitizeError(tr.Error)
		style = failStyle
	} else if tr.Pass {
		symbol = "✔"
		status = "PASS"
		details = fmt.Sprintf("%s (%s)", tr.Result.Status, formatLatency(tr.Result.Latency))
		style = passStyle
	} else {
		symbol = "✘"
		status = "FAIL"
		expectedMsg := fmt.Sprintf("expected: %s, got: %s", tc.Expect, tr.Result.Status)
		if tr.Result.Message != "" {
			details = fmt.Sprintf("%s (%s)", tr.Result.Status, tr.Result.Message)
		} else {
			details = string(tr.Result.Status)
		}
		details = fmt.Sprintf("%s (%s)", details, formatLatency(tr.Result.Latency))
		style = failStyle

		// Print address line
		line := fmt.Sprintf("    %s %s ··· %s %s",
			style.Render(symbol),
			address,
			details,
			style.Render(status),
		)
		fmt.Fprintln(f.writer, line)

		// Print expected vs actual on next line
		fmt.Fprintln(f.writer, dimStyle.Render(fmt.Sprintf("        %s", expectedMsg)))
		return
	}

	// Normal pass/error output
	line := fmt.Sprintf("    %s %s ··· %s %s",
		style.Render(symbol),
		address,
		details,
		style.Render(status),
	)
	fmt.Fprintln(f.writer, line)
}

// groupByEndpoint groups test results by endpoint name
func (f *PrettyFormatter) groupByEndpoint(results []runner.TestResult) map[string][]runner.TestResult {
	grouped := make(map[string][]runner.TestResult)
	for _, result := range results {
		name := result.TestCase.EndpointName
		grouped[name] = append(grouped[name], result)
	}
	return grouped
}

// getEndpointOrder returns endpoint names in a deterministic sorted order
func (f *PrettyFormatter) getEndpointOrder(grouped map[string][]runner.TestResult) []string {
	order := make([]string, 0, len(grouped))
	for name := range grouped {
		order = append(order, name)
	}
	sort.Strings(order)
	return order
}

// formatLatency formats a duration for display
func formatLatency(d interface{}) string {
	switch v := d.(type) {
	case int:
		return fmt.Sprintf("%dms", v)
	case time.Duration:
		return fmt.Sprintf("%dms", v.Milliseconds())
	default:
		// Fallback for unexpected types
		return fmt.Sprintf("%v", d)
	}
}

// sanitizeError returns a user-safe error message without exposing internal details
func sanitizeError(err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()

	// For network testing tool, we want to show meaningful errors
	// but avoid exposing full system paths or internal stack traces
	// Common errors like "connection refused", "timeout", "no route to host" are safe
	safeErrors := []string{
		"connection refused",
		"timeout",
		"no route to host",
		"network unreachable",
		"host unreachable",
		"protocol not supported",
		"failed to resolve hostname",
	}

	for _, safe := range safeErrors {
		if strings.Contains(strings.ToLower(msg), safe) {
			return msg
		}
	}

	// For other errors, return a generic message
	return "test execution error"
}
