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

var (
	configFile  string
	outputMode  string
	failFast    bool
	concurrency int
	filter      string
	noColor     bool
	exitCode    bool
	quiet       bool
)

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(versionCmd)

	// Run command flags
	runCmd.Flags().StringVarP(&configFile, "config", "c", "fwprobe.yaml", "Path to config file")
	runCmd.Flags().StringVarP(&outputMode, "output", "o", "pretty", "Output format: pretty, json, junit")
	runCmd.Flags().BoolVar(&failFast, "fail-fast", false, "Stop on first failure")
	runCmd.Flags().IntVarP(&concurrency, "concurrency", "j", 10, "Max concurrent tests")
	runCmd.Flags().StringVarP(&filter, "filter", "f", "", "Run only endpoints matching name pattern (glob)")
	runCmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	runCmd.Flags().BoolVar(&exitCode, "exit-code", true, "Exit with code 1 if any test fails")
	runCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Only print failures and summary")

	// Validate command flags
	validateCmd.Flags().StringVarP(&configFile, "config", "c", "fwprobe.yaml", "Path to config file")
}

func runTests(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load configuration: %v\n", err)
		return exitWithCode(2)
	}

	// Create runner
	r := runner.New(cfg, concurrency, failFast, filter)

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

func exitWithCode(code int) error {
	os.Exit(code)
	return nil
}
