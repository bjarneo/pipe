package deploy

import (
	"strings"
	"testing"

	"github.com/bjarneo/pipe/internal/config"
)

const testPreviousImage = "myapp:v1.0.0"

func TestBuildRollbackCommands(t *testing.T) {
	cfg := &config.Config{
		Image:         "myapp",
		Tag:           "latest",
		ContainerName: "mycontainer",
		HostPort:      "8080",
		ContainerPort: "3000",
		RestartPolicy: "always",
	}

	result := buildRollbackCommands(cfg, testPreviousImage)

	// Check that it contains the expected docker commands
	if !strings.Contains(result, "docker stop mycontainer") {
		t.Errorf("buildRollbackCommands() missing stop command in %q", result)
	}
	if !strings.Contains(result, "docker rename mycontainer mycontainer_backup") {
		t.Errorf("buildRollbackCommands() missing rename command in %q", result)
	}
	if !strings.Contains(result, "docker run") {
		t.Errorf("buildRollbackCommands() missing run command in %q", result)
	}
	if !strings.Contains(result, testPreviousImage) {
		t.Errorf("buildRollbackCommands() missing previous image %q in %q", testPreviousImage, result)
	}
	// Commands should be chained with &&
	if !strings.Contains(result, " && ") {
		t.Errorf("buildRollbackCommands() commands should be chained with &&, got %q", result)
	}
}

func TestBuildRollbackCommands_PreservesConfig(t *testing.T) {
	cfg := &config.Config{
		Image:         "myapp",
		Tag:           "latest",
		ContainerName: "mycontainer",
		HostPort:      "8080",
		ContainerPort: "3000",
		RestartPolicy: "always",
		Network:       "mynetwork",
		Volumes:       []string{"/data:/app/data"},
		Env:           map[string]string{"NODE_ENV": "production"},
		Labels:        map[string]string{"app": "myapp"},
		CPUs:          "0.5",
		Memory:        "512m",
	}

	result := buildRollbackCommands(cfg, testPreviousImage)

	// Verify container configuration is preserved
	if !strings.Contains(result, "--network mynetwork") {
		t.Errorf("buildRollbackCommands() missing network in %q", result)
	}
	if !strings.Contains(result, "-v /data:/app/data") {
		t.Errorf("buildRollbackCommands() missing volume in %q", result)
	}
	if !strings.Contains(result, "-e NODE_ENV=production") {
		t.Errorf("buildRollbackCommands() missing env var in %q", result)
	}
	if !strings.Contains(result, "--label app=myapp") {
		t.Errorf("buildRollbackCommands() missing label in %q", result)
	}
	if !strings.Contains(result, "--cpus 0.5") {
		t.Errorf("buildRollbackCommands() missing cpus in %q", result)
	}
	if !strings.Contains(result, "--memory 512m") {
		t.Errorf("buildRollbackCommands() missing memory in %q", result)
	}
}

func TestBuildRollbackCommands_WithHealthCheck(t *testing.T) {
	cfg := &config.Config{
		Image:          "myapp",
		Tag:            "latest",
		ContainerName:  "mycontainer",
		HostPort:       "8080",
		ContainerPort:  "3000",
		RestartPolicy:  "always",
		HealthCmd:      "curl -f http://localhost/health",
		HealthInterval: "30s",
		HealthTimeout:  "10s",
		HealthRetries:  3,
	}

	result := buildRollbackCommands(cfg, testPreviousImage)

	if !strings.Contains(result, "--health-cmd") {
		t.Errorf("buildRollbackCommands() missing health-cmd in %q", result)
	}
	if !strings.Contains(result, "--health-interval 30s") {
		t.Errorf("buildRollbackCommands() missing health-interval in %q", result)
	}
	if !strings.Contains(result, "--health-timeout 10s") {
		t.Errorf("buildRollbackCommands() missing health-timeout in %q", result)
	}
	if !strings.Contains(result, "--health-retries 3") {
		t.Errorf("buildRollbackCommands() missing health-retries in %q", result)
	}
}

