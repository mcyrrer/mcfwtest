package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the root configuration structure
type Config struct {
	Version   string    `yaml:"version"`
	Defaults  Defaults  `yaml:"defaults"`
	Endpoints []Endpoint `yaml:"endpoints"`
}

// Defaults contains global default values
type Defaults struct {
	Timeout  Duration `yaml:"timeout"`
	Protocol string   `yaml:"protocol"`
}

// Endpoint represents a single endpoint test configuration
type Endpoint struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Host        string   `yaml:"host"`
	Hosts       []string `yaml:"hosts"`
	Port        int      `yaml:"port"`
	Ports       []int    `yaml:"ports"`
	Protocol    string   `yaml:"protocol"`
	Timeout     Duration `yaml:"timeout"`
	Expect      string   `yaml:"expect"`
}

// Duration is a wrapper around time.Duration that supports YAML unmarshaling
type Duration struct {
	time.Duration
}

// UnmarshalYAML implements custom YAML unmarshaling for Duration
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}

	d.Duration = duration
	return nil
}

// Load reads and parses a configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Apply defaults
	if err := applyDefaults(&cfg); err != nil {
		return nil, err
	}

	// Validate configuration
	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// applyDefaults applies default values to the configuration
func applyDefaults(cfg *Config) error {
	// Set global defaults if not specified
	if cfg.Defaults.Timeout.Duration == 0 {
		cfg.Defaults.Timeout.Duration = 2 * time.Second
	}
	if cfg.Defaults.Protocol == "" {
		cfg.Defaults.Protocol = "tcp"
	}

	// Apply defaults to each endpoint
	for i := range cfg.Endpoints {
		ep := &cfg.Endpoints[i]

		if ep.Protocol == "" {
			ep.Protocol = cfg.Defaults.Protocol
		}
		if ep.Timeout.Duration == 0 {
			ep.Timeout.Duration = cfg.Defaults.Timeout.Duration
		}
		if ep.Expect == "" {
			ep.Expect = "open"
		}
	}

	return nil
}

// validate validates the configuration against all rules
func validate(cfg *Config) error {
	// Validate version
	if cfg.Version != "1" {
		return fmt.Errorf("unsupported config version %q, must be \"1\"", cfg.Version)
	}

	// Validate global defaults
	if cfg.Defaults.Timeout.Duration < 100*time.Millisecond {
		return fmt.Errorf("global timeout must be at least 100ms")
	}
	if cfg.Defaults.Timeout.Duration > 30*time.Second {
		return fmt.Errorf("global timeout must not exceed 30s")
	}
	if cfg.Defaults.Protocol != "tcp" && cfg.Defaults.Protocol != "udp" {
		return fmt.Errorf("global protocol must be \"tcp\" or \"udp\"")
	}

	// Track endpoint names for uniqueness check
	names := make(map[string]bool)

	// Validate each endpoint
	for i, ep := range cfg.Endpoints {
		if err := validateEndpoint(&ep, i); err != nil {
			return err
		}

		// Check name uniqueness
		if names[ep.Name] {
			return fmt.Errorf("duplicate endpoint name: %q", ep.Name)
		}
		names[ep.Name] = true
	}

	return nil
}

// validateEndpoint validates a single endpoint configuration
func validateEndpoint(ep *Endpoint, index int) error {
	// Validate name
	if ep.Name == "" {
		return fmt.Errorf("endpoint[%d]: name is required", index)
	}

	// Validate host/hosts exclusivity
	if ep.Host != "" && len(ep.Hosts) > 0 {
		return fmt.Errorf("endpoint %q: cannot specify both 'host' and 'hosts'", ep.Name)
	}
	if ep.Host == "" && len(ep.Hosts) == 0 {
		return fmt.Errorf("endpoint %q: must specify either 'host' or 'hosts'", ep.Name)
	}

	// Validate port/ports exclusivity
	if ep.Port != 0 && len(ep.Ports) > 0 {
		return fmt.Errorf("endpoint %q: cannot specify both 'port' and 'ports'", ep.Name)
	}
	if ep.Port == 0 && len(ep.Ports) == 0 {
		return fmt.Errorf("endpoint %q: must specify either 'port' or 'ports'", ep.Name)
	}

	// Validate port ranges
	if ep.Port != 0 {
		if err := validatePort(ep.Port); err != nil {
			return fmt.Errorf("endpoint %q: %w", ep.Name, err)
		}
	}
	for _, port := range ep.Ports {
		if err := validatePort(port); err != nil {
			return fmt.Errorf("endpoint %q: %w", ep.Name, err)
		}
	}

	// Validate protocol
	if ep.Protocol != "tcp" && ep.Protocol != "udp" {
		return fmt.Errorf("endpoint %q: protocol must be \"tcp\" or \"udp\"", ep.Name)
	}

	// Validate timeout
	if ep.Timeout.Duration < 100*time.Millisecond {
		return fmt.Errorf("endpoint %q: timeout must be at least 100ms", ep.Name)
	}
	if ep.Timeout.Duration > 30*time.Second {
		return fmt.Errorf("endpoint %q: timeout must not exceed 30s", ep.Name)
	}

	// Validate expect
	if ep.Expect != "open" && ep.Expect != "closed" {
		return fmt.Errorf("endpoint %q: expect must be \"open\" or \"closed\"", ep.Name)
	}

	return nil
}

// validatePort validates a port number
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port %d is out of range (1-65535)", port)
	}
	return nil
}
