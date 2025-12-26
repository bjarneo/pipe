package ssh

import (
	"testing"

	"github.com/bjarneo/pipe/internal/config"
)

func TestGetCommand(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		expected string
	}{
		{
			name: "basic config",
			config: &config.Config{
				Host: "example.com",
				User: "deploy",
			},
			expected: "ssh deploy@example.com",
		},
		{
			name: "with SSH key",
			config: &config.Config{
				Host:   "example.com",
				User:   "deploy",
				SSHKey: "/path/to/key",
			},
			expected: "ssh -i /path/to/key deploy@example.com",
		},
		{
			name: "with custom port",
			config: &config.Config{
				Host:    "example.com",
				User:    "deploy",
				SSHPort: "2222",
			},
			expected: "ssh -p 2222 deploy@example.com",
		},
		{
			name: "with default port 22",
			config: &config.Config{
				Host:    "example.com",
				User:    "deploy",
				SSHPort: "22",
			},
			expected: "ssh deploy@example.com",
		},
		{
			name: "with key and custom port",
			config: &config.Config{
				Host:    "example.com",
				User:   "deploy",
				SSHKey:  "/path/to/key",
				SSHPort: "2222",
			},
			expected: "ssh -i /path/to/key -p 2222 deploy@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCommand(tt.config)
			if result != tt.expected {
				t.Errorf("GetCommand() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetSCPCommand(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		expected string
	}{
		{
			name: "basic config",
			config: &config.Config{
				Host: "example.com",
				User: "deploy",
			},
			expected: "scp",
		},
		{
			name: "with SSH key",
			config: &config.Config{
				Host:   "example.com",
				User:   "deploy",
				SSHKey: "/path/to/key",
			},
			expected: "scp -i /path/to/key",
		},
		{
			name: "with custom port",
			config: &config.Config{
				Host:    "example.com",
				User:    "deploy",
				SSHPort: "2222",
			},
			expected: "scp -P 2222",
		},
		{
			name: "with key and custom port",
			config: &config.Config{
				Host:    "example.com",
				User:    "deploy",
				SSHKey:  "/path/to/key",
				SSHPort: "2222",
			},
			expected: "scp -i /path/to/key -P 2222",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSCPCommand(tt.config)
			if result != tt.expected {
				t.Errorf("GetSCPCommand() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestKeyFlag(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{"empty key", "", ""},
		{"with key", "/path/to/key", "-i /path/to/key"},
		{"key with spaces in path", "/path/to/my key", "-i /path/to/my key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := keyFlag(tt.key)
			if result != tt.expected {
				t.Errorf("keyFlag(%q) = %q, want %q", tt.key, result, tt.expected)
			}
		})
	}
}

func TestPortFlag(t *testing.T) {
	tests := []struct {
		name     string
		port     string
		flag     string
		expected string
	}{
		{"empty port", "", "-p", ""},
		{"default port 22", "22", "-p", ""},
		{"custom port with -p", "2222", "-p", "-p 2222"},
		{"custom port with -P", "2222", "-P", "-P 2222"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := portFlag(tt.port, tt.flag)
			if result != tt.expected {
				t.Errorf("portFlag(%q, %q) = %q, want %q", tt.port, tt.flag, result, tt.expected)
			}
		})
	}
}

func TestGetKeyFlag(t *testing.T) {
	cfg := &config.Config{SSHKey: "/path/to/key"}
	expected := "-i /path/to/key"
	result := GetKeyFlag(cfg)
	if result != expected {
		t.Errorf("GetKeyFlag() = %q, want %q", result, expected)
	}

	cfg = &config.Config{}
	result = GetKeyFlag(cfg)
	if result != "" {
		t.Errorf("GetKeyFlag() with empty key = %q, want empty string", result)
	}
}

func TestGetPortFlag(t *testing.T) {
	cfg := &config.Config{SSHPort: "2222"}
	expected := "-p 2222"
	result := GetPortFlag(cfg)
	if result != expected {
		t.Errorf("GetPortFlag() = %q, want %q", result, expected)
	}

	cfg = &config.Config{SSHPort: "22"}
	result = GetPortFlag(cfg)
	if result != "" {
		t.Errorf("GetPortFlag() with default port = %q, want empty string", result)
	}
}
