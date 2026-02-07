package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mcyrrer/mcfwtest/internal/config"
	"github.com/mcyrrer/mcfwtest/internal/output"
	"github.com/mcyrrer/mcfwtest/internal/runner"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "fwprobe",
	Short: "Firewall Rule Testing Tool",
	Long:  "FWProbe is a cross-platform, CLI-based network testing tool that validates firewall rules.",
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run firewall tests",
	Long:  "Run firewall rule tests defined in a configuration file.",
	RunE:  runTests,
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	Long:  "Validate the configuration file and print a summary of planned tests.",
	RunE:  validateConfig,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("FWProbe %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
	},
}

var createconfigCmd = &cobra.Command{
	Use:   "createconfig",
	Short: "Create a template configuration file",
	Long:  "Generate a template configuration file with all valid configuration options to aid in creating your own config.",
	RunE:  createConfig,
}

var (
	configFile  string
	outputMode  string
	failFast    bool
	concurrency int
	filter      string
	noColor     bool
	exitCode    bool
	quiet       bool
	ipv4Only    bool
	ipv6Only    bool
	outputFile  string
)

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(createconfigCmd)

	// Run command flags
	runCmd.Flags().StringVarP(&configFile, "config", "c", "fwprobe.yaml", "Path to config file")
	runCmd.Flags().StringVarP(&outputMode, "output", "o", "pretty", "Output format: pretty, json, junit")
	runCmd.Flags().BoolVar(&failFast, "fail-fast", false, "Stop on first failure")
	runCmd.Flags().IntVarP(&concurrency, "concurrency", "j", 10, "Max concurrent tests")
	runCmd.Flags().StringVarP(&filter, "filter", "f", "", "Run only endpoints matching name pattern (glob)")
	runCmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	runCmd.Flags().BoolVar(&exitCode, "exit-code", true, "Exit with code 1 if any test fails")
	runCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Only print failures and summary")
	runCmd.Flags().BoolVar(&ipv4Only, "ipv4-only", false, "Test only IPv4 addresses")
	runCmd.Flags().BoolVar(&ipv6Only, "ipv6-only", false, "Test only IPv6 addresses")

	// Validate command flags
	validateCmd.Flags().StringVarP(&configFile, "config", "c", "fwprobe.yaml", "Path to config file")

	// Createconfig command flags
	createconfigCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write to file instead of stdout")
}

func runTests(cmd *cobra.Command, args []string) error {
	// Validate flags
	if ipv4Only && ipv6Only {
		fmt.Fprintf(os.Stderr, "Error: Cannot specify both --ipv4-only and --ipv6-only\n")
		return exitWithCode(2)
	}

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load configuration: %v\n", err)
		return exitWithCode(2)
	}

	// Determine IP filter mode
	ipFilter := runner.IPFilterBoth
	if ipv4Only {
		ipFilter = runner.IPFilterIPv4Only
	} else if ipv6Only {
		ipFilter = runner.IPFilterIPv6Only
	}

	// Create runner
	r := runner.New(cfg, concurrency, failFast, filter, ipFilter)

	// Execute tests
	ctx := context.Background()
	result, err := r.Run(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to run tests: %v\n", err)
		return exitWithCode(3)
	}

	// Format and print results
	switch outputMode {
	case "pretty":
		formatter := output.NewPrettyFormatter(os.Stdout, noColor, quiet)
		if err := formatter.Format(result, configFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to format output: %v\n", err)
			return exitWithCode(3)
		}
	case "json":
		formatter := output.NewJSONFormatter(os.Stdout)
		if err := formatter.Format(result, configFile, version); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to format JSON output: %v\n", err)
			return exitWithCode(3)
		}
	case "junit":
		formatter := output.NewJUnitFormatter(os.Stdout)
		if err := formatter.Format(result, configFile, version); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to format JUnit output: %v\n", err)
			return exitWithCode(3)
		}
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown output format %q\n", outputMode)
		return exitWithCode(3)
	}

	// Exit with appropriate code
	if exitCode && (result.Failed > 0 || result.Errors > 0) {
		return exitWithCode(1)
	}

	return nil
}

func validateConfig(cmd *cobra.Command, args []string) error {
	// Load and validate configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load configuration: %v\n", err)
		return exitWithCode(2)
	}

	// Print summary
	fmt.Printf("Configuration is valid: %s\n", configFile)
	fmt.Printf("  Version: %s\n", cfg.Version)
	fmt.Printf("  Endpoints: %d\n", len(cfg.Endpoints))

	// Count total tests
	totalTests := 0
	for _, ep := range cfg.Endpoints {
		hostCount := 1
		if len(ep.Hosts) > 0 {
			hostCount = len(ep.Hosts)
		}

		portCount := 1
		if len(ep.Ports) > 0 {
			portCount = len(ep.Ports)
		}

		totalTests += hostCount * portCount
	}

	fmt.Printf("  Estimated tests: %d\n", totalTests)

	return nil
}

func createConfig(cmd *cobra.Command, args []string) error {
	templateConfig := `# FWProbe Configuration Template
# This file contains all valid configuration options with examples

version: "1"

# Global defaults applied to all endpoints
defaults:
  # Default timeout for all tests (min: 100ms, max: 30s)
  timeout: 2s
  # Default protocol: tcp or udp
  protocol: tcp

# List of endpoints to test
endpoints:
  # Example 1: Single host and port
  - name: example-web-server
    description: Test if web server is accessible
    host: example.com
    port: 443
    protocol: tcp
    timeout: 5s
    expect: open  # Expected result: "open" or "closed"

  # Example 2: Multiple hosts with single port
  - name: dns-servers
    description: Test multiple DNS servers
    hosts:
      - 8.8.8.8
      - 8.8.4.4
      - 1.1.1.1
    port: 53
    protocol: udp
    timeout: 2s
    expect: open

  # Example 3: Single host with multiple ports
  - name: common-ports
    description: Test common ports on a server
    host: 192.168.1.1
    ports:
      - 22
      - 80
      - 443
    protocol: tcp
    timeout: 3s
    expect: open

  # Example 4: CIDR range testing
  - name: network-scan
    description: Scan a subnet for SSH access
    host: 192.168.1.0/24
    port: 22
    protocol: tcp
    timeout: 1s
    expect: closed

  # Example 5: Expect closed port (firewall should block)
  - name: blocked-service
    description: Verify that a port is blocked
    host: internal-server.local
    port: 3306
    protocol: tcp
    timeout: 2s
    expect: closed
`

	// Write to file or stdout
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(templateConfig), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to write config file: %v\n", err)
			return exitWithCode(1)
		}
		fmt.Printf("Template configuration written to: %s\n", outputFile)
	} else {
		fmt.Print(templateConfig)
	}

	return nil
}

func exitWithCode(code int) error {
	os.Exit(code)
	return nil
}
