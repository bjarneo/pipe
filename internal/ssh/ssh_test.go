package ssh

import (
	"testing"

	"github.com/bjarneo/pipe/internal/config"
)

func TestBuildSSHCommand(t *testing.T) {
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
				User:    "deploy",
				SSHKey:  "/path/to/key",
				SSHPort: "2222",
			},
			expected: "ssh -i /path/to/key -p 2222 deploy@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildSSHCommand(tt.config)
			if result != tt.expected {
				t.Errorf("BuildSSHCommand() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBuildSCPCommand(t *testing.T) {
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
			result := BuildSCPCommand(tt.config)
			if result != tt.expected {
				t.Errorf("BuildSCPCommand() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBuildKeyFlagInternal(t *testing.T) {
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
			result := buildKeyFlag(tt.key)
			if result != tt.expected {
				t.Errorf("buildKeyFlag(%q) = %q, want %q", tt.key, result, tt.expected)
			}
		})
	}
}

func TestBuildPortFlagInternal(t *testing.T) {
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
			result := buildPortFlag(tt.port, tt.flag)
			if result != tt.expected {
				t.Errorf("buildPortFlag(%q, %q) = %q, want %q", tt.port, tt.flag, result, tt.expected)
			}
		})
	}
}

func TestBuildKeyFlag(t *testing.T) {
	cfg := &config.Config{SSHKey: "/path/to/key"}
	expected := "-i /path/to/key"
	result := BuildKeyFlag(cfg)
	if result != expected {
		t.Errorf("BuildKeyFlag() = %q, want %q", result, expected)
	}

	cfg = &config.Config{}
	result = BuildKeyFlag(cfg)
	if result != "" {
		t.Errorf("BuildKeyFlag() with empty key = %q, want empty string", result)
	}
}

func TestBuildPortFlag(t *testing.T) {
	cfg := &config.Config{SSHPort: "2222"}
	expected := "-p 2222"
	result := BuildPortFlag(cfg)
	if result != expected {
		t.Errorf("BuildPortFlag() = %q, want %q", result, expected)
	}

	cfg = &config.Config{SSHPort: "22"}
	result = BuildPortFlag(cfg)
	if result != "" {
		t.Errorf("BuildPortFlag() with default port = %q, want empty string", result)
	}
}
