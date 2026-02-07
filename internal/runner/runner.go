package runner

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mcyrrer/mcfwtest/internal/config"
	"github.com/mcyrrer/mcfwtest/internal/network"
	"github.com/mcyrrer/mcfwtest/internal/probe"
)

// IPFilter represents IP version filtering mode
type IPFilter int

const (
	// IPFilterBoth allows both IPv4 and IPv6 addresses
	IPFilterBoth IPFilter = iota
	// IPFilterIPv4Only filters to only IPv4 addresses
	IPFilterIPv4Only
	// IPFilterIPv6Only filters to only IPv6 addresses
	IPFilterIPv6Only
)

// TestCase represents a single test to be executed
type TestCase struct {
	EndpointName string
	Host         string
	Port         int
	Protocol     string
	Timeout      time.Duration
	Expect       string
}

// TestResult represents the result of a single test execution
type TestResult struct {
	TestCase TestCase
	Result   probe.Result
	Pass     bool
	Error    error
}

// RunResult represents the aggregated results of a test run
type RunResult struct {
	Results  []TestResult
	Total    int
	Passed   int
	Failed   int
	Errors   int
	Duration time.Duration
}

// Runner orchestrates test execution
type Runner struct {
	Config      *config.Config
	Concurrency int
	FailFast    bool
	Filter      string
	IPFilter    IPFilter
	Prober      probe.Prober
}

// New creates a new Runner
func New(cfg *config.Config, concurrency int, failFast bool, filter string, ipFilter IPFilter) *Runner {
	return &Runner{
		Config:      cfg,
		Concurrency: concurrency,
		FailFast:    failFast,
		Filter:      filter,
		IPFilter:    ipFilter,
		Prober:      probe.NewTCPProber(),
	}
}

// Run executes all tests and returns the results
func (r *Runner) Run(ctx context.Context) (*RunResult, error) {
	start := time.Now()

	// Expand endpoints into individual test cases
	testCases, err := r.expandTestCases()
	if err != nil {
		return nil, err
	}

	// Apply filter if specified
	if r.Filter != "" {
		testCases = r.filterTestCases(testCases)
	}

	// Execute tests with concurrency control
	results := r.executeTests(ctx, testCases)

	// Calculate summary
	runResult := &RunResult{
		Results:  results,
		Total:    len(results),
		Duration: time.Since(start),
	}

	for _, result := range results {
		if result.Error != nil {
			runResult.Errors++
		} else if result.Pass {
			runResult.Passed++
		} else {
			runResult.Failed++
		}
	}

	return runResult, nil
}

// expandTestCases expands endpoint configurations into individual test cases
func (r *Runner) expandTestCases() ([]TestCase, error) {
	var testCases []TestCase

	for _, ep := range r.Config.Endpoints {
		// Expand hosts
		hosts, err := r.expandHosts(ep)
		if err != nil {
			return nil, fmt.Errorf("endpoint %q: %w", ep.Name, err)
		}

		// Expand ports
		ports := r.expandPorts(ep)

		// Create test case for each host x port combination
		for _, host := range hosts {
			for _, port := range ports {
				testCases = append(testCases, TestCase{
					EndpointName: ep.Name,
					Host:         host,
					Port:         port,
					Protocol:     ep.Protocol,
					Timeout:      ep.Timeout.Duration,
					Expect:       ep.Expect,
				})
			}
		}
	}

	return testCases, nil
}

// expandHosts expands the host/hosts field, handling CIDR ranges and DNS resolution
func (r *Runner) expandHosts(ep config.Endpoint) ([]string, error) {
	var hosts []string

	// Handle single host
	if ep.Host != "" {
		// Check if it's a CIDR range
		if strings.Contains(ep.Host, "/") {
			ips, err := network.ExpandCIDR(ep.Host)
			if err != nil {
				return nil, err
			}
			hosts = append(hosts, ips...)
		} else {
			// Resolve hostname or use IP directly
			ips, err := network.ResolveHost(ep.Host)
			if err != nil {
				return nil, err
			}
			hosts = append(hosts, ips...)
		}
	}

	// Handle multiple hosts
	for _, host := range ep.Hosts {
		// Check if it's a CIDR range
		if strings.Contains(host, "/") {
			ips, err := network.ExpandCIDR(host)
			if err != nil {
				return nil, err
			}
			hosts = append(hosts, ips...)
		} else {
			// Resolve hostname or use IP directly
			ips, err := network.ResolveHost(host)
			if err != nil {
				return nil, err
			}
			hosts = append(hosts, ips...)
		}
	}

	// Apply IP version filtering
	if r.IPFilter != IPFilterBoth {
		hosts = r.filterIPsByVersion(hosts)
	}

	return hosts, nil
}

