package docker

import (
	"strings"
	"testing"

	"github.com/bjarneo/pipe/internal/config"
)

func TestBuildImageRef(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		tag      string
		expected string
	}{
		{"basic", "myapp", "latest", "myapp:latest"},
		{"with registry", "registry.example.com/myapp", "v1.0.0", "registry.example.com/myapp:v1.0.0"},
		{"with path", "myorg/myapp", "v1", "myorg/myapp:v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{Image: tt.image, Tag: tt.tag}
			result := buildImageRef(cfg)
			if result != tt.expected {
				t.Errorf("buildImageRef() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSortedKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected []string
	}{
		{"empty map", map[string]string{}, []string{}},
		{"single key", map[string]string{"a": "1"}, []string{"a"}},
		{"multiple keys", map[string]string{"c": "3", "a": "1", "b": "2"}, []string{"a", "b", "c"}},
		{"numeric keys", map[string]string{"2": "b", "1": "a", "3": "c"}, []string{"1", "2", "3"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sortedKeys(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("sortedKeys() returned %d keys, want %d", len(result), len(tt.expected))
				return
			}
			for i, key := range result {
				if key != tt.expected[i] {
					t.Errorf("sortedKeys()[%d] = %q, want %q", i, key, tt.expected[i])
				}
			}
		})
	}
}

func TestParseLayerList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty", "", 0},
		{"single layer", "sha256:abc123\n", 1},
		{"multiple layers", "sha256:abc123\nsha256:def456\nsha256:ghi789\n", 3},
		{"with empty lines", "sha256:abc123\n\nsha256:def456\n", 2},
		{"invalid prefix filtered", "invalid:abc123\nsha256:def456\n", 1},
		{"whitespace only", "   \n  \n", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLayerList(tt.input)
			if len(result) != tt.expected {
				t.Errorf("parseLayerList() returned %d layers, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestCalculateLayerDiff(t *testing.T) {
	tests := []struct {
		name         string
		localLayers  []string
		remoteLayers map[string]bool
		wantNew      int
		wantCached   int
	}{
		{
			name:         "all new layers",
			localLayers:  []string{"sha256:a", "sha256:b", "sha256:c"},
			remoteLayers: map[string]bool{},
			wantNew:      3,
			wantCached:   0,
		},
		{
			name:         "all cached layers",
			localLayers:  []string{"sha256:a", "sha256:b"},
			remoteLayers: map[string]bool{"sha256:a": true, "sha256:b": true},
			wantNew:      0,
			wantCached:   2,
		},
		{
			name:         "mixed layers",
			localLayers:  []string{"sha256:a", "sha256:b", "sha256:c"},
			remoteLayers: map[string]bool{"sha256:a": true},
			wantNew:      2,
			wantCached:   1,
		},
		{
			name:         "empty local layers",
			localLayers:  []string{},
			remoteLayers: map[string]bool{"sha256:a": true},
			wantNew:      0,
			wantCached:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newCount, cachedCount := calculateLayerDiff(tt.localLayers, tt.remoteLayers)
			if newCount != tt.wantNew {
				t.Errorf("calculateLayerDiff() newCount = %d, want %d", newCount, tt.wantNew)
			}
			if cachedCount != tt.wantCached {
				t.Errorf("calculateLayerDiff() cachedCount = %d, want %d", cachedCount, tt.wantCached)
			}
		})
	}
}

func TestShouldUseFullTransfer(t *testing.T) {
	tests := []struct {
		name         string
		remoteLayers map[string]bool
		newLayers    int
		totalLayers  int
		expected     bool
	}{
		{"no remote layers", map[string]bool{}, 5, 5, true},
		{"all layers new", map[string]bool{"sha256:old": true}, 5, 5, true},
		{"some cached layers", map[string]bool{"sha256:a": true}, 3, 5, false},
		{"all cached", map[string]bool{"sha256:a": true}, 0, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldUseFullTransfer(tt.remoteLayers, tt.newLayers, tt.totalLayers)
			if result != tt.expected {
				t.Errorf("shouldUseFullTransfer() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseSizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"bytes", "100", 100},
		{"kilobytes lowercase", "10k", 10 * 1024},
		{"kilobytes uppercase", "10K", 10 * 1024},
		{"kilobytes KB", "10KB", 10 * 1024},
		{"kilobytes KiB", "10KiB", 10 * 1024},
		{"megabytes", "5M", 5 * 1024 * 1024},
		{"megabytes MB", "5MB", 5 * 1024 * 1024},
		{"megabytes MiB", "5MiB", 5 * 1024 * 1024},
		{"gigabytes", "2G", 2 * 1024 * 1024 * 1024},
		{"gigabytes GB", "2GB", 2 * 1024 * 1024 * 1024},
		{"decimal megabytes", "1.5M", int64(1.5 * 1024 * 1024)},
		{"decimal with spaces", " 10.5 MiB ", int64(10.5 * 1024 * 1024)},
		{"empty string", "", 0},
		{"invalid", "abc", 0},
		{"terabytes", "1T", 1024 * 1024 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSizeString(tt.input)
			if result != tt.expected {
				t.Errorf("parseSizeString(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSizeMultiplier(t *testing.T) {
	tests := []struct {
		unit     string
		expected int64
	}{
		{"K", kilobyte},
		{"KB", kilobyte},
		{"KiB", kilobyte},
		{"M", megabyte},
		{"MB", megabyte},
		{"MiB", megabyte},
		{"G", gigabyte},
		{"GB", gigabyte},
		{"GiB", gigabyte},
		{"T", terabyte},
		{"TB", terabyte},
		{"TiB", terabyte},
		{"", 1},
		{"unknown", 1},
		{"  K  ", kilobyte}, // with whitespace
	}

	for _, tt := range tests {
		t.Run(tt.unit, func(t *testing.T) {
			result := sizeMultiplier(tt.unit)
			if result != tt.expected {
				t.Errorf("sizeMultiplier(%q) = %d, want %d", tt.unit, result, tt.expected)
			}
		})
	}
}

func TestParsePvOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"simple megabytes", "10MiB", 10 * 1024 * 1024},
		{"multiline with last valid", "line1\nline2\n5.5MiB", int64(5.5 * 1024 * 1024)},
		{"empty", "", 0},
		{"whitespace only", "   \n  \n  ", 0},
		{"size only on line", "100MB", 100 * 1024 * 1024},
		{"with prefix text", "Transferred: 100MB", 0}, // parsePvOutput expects size at start of line
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePvOutput(tt.input)
			if result != tt.expected {
				t.Errorf("parsePvOutput(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseTransferSize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"valid transfer size", "TRANSFER_SIZE:1048576", 1048576},
		{"multiline with transfer size", "some output\nTRANSFER_SIZE:2097152\nmore output", 2097152},
		{"no transfer size", "some output without size", 0},
		{"invalid number", "TRANSFER_SIZE:invalid", 0},
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTransferSize(tt.input)
			if result != tt.expected {
				t.Errorf("parseTransferSize(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetCachedLayerPrefixes(t *testing.T) {
	localLayers := []string{
		"sha256:abcdef123456789012345678901234567890",
		"sha256:fedcba098765432109876543210987654321",
		"sha256:111111222222333333444444555555666666",
	}
	remoteLayers := map[string]bool{
		"sha256:abcdef123456789012345678901234567890": true,
		"sha256:111111222222333333444444555555666666": true,
	}

	result := getCachedLayerPrefixes(localLayers, remoteLayers)

	if len(result) != 2 {
		t.Errorf("getCachedLayerPrefixes() returned %d prefixes, want 2", len(result))
	}

	// Check that prefixes are 12 characters
	for _, prefix := range result {
		if len(prefix) != layerHashPrefixLen {
			t.Errorf("prefix %q has length %d, want %d", prefix, len(prefix), layerHashPrefixLen)
		}
	}
}

func TestRunBuilder_Build(t *testing.T) {
	cfg := &config.Config{
		Image:         "myapp",
		Tag:           "latest",
		ContainerName: "mycontainer",
		HostPort:      "8080",
		ContainerPort: "3000",
		RestartPolicy: "always",
	}

	builder := NewRunBuilder(cfg)
	args := builder.Build()

	// Check required flags are present
	requiredFlags := []string{"-d", "--name", "mycontainer", "--restart", "always", "-p", "8080:3000", "myapp:latest"}
	argsStr := strings.Join(args, " ")

	for _, flag := range requiredFlags {
		if !strings.Contains(argsStr, flag) {
			t.Errorf("Build() missing required flag/value: %q in %q", flag, argsStr)
		}
	}
}

func TestRunBuilder_WithNetwork(t *testing.T) {
	cfg := &config.Config{
		Image:         "myapp",
		Tag:           "latest",
		ContainerName: "mycontainer",
		HostPort:      "8080",
		ContainerPort: "3000",
		RestartPolicy: "always",
		Network:       "mynetwork",
	}

	builder := NewRunBuilder(cfg)
	args := builder.Build()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "--network mynetwork") {
		t.Errorf("Build() missing network flag in %q", argsStr)
	}
}

func TestRunBuilder_WithVolumes(t *testing.T) {
	cfg := &config.Config{
		Image:         "myapp",
		Tag:           "latest",
		ContainerName: "mycontainer",
		HostPort:      "8080",
		ContainerPort: "3000",
		RestartPolicy: "always",
		Volumes:       []string{"/host/data:/container/data", "/host/logs:/container/logs"},
	}

	builder := NewRunBuilder(cfg)
	args := builder.Build()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "-v /host/data:/container/data") {
		t.Errorf("Build() missing first volume in %q", argsStr)
	}
	if !strings.Contains(argsStr, "-v /host/logs:/container/logs") {
		t.Errorf("Build() missing second volume in %q", argsStr)
	}
}

func TestRunBuilder_WithEnvironment(t *testing.T) {
	cfg := &config.Config{
		Image:         "myapp",
		Tag:           "latest",
		ContainerName: "mycontainer",
		HostPort:      "8080",
		ContainerPort: "3000",
		RestartPolicy: "always",
		Env:           map[string]string{"NODE_ENV": "production", "DEBUG": "false"},
		EnvFile:       ".env",
	}

	builder := NewRunBuilder(cfg)
	args := builder.Build()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "--env-file ~/.env") {
		t.Errorf("Build() missing env-file in %q", argsStr)
	}
	if !strings.Contains(argsStr, "-e DEBUG=false") {
		t.Errorf("Build() missing DEBUG env in %q", argsStr)
	}
	if !strings.Contains(argsStr, "-e NODE_ENV=production") {
		t.Errorf("Build() missing NODE_ENV env in %q", argsStr)
	}
}

func TestRunBuilder_WithLabels(t *testing.T) {
	cfg := &config.Config{
		Image:         "myapp",
		Tag:           "latest",
		ContainerName: "mycontainer",
		HostPort:      "8080",
		ContainerPort: "3000",
		RestartPolicy: "always",
		Labels:        map[string]string{"app": "myapp", "version": "1.0"},
	}

	builder := NewRunBuilder(cfg)
	args := builder.Build()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "--label app=myapp") {
		t.Errorf("Build() missing app label in %q", argsStr)
	}
	if !strings.Contains(argsStr, "--label version=1.0") {
		t.Errorf("Build() missing version label in %q", argsStr)
	}
}

func TestRunBuilder_WithHealthCheck(t *testing.T) {
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
		HealthStart:    "5s",
	}

	builder := NewRunBuilder(cfg)
	args := builder.Build()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "--health-cmd") {
		t.Errorf("Build() missing health-cmd in %q", argsStr)
	}
	if !strings.Contains(argsStr, "--health-interval 30s") {
		t.Errorf("Build() missing health-interval in %q", argsStr)
	}
	if !strings.Contains(argsStr, "--health-timeout 10s") {
		t.Errorf("Build() missing health-timeout in %q", argsStr)
	}
	if !strings.Contains(argsStr, "--health-retries 3") {
		t.Errorf("Build() missing health-retries in %q", argsStr)
	}
	if !strings.Contains(argsStr, "--health-start-period 5s") {
		t.Errorf("Build() missing health-start-period in %q", argsStr)
	}
}

func TestRunBuilder_WithSecurity(t *testing.T) {
	cfg := &config.Config{
		Image:         "myapp",
		Tag:           "latest",
		ContainerName: "mycontainer",
		HostPort:      "8080",
		ContainerPort: "3000",
		RestartPolicy: "always",
		Privileged:    true,
		Init:          true,
		ReadOnly:      true,
		CapAdd:        []string{"NET_ADMIN", "SYS_TIME"},
		CapDrop:       []string{"MKNOD"},
	}

	builder := NewRunBuilder(cfg)
	args := builder.Build()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "--privileged") {
		t.Errorf("Build() missing --privileged in %q", argsStr)
	}
	if !strings.Contains(argsStr, "--init") {
		t.Errorf("Build() missing --init in %q", argsStr)
	}
	if !strings.Contains(argsStr, "--read-only") {
		t.Errorf("Build() missing --read-only in %q", argsStr)
	}
	if !strings.Contains(argsStr, "--cap-add NET_ADMIN") {
		t.Errorf("Build() missing --cap-add NET_ADMIN in %q", argsStr)
	}
	if !strings.Contains(argsStr, "--cap-add SYS_TIME") {
		t.Errorf("Build() missing --cap-add SYS_TIME in %q", argsStr)
	}
	if !strings.Contains(argsStr, "--cap-drop MKNOD") {
		t.Errorf("Build() missing --cap-drop MKNOD in %q", argsStr)
	}
}

func TestRunBuilder_WithResources(t *testing.T) {
	cfg := &config.Config{
		Image:         "myapp",
		Tag:           "latest",
		ContainerName: "mycontainer",
		HostPort:      "8080",
		ContainerPort: "3000",
		RestartPolicy: "always",
		CPUs:          "0.5",
		Memory:        "512m",
	}

	builder := NewRunBuilder(cfg)
	args := builder.Build()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "--cpus 0.5") {
		t.Errorf("Build() missing --cpus in %q", argsStr)
	}
	if !strings.Contains(argsStr, "--memory 512m") {
		t.Errorf("Build() missing --memory in %q", argsStr)
	}
}

func TestRunBuilder_WithLogging(t *testing.T) {
	cfg := &config.Config{
		Image:         "myapp",
		Tag:           "latest",
		ContainerName: "mycontainer",
		HostPort:      "8080",
		ContainerPort: "3000",
		RestartPolicy: "always",
		LogDriver:     "json-file",
		LogOpts:       map[string]string{"max-size": "10m", "max-file": "3"},
	}

	builder := NewRunBuilder(cfg)
	args := builder.Build()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "--log-driver json-file") {
		t.Errorf("Build() missing --log-driver in %q", argsStr)
	}
	if !strings.Contains(argsStr, "--log-opt max-file=3") {
		t.Errorf("Build() missing --log-opt max-file in %q", argsStr)
	}
	if !strings.Contains(argsStr, "--log-opt max-size=10m") {
		t.Errorf("Build() missing --log-opt max-size in %q", argsStr)
	}
}

func TestRunBuilder_BuildWithImage(t *testing.T) {
	cfg := &config.Config{
		Image:         "myapp",
		Tag:           "latest",
		ContainerName: "mycontainer",
		HostPort:      "8080",
		ContainerPort: "3000",
		RestartPolicy: "always",
	}

	builder := NewRunBuilder(cfg)
	args := builder.BuildWithImage("myapp:v1.0.0")
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "myapp:v1.0.0") {
		t.Errorf("BuildWithImage() should use custom image, got %q", argsStr)
	}
	if strings.Contains(argsStr, "myapp:latest") {
		t.Errorf("BuildWithImage() should not use default image, got %q", argsStr)
	}
}

func TestBuildDockerBuildCmd(t *testing.T) {
	cfg := &config.Config{
		Image:     "myapp",
		Tag:       "latest",
		Platform:  "linux/amd64",
		BuildArgs: map[string]string{"VERSION": "1.0.0", "ENV": "prod"},
	}

	cmd := buildDockerBuildCmd(cfg)

	if !strings.Contains(cmd, "docker build") {
		t.Errorf("buildDockerBuildCmd() missing 'docker build' in %q", cmd)
	}
	if !strings.Contains(cmd, "--platform linux/amd64") {
		t.Errorf("buildDockerBuildCmd() missing platform in %q", cmd)
	}
	if !strings.Contains(cmd, "-t myapp:latest") {
		t.Errorf("buildDockerBuildCmd() missing tag in %q", cmd)
	}
	if !strings.Contains(cmd, "--build-arg ENV=prod") {
		t.Errorf("buildDockerBuildCmd() missing ENV build-arg in %q", cmd)
	}
	if !strings.Contains(cmd, "--build-arg VERSION=1.0.0") {
		t.Errorf("buildDockerBuildCmd() missing VERSION build-arg in %q", cmd)
	}
}
