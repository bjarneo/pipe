package container

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTruncateID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected string
	}{
		{"short id", "abc123", "abc123"},
		{"exactly 12", "abcdef123456", "abcdef123456"},
		{"longer than 12", "abcdef1234567890", "abcdef123456"},
		{"empty", "", ""},
		{"64 char sha", "sha256abcdef1234567890abcdef1234567890abcdef1234567890abcdef12", "sha256abcdef"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateID(tt.id)
			if result != tt.expected {
				t.Errorf("truncateID(%q) = %q, want %q", tt.id, result, tt.expected)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  int
		expectErr bool
	}{
		{"valid integer", "42", 42, false},
		{"zero", "0", 0, false},
		{"negative", "-5", -5, false},
		{"with leading whitespace", " 10", 10, false}, // Sscanf handles leading whitespace
		{"invalid", "abc", 0, true},
		{"empty", "", 0, true},
		{"float", "3.14", 3, false}, // Sscanf parses the integer part
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseInt(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Errorf("parseInt(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("parseInt(%q) unexpected error: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("parseInt(%q) = %d, want %d", tt.input, result, tt.expected)
				}
			}
		})
	}
}

func TestFormatCreatedTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"valid RFC3339Nano",
			"2024-01-15T10:30:00.123456789Z",
			"2024-01-15 10:30:00",
		},
		{
			"valid with timezone",
			"2024-06-20T15:45:30.000000000+02:00",
			"2024-06-20 15:45:30",
		},
		{
			"invalid format",
			"not a date",
			"not a date", // Returns original on error
		},
		{
			"empty",
			"",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCreatedTime(tt.input)
			if result != tt.expected {
				t.Errorf("formatCreatedTime(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCalculateUptime(t *testing.T) {
	// Test with a time from the past
	pastTime := time.Now().Add(-2*time.Hour - 30*time.Minute).Format(time.RFC3339Nano)
	result := calculateUptime(pastTime)

	if !strings.Contains(result, "h") || !strings.Contains(result, "m") {
		t.Errorf("calculateUptime() for 2h30m ago = %q, expected to contain 'h' and 'm'", result)
	}

	// Test with invalid time
	result = calculateUptime("invalid")
	if result != "unknown" {
		t.Errorf("calculateUptime(invalid) = %q, want 'unknown'", result)
	}
}

func TestCalculateUptime_Days(t *testing.T) {
	// Test with a time from days ago
	pastTime := time.Now().Add(-3*24*time.Hour - 5*time.Hour - 30*time.Minute).Format(time.RFC3339Nano)
	result := calculateUptime(pastTime)

	if !strings.Contains(result, "d") {
		t.Errorf("calculateUptime() for 3+ days ago = %q, expected to contain 'd'", result)
	}
}

func TestCalculateUptime_Minutes(t *testing.T) {
	// Test with just minutes
	pastTime := time.Now().Add(-45 * time.Minute).Format(time.RFC3339Nano)
	result := calculateUptime(pastTime)

	if !strings.Contains(result, "m") {
		t.Errorf("calculateUptime() for 45m ago = %q, expected to contain 'm'", result)
	}
	// Should not contain 'd' or 'h' for less than an hour
	if strings.Contains(result, "d") || strings.Contains(result, "h") {
		t.Errorf("calculateUptime() for 45m = %q, should not contain 'd' or 'h'", result)
	}
}

func TestParsePortMappings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			"single mapping",
			"8080/tcp -> 0.0.0.0:8080",
			[]string{"8080/tcp -> 0.0.0.0:8080"},
		},
		{
			"multiple mappings",
			"8080/tcp -> 0.0.0.0:8080\n443/tcp -> 0.0.0.0:443",
			[]string{"8080/tcp -> 0.0.0.0:8080", "443/tcp -> 0.0.0.0:443"},
		},
		{
			"with empty lines",
			"8080/tcp -> 0.0.0.0:8080\n\n443/tcp -> 0.0.0.0:443\n",
			[]string{"8080/tcp -> 0.0.0.0:8080", "443/tcp -> 0.0.0.0:443"},
		},
		{
			"empty input",
			"",
			nil,
		},
		{
			"only whitespace",
			"  \n  \n  ",
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePortMappings(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parsePortMappings() returned %d items, want %d", len(result), len(tt.expected))
				return
			}
			for i, port := range result {
				if port != tt.expected[i] {
					t.Errorf("parsePortMappings()[%d] = %q, want %q", i, port, tt.expected[i])
				}
			}
		})
	}
}

