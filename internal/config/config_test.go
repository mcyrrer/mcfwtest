package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		validate    func(t *testing.T, cfg *Config)
	}{
		{
			name: "valid basic config",
			configYAML: `version: "1"
endpoints:
  - name: test
    host: 127.0.0.1
    port: 80
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "1", cfg.Version)
				assert.Len(t, cfg.Endpoints, 1)
				assert.Equal(t, "test", cfg.Endpoints[0].Name)
				assert.Equal(t, "127.0.0.1", cfg.Endpoints[0].Host)
				assert.Equal(t, 80, cfg.Endpoints[0].Port)
				// Check defaults
				assert.Equal(t, "tcp", cfg.Endpoints[0].Protocol)
				assert.Equal(t, "open", cfg.Endpoints[0].Expect)
				assert.Equal(t, 2*time.Second, cfg.Endpoints[0].Timeout.Duration)
			},
		},
		{
			name: "custom defaults",
			configYAML: `version: "1"
defaults:
  timeout: 5s
  protocol: udp
endpoints:
  - name: test
    host: 127.0.0.1
    port: 53
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 5*time.Second, cfg.Defaults.Timeout.Duration)
				assert.Equal(t, "udp", cfg.Defaults.Protocol)
				assert.Equal(t, "udp", cfg.Endpoints[0].Protocol)
				assert.Equal(t, 5*time.Second, cfg.Endpoints[0].Timeout.Duration)
			},
		},
		{
			name: "endpoint override",
			configYAML: `version: "1"
defaults:
  timeout: 2s
  protocol: tcp
endpoints:
  - name: test
    host: 127.0.0.1
    port: 53
    protocol: udp
    timeout: 3s
    expect: closed
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "udp", cfg.Endpoints[0].Protocol)
				assert.Equal(t, 3*time.Second, cfg.Endpoints[0].Timeout.Duration)
				assert.Equal(t, "closed", cfg.Endpoints[0].Expect)
			},
		},
		{
			name: "invalid version",
			configYAML: `version: "2"
endpoints:
  - name: test
    host: 127.0.0.1
    port: 80
`,
			expectError: true,
		},
		{
			name: "missing host and hosts",
			configYAML: `version: "1"
endpoints:
  - name: test
    port: 80
`,
			expectError: true,
		},
		{
			name: "both host and hosts",
			configYAML: `version: "1"
endpoints:
  - name: test
    host: 127.0.0.1
    hosts:
      - 127.0.0.2
    port: 80
`,
			expectError: true,
		},
		{
			name: "missing port and ports",
			configYAML: `version: "1"
endpoints:
  - name: test
    host: 127.0.0.1
`,
			expectError: true,
		},
		{
			name: "both port and ports",
			configYAML: `version: "1"
endpoints:
  - name: test
    host: 127.0.0.1
    port: 80
    ports:
      - 443
`,
			expectError: true,
		},
		{
			name: "invalid port range",
			configYAML: `version: "1"
endpoints:
  - name: test
    host: 127.0.0.1
    port: 70000
`,
			expectError: true,
		},
		{
			name: "duplicate endpoint names",
			configYAML: `version: "1"
endpoints:
  - name: test
    host: 127.0.0.1
    port: 80
  - name: test
    host: 127.0.0.2
    port: 443
`,
			expectError: true,
		},
		{
			name: "timeout too small",
			configYAML: `version: "1"
defaults:
  timeout: 50ms
endpoints:
  - name: test
    host: 127.0.0.1
    port: 80
`,
			expectError: true,
		},
		{
			name: "timeout too large",
			configYAML: `version: "1"
defaults:
  timeout: 60s
endpoints:
  - name: test
    host: 127.0.0.1
    port: 80
`,
			expectError: true,
		},
		{
			name: "invalid protocol",
			configYAML: `version: "1"
endpoints:
  - name: test
    host: 127.0.0.1
    port: 80
    protocol: icmp
`,
			expectError: true,
		},
		{
			name: "invalid expect",
			configYAML: `version: "1"
endpoints:
  - name: test
    host: 127.0.0.1
    port: 80
    expect: filtered
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			err := os.WriteFile(configPath, []byte(tt.configYAML), 0644)
			require.NoError(t, err)

			// Load config
			cfg, err := Load(configPath)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, cfg)
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}
		})
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
	require.NoError(t, err)

	_, err = Load(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse YAML")
}

func TestDurationUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    time.Duration
		expectError bool
	}{
		{
			name:     "seconds",
			input:    "2s",
			expected: 2 * time.Second,
		},
		{
			name:     "milliseconds",
			input:    "500ms",
			expected: 500 * time.Millisecond,
		},
		{
			name:     "multiple seconds",
			input:    "10s",
			expected: 10 * time.Second,
		},
		{
			name:        "invalid format",
			input:       "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configYAML := `version: "1"
defaults:
  timeout: ` + tt.input + `
endpoints:
  - name: test
    host: 127.0.0.1
    port: 80
`
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			err := os.WriteFile(configPath, []byte(configYAML), 0644)
			require.NoError(t, err)

			cfg, err := Load(configPath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, cfg.Defaults.Timeout.Duration)
			}
		})
	}
}