// filterIPsByVersion filters IP addresses based on the configured IP version filter
func (r *Runner) filterIPsByVersion(ips []string) []string {
	var filtered []string
	for _, ip := range ips {
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			continue
		}

		isIPv4 := parsedIP.To4() != nil

		switch r.IPFilter {
		case IPFilterIPv4Only:
			if isIPv4 {
				filtered = append(filtered, ip)
			}
		case IPFilterIPv6Only:
			if !isIPv4 {
				filtered = append(filtered, ip)
			}
		}
	}
	return filtered
}

// expandPorts expands the port/ports field
func (r *Runner) expandPorts(ep config.Endpoint) []int {
	if ep.Port != 0 {
		return []int{ep.Port}
	}
	return ep.Ports
}

// filterTestCases filters test cases by endpoint name using glob pattern matching
func (r *Runner) filterTestCases(testCases []TestCase) []TestCase {
	var filtered []TestCase
	for _, tc := range testCases {
		matched, err := filepath.Match(r.Filter, tc.EndpointName)
		if err == nil && matched {
			filtered = append(filtered, tc)
		}
	}
	return filtered
}

// executeTests runs all test cases with concurrency control
func (r *Runner) executeTests(ctx context.Context, testCases []TestCase) []TestResult {
	results := make([]TestResult, len(testCases))

	// Create a semaphore channel to limit concurrency
	sem := make(chan struct{}, r.Concurrency)

	// WaitGroup to wait for all tests to complete
	var wg sync.WaitGroup

	// Channel to signal early termination on fail-fast
	done := make(chan struct{})

	// Mutex to protect results slice
	var mu sync.Mutex
	failedOnce := false

	for i, tc := range testCases {
		// Check if we should stop (fail-fast mode)
		if r.FailFast {
			mu.Lock()
			if failedOnce {
				mu.Unlock()
				break
			}
			mu.Unlock()
		}

		wg.Add(1)
		go func(index int, testCase TestCase) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-done:
				return
			case <-ctx.Done():
				return
			}

			// Check context cancellation
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Execute the test
			result := r.executeTestCase(ctx, testCase)

			// Store result
			mu.Lock()
			results[index] = result
			if !result.Pass && result.Error == nil && r.FailFast {
				failedOnce = true
				close(done)
			}
			mu.Unlock()
		}(i, tc)
	}

	wg.Wait()

	// Filter out unexecuted tests in fail-fast mode
	if r.FailFast {
		var executed []TestResult
		for _, res := range results {
			if res.TestCase.EndpointName != "" {
				executed = append(executed, res)
			}
		}
		return executed
	}

	return results
}

// executeTestCase executes a single test case
func (r *Runner) executeTestCase(ctx context.Context, tc TestCase) TestResult {
	// Select the appropriate prober based on protocol
	var prober probe.Prober
	switch tc.Protocol {
	case "tcp":
		prober = probe.NewTCPProber()
	case "udp":
		prober = probe.NewUDPProber()
	default:
		return TestResult{
			TestCase: tc,
			Error:    fmt.Errorf("unsupported protocol %q", tc.Protocol),
		}
	}

	// Execute the probe
	probeResult := prober.Probe(ctx, tc.Host, tc.Port, tc.Timeout)

	// Determine pass/fail
	pass := false
	if probeResult.Status == StatusError {
		// Error status doesn't count as pass or fail
		return TestResult{
			TestCase: tc,
			Result:   probeResult,
			Error:    fmt.Errorf("%s", probeResult.Message),
		}
	}

	// Compare result with expectation
	actualStatus := string(probeResult.Status)
	pass = actualStatus == tc.Expect

	return TestResult{
		TestCase: tc,
		Result:   probeResult,
		Pass:     pass,
	}
}

// StatusError is a helper to check if a status is an error
const StatusError = probe.StatusError