func TestColorizeStatus(t *testing.T) {
	tests := []struct {
		status   string
		contains string
	}{
		{"running", "🟢"},
		{"Running", "🟢"},
		{"Up 2 hours (running)", "🟢"}, // Contains "running"
		{"exited", "🔴"},
		{"Exited (0)", "🔴"},
		{"created", "🟡"},
		{"paused", "🟡"},
		{"restarting", "🟡"},
		{"Up 2 hours", "🟡"}, // "Up" without "running" gets yellow
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := colorizeStatus(tt.status)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("colorizeStatus(%q) = %q, should contain %q", tt.status, result, tt.contains)
			}
			if !strings.Contains(result, tt.status) {
				t.Errorf("colorizeStatus(%q) = %q, should contain original status", tt.status, result)
			}
		})
	}
}

func TestColorizeHealth(t *testing.T) {
	tests := []struct {
		health   string
		contains string
	}{
		{"healthy", "🟢"},
		{"unhealthy", "🔴"},
		{"starting", "🟡"},
		{"none", "none"}, // Unknown status, no emoji
		{"N/A", "N/A"},
	}

	for _, tt := range tests {
		t.Run(tt.health, func(t *testing.T) {
			result := colorizeHealth(tt.health)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("colorizeHealth(%q) = %q, should contain %q", tt.health, result, tt.contains)
			}
		})
	}
}

func TestCenterText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected string
	}{
		{"short text", "Hi", 10, "    Hi    "},
		{"exact width", "Hello", 5, "Hello"},
		{"longer than width", "Hello World", 5, "Hello"},
		{"empty text", "", 10, "          "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := centerText(tt.text, tt.width)
			if result != tt.expected {
				t.Errorf("centerText(%q, %d) = %q, want %q", tt.text, tt.width, result, tt.expected)
			}
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected string
	}{
		{"short text", "Hi", 10, "Hi        "},
		{"exact width", "Hello", 5, "Hello"},
		{"longer than width", "Hello World", 5, "Hello World"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padRight(tt.text, tt.width)
			if result != tt.expected {
				t.Errorf("padRight(%q, %d) = %q, want %q", tt.text, tt.width, result, tt.expected)
			}
		})
	}
}

func TestPadRight_WithEmoji(t *testing.T) {
	// Emojis take 2 terminal columns
	text := "🟢 running"
	width := 15

	result := padRight(text, width)
	// The emoji counts as 2 in display width, so we need less padding
	// "🟢 running" = 2 + 1 + 7 = 10 display chars, need 5 spaces for width 15
	expectedPadding := 5
	actualPadding := len(result) - len(text)

	if actualPadding != expectedPadding {
		t.Errorf("padRight with emoji: got %d padding chars, want %d", actualPadding, expectedPadding)
	}
}