func TestBuildRollbackCommands_WithSecurity(t *testing.T) {
	cfg := &config.Config{
		Image:         "myapp",
		Tag:           "latest",
		ContainerName: "mycontainer",
		HostPort:      "8080",
		ContainerPort: "3000",
		RestartPolicy: "always",
		Privileged:    true,
		ReadOnly:      true,
		Init:          true,
		CapAdd:        []string{"NET_ADMIN"},
		CapDrop:       []string{"MKNOD"},
	}

	result := buildRollbackCommands(cfg, testPreviousImage)

	if !strings.Contains(result, "--privileged") {
		t.Errorf("buildRollbackCommands() missing --privileged in %q", result)
	}
	if !strings.Contains(result, "--read-only") {
		t.Errorf("buildRollbackCommands() missing --read-only in %q", result)
	}
	if !strings.Contains(result, "--init") {
		t.Errorf("buildRollbackCommands() missing --init in %q", result)
	}
	if !strings.Contains(result, "--cap-add NET_ADMIN") {
		t.Errorf("buildRollbackCommands() missing --cap-add in %q", result)
	}
	if !strings.Contains(result, "--cap-drop MKNOD") {
		t.Errorf("buildRollbackCommands() missing --cap-drop in %q", result)
	}
}

func TestFindPreviousImage(t *testing.T) {
	tests := []struct {
		name         string
		images       []string
		currentImage string
		expected     string
		expectErr    bool
	}{
		{
			name:         "find previous version",
			images:       []string{"myapp:v2___2024-01-02", "myapp:v1___2024-01-01"},
			currentImage: "myapp:v2",
			expected:     "myapp:v1",
			expectErr:    false,
		},
		{
			name:         "no previous version",
			images:       []string{"myapp:v1___2024-01-01"},
			currentImage: "myapp:v1",
			expected:     "",
			expectErr:    true,
		},
		{
			name:         "empty list",
			images:       []string{},
			currentImage: "myapp:v1",
			expected:     "",
			expectErr:    true,
		},
		{
			name:         "current not found",
			images:       []string{"myapp:v2___2024-01-02", "myapp:v1___2024-01-01"},
			currentImage: "myapp:v3",
			expected:     "",
			expectErr:    true,
		},
		{
			name:         "with multiple versions",
			images:       []string{"myapp:v3___2024-01-03", "myapp:v2___2024-01-02", "myapp:v1___2024-01-01"},
			currentImage: "myapp:v3",
			expected:     "myapp:v2",
			expectErr:    false,
		},
		{
			name:         "timestamped latest tags",
			images:       []string{"myapp:latest-20251226150200___2025-12-26 15:02:00", "myapp:latest-20251226150100___2025-12-26 15:01:00", "myapp:latest-20251226150000___2025-12-26 15:00:00"},
			currentImage: "myapp:latest-20251226150200",
			expected:     "myapp:latest-20251226150100",
			expectErr:    false,
		},
		{
			name:         "rollback from first timestamped tag",
			images:       []string{"myapp:latest-20251226150000___2025-12-26 15:00:00"},
			currentImage: "myapp:latest-20251226150000",
			expected:     "",
			expectErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := findPreviousImage(tt.images, tt.currentImage)
			if tt.expectErr {
				if err == nil {
					t.Errorf("findPreviousImage() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("findPreviousImage() unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("findPreviousImage() = %q, want %q", result, tt.expected)
				}
			}
		})
	}
}

func TestFindPreviousImage_MalformedEntries(t *testing.T) {
	// Test that malformed entries before the current image are skipped
	images := []string{
		"___",                   // malformed entry - skipped during iteration
		"myapp:v2___2024-01-02", // valid current
		"myapp:v1___2024-01-01", // valid previous (returned)
	}

	result, err := findPreviousImage(images, "myapp:v2")
	if err != nil {
		t.Errorf("findPreviousImage() with malformed entries should still work: %v", err)
	}
	if result != "myapp:v1" {
		t.Errorf("findPreviousImage() = %q, want %q", result, "myapp:v1")
	}
}