func TestStats_ToJSON(t *testing.T) {
	s := &Stats{
		Name:         "testcontainer",
		ID:           "abc123def456",
		Status:       "running",
		Health:       "healthy",
		Image:        "myapp:latest",
		Created:      "2024-01-15 10:30:00",
		Uptime:       "2h 30m",
		CPUPercent:   "2.5%",
		MemUsage:     "256MiB / 1GiB",
		MemPercent:   "25%",
		NetIO:        "1.2MB / 500KB",
		BlockIO:      "100MB / 50MB",
		PIDs:         "15",
		RestartCount: 0,
		Ports:        []string{"8080/tcp -> 0.0.0.0:8080"},
	}

	jsonStr, err := s.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	var output StatsJSON
	if err := json.Unmarshal([]byte(jsonStr), &output); err != nil {
		t.Fatalf("ToJSON() produced invalid JSON: %v", err)
	}

	// Verify structure
	if output.Container.Name != "testcontainer" {
		t.Errorf("Container.Name = %q, want %q", output.Container.Name, "testcontainer")
	}
	if output.Container.ID != "abc123def456" {
		t.Errorf("Container.ID = %q, want %q", output.Container.ID, "abc123def456")
	}
	if output.Container.Status != "running" {
		t.Errorf("Container.Status = %q, want %q", output.Container.Status, "running")
	}
	if output.Container.Health != "healthy" {
		t.Errorf("Container.Health = %q, want %q", output.Container.Health, "healthy")
	}
	if output.Resources.CPUPercent != "2.5%" {
		t.Errorf("Resources.CPUPercent = %q, want %q", output.Resources.CPUPercent, "2.5%")
	}
	if output.Resources.MemUsage != "256MiB / 1GiB" {
		t.Errorf("Resources.MemUsage = %q, want %q", output.Resources.MemUsage, "256MiB / 1GiB")
	}
	if len(output.Network.Ports) != 1 {
		t.Errorf("Network.Ports length = %d, want 1", len(output.Network.Ports))
	}
}

func TestStats_PrintStats_NoPanic(t *testing.T) {
	// Test that PrintStats doesn't panic with various inputs
	testCases := []struct {
		name string
		s    *Stats
	}{
		{"empty stats", &Stats{}},
		{"minimal stats", &Stats{Name: "test", ID: "abc123"}},
		{"with health", &Stats{Name: "test", Health: "healthy"}},
		{"with ports", &Stats{Name: "test", Ports: []string{"8080/tcp -> 0.0.0.0:8080"}}},
		{"with restart count", &Stats{Name: "test", RestartCount: 5}},
		{"full stats", &Stats{
			Name:         "test",
			ID:           "abc123def456",
			Status:       "running",
			Health:       "healthy",
			Image:        "myapp:latest",
			Created:      "2024-01-15 10:30:00",
			Uptime:       "2h 30m",
			CPUPercent:   "2.5%",
			MemUsage:     "256MiB / 1GiB",
			MemPercent:   "25%",
			NetIO:        "1.2MB / 500KB",
			BlockIO:      "100MB / 50MB",
			PIDs:         "15",
			RestartCount: 2,
			Ports:        []string{"8080/tcp -> 0.0.0.0:8080", "443/tcp -> 0.0.0.0:443"},
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PrintStats() panicked: %v", r)
				}
			}()
			tc.s.PrintStats()
		})
	}
}

func TestStatsJSONStructure(t *testing.T) {
	// Verify StatsJSON can be properly marshaled/unmarshaled
	output := StatsJSON{}
	output.Container.Name = "test"
	output.Container.ID = "abc123"
	output.Container.Image = "myapp:latest"
	output.Container.Status = "running"
	output.Container.Health = "healthy"
	output.Container.Created = "2024-01-15 10:30:00"
	output.Container.Uptime = "2h 30m"
	output.Resources.CPUPercent = "2.5%"
	output.Resources.MemUsage = "256MiB / 1GiB"
	output.Resources.MemPercent = "25%"
	output.Resources.PIDs = "15"
	output.Network.IO = "1.2MB / 500KB"
	output.Network.Ports = []string{"8080/tcp -> 0.0.0.0:8080"}
	output.Storage.BlockIO = "100MB / 50MB"
	output.RestartCount = 0

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal StatsJSON: %v", err)
	}

	var parsed StatsJSON
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal StatsJSON: %v", err)
	}

	if parsed.Container.Name != output.Container.Name {
		t.Errorf("Container.Name mismatch")
	}
}