func TestFindPreviousImage_MalformedNextEntry(t *testing.T) {
	// When the entry after current is malformed, it returns the malformed result
	images := []string{
		"myapp:v2___2024-01-02", // valid current
		"___",                   // malformed - but this is what gets returned
		"myapp:v1___2024-01-01", // not reached
	}

	result, err := findPreviousImage(images, "myapp:v2")
	// The function returns whatever is in nextParts[0], which is empty string
	if err != nil {
		t.Errorf("findPreviousImage() unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("findPreviousImage() = %q, want empty string (from malformed entry)", result)
	}
}

func TestPrintDryRunSummary(t *testing.T) {
	// Test that printDryRunSummary doesn't panic with various configs
	testCases := []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "minimal config",
			cfg: &config.Config{
				Host:          "example.com",
				User:          "deploy",
				Image:         "myapp",
				Tag:           "latest",
				ContainerName: "mycontainer",
				HostPort:      "8080",
				ContainerPort: "3000",
				Platform:      "linux/amd64",
				RestartPolicy: "always",
			},
		},
		{
			name: "with network",
			cfg: &config.Config{
				Host:          "example.com",
				User:          "deploy",
				Image:         "myapp",
				Tag:           "latest",
				ContainerName: "mycontainer",
				HostPort:      "8080",
				ContainerPort: "3000",
				Platform:      "linux/amd64",
				RestartPolicy: "always",
				Network:       "mynetwork",
			},
		},
		{
			name: "with resources",
			cfg: &config.Config{
				Host:          "example.com",
				User:          "deploy",
				Image:         "myapp",
				Tag:           "latest",
				ContainerName: "mycontainer",
				HostPort:      "8080",
				ContainerPort: "3000",
				Platform:      "linux/amd64",
				RestartPolicy: "always",
				CPUs:          "0.5",
				Memory:        "512m",
			},
		},
		{
			name: "with env file",
			cfg: &config.Config{
				Host:          "example.com",
				User:          "deploy",
				Image:         "myapp",
				Tag:           "latest",
				ContainerName: "mycontainer",
				HostPort:      "8080",
				ContainerPort: "3000",
				Platform:      "linux/amd64",
				RestartPolicy: "always",
				EnvFile:       ".env",
			},
		},
		{
			name: "with env vars",
			cfg: &config.Config{
				Host:          "example.com",
				User:          "deploy",
				Image:         "myapp",
				Tag:           "latest",
				ContainerName: "mycontainer",
				HostPort:      "8080",
				ContainerPort: "3000",
				Platform:      "linux/amd64",
				RestartPolicy: "always",
				Env:           map[string]string{"NODE_ENV": "production"},
			},
		},
		{
			name: "with volumes",
			cfg: &config.Config{
				Host:          "example.com",
				User:          "deploy",
				Image:         "myapp",
				Tag:           "latest",
				ContainerName: "mycontainer",
				HostPort:      "8080",
				ContainerPort: "3000",
				Platform:      "linux/amd64",
				RestartPolicy: "always",
				Volumes:       []string{"/data:/app/data"},
			},
		},
		{
			name: "with labels",
			cfg: &config.Config{
				Host:          "example.com",
				User:          "deploy",
				Image:         "myapp",
				Tag:           "latest",
				ContainerName: "mycontainer",
				HostPort:      "8080",
				ContainerPort: "3000",
				Platform:      "linux/amd64",
				RestartPolicy: "always",
				Labels:        map[string]string{"app": "myapp"},
			},
		},
		{
			name: "with health check",
			cfg: &config.Config{
				Host:          "example.com",
				User:          "deploy",
				Image:         "myapp",
				Tag:           "latest",
				ContainerName: "mycontainer",
				HostPort:      "8080",
				ContainerPort: "3000",
				Platform:      "linux/amd64",
				RestartPolicy: "always",
				HealthCmd:     "curl -f http://localhost/health",
			},
		},
		{
			name: "with build args",
			cfg: &config.Config{
				Host:          "example.com",
				User:          "deploy",
				Image:         "myapp",
				Tag:           "latest",
				ContainerName: "mycontainer",
				HostPort:      "8080",
				ContainerPort: "3000",
				Platform:      "linux/amd64",
				RestartPolicy: "always",
				BuildArgs:     map[string]string{"VERSION": "1.0.0"},
			},
		},
		{
			name: "with remote commands",
			cfg: &config.Config{
				Host:           "example.com",
				User:           "deploy",
				Image:          "myapp",
				Tag:            "latest",
				ContainerName:  "mycontainer",
				HostPort:       "8080",
				ContainerPort:  "3000",
				Platform:       "linux/amd64",
				RestartPolicy:  "always",
				RemoteCommands: []string{"docker system prune -f", "echo done"},
			},
		},
		{
			name: "with security options",
			cfg: &config.Config{
				Host:          "example.com",
				User:          "deploy",
				Image:         "myapp",
				Tag:           "latest",
				ContainerName: "mycontainer",
				HostPort:      "8080",
				ContainerPort: "3000",
				Platform:      "linux/amd64",
				RestartPolicy: "always",
				Privileged:    true,
				ReadOnly:      true,
				Init:          true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("printDryRunSummary() panicked: %v", r)
				}
			}()

			// Create a mock logger that does nothing
			err := printDryRunSummary(tc.cfg, nil)
			if err != nil {
				t.Errorf("printDryRunSummary() error = %v", err)
			}
		})
	}
}
