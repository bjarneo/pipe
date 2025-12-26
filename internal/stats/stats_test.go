package stats

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	s := New()

	if s == nil {
		t.Fatal("New() returned nil")
	}
	if s.StartTime.IsZero() {
		t.Error("New() should set StartTime")
	}
}

func TestStats_SetImageInfo(t *testing.T) {
	s := New()
	s.SetImageInfo("myapp", "v1.0.0")

	if s.ImageName != "myapp" {
		t.Errorf("SetImageInfo() ImageName = %q, want %q", s.ImageName, "myapp")
	}
	if s.ImageTag != "v1.0.0" {
		t.Errorf("SetImageInfo() ImageTag = %q, want %q", s.ImageTag, "v1.0.0")
	}
}

func TestStats_SetContainerInfo(t *testing.T) {
	s := New()
	s.SetContainerInfo("mycontainer", "example.com")

	if s.ContainerName != "mycontainer" {
		t.Errorf("SetContainerInfo() ContainerName = %q, want %q", s.ContainerName, "mycontainer")
	}
	if s.Host != "example.com" {
		t.Errorf("SetContainerInfo() Host = %q, want %q", s.Host, "example.com")
	}
}

func TestStats_SetLayerStats(t *testing.T) {
	s := New()
	s.SetLayerStats(10, 7, 3)

	if s.TotalLayers != 10 {
		t.Errorf("SetLayerStats() TotalLayers = %d, want %d", s.TotalLayers, 10)
	}
	if s.CachedLayers != 7 {
		t.Errorf("SetLayerStats() CachedLayers = %d, want %d", s.CachedLayers, 7)
	}
	if s.NewLayers != 3 {
		t.Errorf("SetLayerStats() NewLayers = %d, want %d", s.NewLayers, 3)
	}
}

func TestStats_SetImageSize(t *testing.T) {
	s := New()
	s.SetImageSize(1024 * 1024 * 100) // 100 MB

	if s.ImageSize != 104857600 {
		t.Errorf("SetImageSize() ImageSize = %d, want %d", s.ImageSize, 104857600)
	}
}

func TestStats_SetTransferredBytes(t *testing.T) {
	s := New()
	s.SetTransferredBytes(1024 * 1024 * 50) // 50 MB

	if s.TransferredBytes != 52428800 {
		t.Errorf("SetTransferredBytes() TransferredBytes = %d, want %d", s.TransferredBytes, 52428800)
	}
}

func TestStats_Finish(t *testing.T) {
	s := New()
	time.Sleep(10 * time.Millisecond) // Small delay to ensure time difference
	s.Finish()

	if s.EndTime.IsZero() {
		t.Error("Finish() should set EndTime")
	}
	if !s.EndTime.After(s.StartTime) {
		t.Error("EndTime should be after StartTime")
	}
}

func TestStats_Duration(t *testing.T) {
	s := New()
	time.Sleep(50 * time.Millisecond)

	// Before Finish, should use time.Since
	d1 := s.Duration()
	if d1 < 50*time.Millisecond {
		t.Errorf("Duration() before Finish should be >= 50ms, got %v", d1)
	}

	s.Finish()
	d2 := s.Duration()

	// After Finish, duration should be fixed
	time.Sleep(10 * time.Millisecond)
	d3 := s.Duration()

	if d2 != d3 {
		t.Errorf("Duration() after Finish should be constant, got %v and %v", d2, d3)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero bytes", 0, "0 B"},
		{"bytes", 500, "500 B"},
		{"exactly 1KB", 1024, "1.00 KB"},
		{"kilobytes", 1536, "1.50 KB"},
		{"megabytes", 1048576, "1.00 MB"},
		{"megabytes with decimal", 1572864, "1.50 MB"},
		{"gigabytes", 1073741824, "1.00 GB"},
		{"gigabytes with decimal", 1610612736, "1.50 GB"},
		{"terabytes", 1099511627776, "1.00 TB"},
		{"large value", 5368709120, "5.00 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"milliseconds", 500 * time.Millisecond, "500ms"},
		{"one second", time.Second, "1.0s"},
		{"seconds", 5500 * time.Millisecond, "5.5s"},
		{"one minute", time.Minute, "1m 0s"},
		{"minutes and seconds", 90 * time.Second, "1m 30s"},
		{"multiple minutes", 5*time.Minute + 30*time.Second, "5m 30s"},
		{"under a second", 100 * time.Millisecond, "100ms"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestStats_ToJSON(t *testing.T) {
	s := New()
	s.SetImageInfo("myapp", "v1.0.0")
	s.SetContainerInfo("mycontainer", "example.com")
	s.SetLayerStats(10, 7, 3)
	s.SetImageSize(104857600)       // 100 MB
	s.SetTransferredBytes(52428800) // 50 MB

	jsonStr, err := s.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Parse JSON to verify structure
	var output JSONOutput
	if err := json.Unmarshal([]byte(jsonStr), &output); err != nil {
		t.Fatalf("ToJSON() produced invalid JSON: %v", err)
	}

	// Verify fields
	if output.Image.Name != "myapp" {
		t.Errorf("ToJSON() Image.Name = %q, want %q", output.Image.Name, "myapp")
	}
	if output.Image.Tag != "v1.0.0" {
		t.Errorf("ToJSON() Image.Tag = %q, want %q", output.Image.Tag, "v1.0.0")
	}
	if output.Container.Name != "mycontainer" {
		t.Errorf("ToJSON() Container.Name = %q, want %q", output.Container.Name, "mycontainer")
	}
	if output.Container.Host != "example.com" {
		t.Errorf("ToJSON() Container.Host = %q, want %q", output.Container.Host, "example.com")
	}
	if output.Transfer.TotalLayers != 10 {
		t.Errorf("ToJSON() Transfer.TotalLayers = %d, want %d", output.Transfer.TotalLayers, 10)
	}
	if output.Transfer.CachedLayers != 7 {
		t.Errorf("ToJSON() Transfer.CachedLayers = %d, want %d", output.Transfer.CachedLayers, 7)
	}
	if output.Transfer.NewLayers != 3 {
		t.Errorf("ToJSON() Transfer.NewLayers = %d, want %d", output.Transfer.NewLayers, 3)
	}
	if output.Transfer.CachePercent != 70 {
		t.Errorf("ToJSON() Transfer.CachePercent = %f, want %f", output.Transfer.CachePercent, 70.0)
	}
	if output.Transfer.BandwidthSaved != 50 {
		t.Errorf("ToJSON() Transfer.BandwidthSaved = %f, want %f", output.Transfer.BandwidthSaved, 50.0)
	}
	if !output.Success {
		t.Error("ToJSON() Success should be true")
	}
}

func TestStats_ToJSON_ZeroLayers(t *testing.T) {
	s := New()
	s.SetLayerStats(0, 0, 0)

	jsonStr, err := s.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	var output JSONOutput
	if err := json.Unmarshal([]byte(jsonStr), &output); err != nil {
		t.Fatalf("ToJSON() produced invalid JSON: %v", err)
	}

	// Should not panic on division by zero, cache percent should be 0
	if output.Transfer.CachePercent != 0 {
		t.Errorf("ToJSON() with zero layers CachePercent = %f, want 0", output.Transfer.CachePercent)
	}
}

func TestStats_ToJSON_ZeroImageSize(t *testing.T) {
	s := New()
	s.SetImageSize(0)
	s.SetTransferredBytes(0)

	jsonStr, err := s.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	var output JSONOutput
	if err := json.Unmarshal([]byte(jsonStr), &output); err != nil {
		t.Fatalf("ToJSON() produced invalid JSON: %v", err)
	}

	// Should not panic on division by zero, bandwidth saved should be 0
	if output.Transfer.BandwidthSaved != 0 {
		t.Errorf("ToJSON() with zero size BandwidthSaved = %f, want 0", output.Transfer.BandwidthSaved)
	}
}

func TestJSONOutputStructure(t *testing.T) {
	// Test that JSONOutput can be marshaled and unmarshaled
	output := JSONOutput{}
	output.Image.Name = "test"
	output.Image.Tag = "latest"
	output.Container.Name = "testcontainer"
	output.Container.Host = "localhost"
	output.Transfer.TotalLayers = 5
	output.Transfer.CachedLayers = 3
	output.Transfer.NewLayers = 2
	output.Transfer.CachePercent = 60.0
	output.Transfer.ImageSizeBytes = 1000000
	output.Transfer.ImageSize = "1.00 MB"
	output.Transfer.TransferredBytes = 500000
	output.Transfer.Transferred = "500.00 KB"
	output.Transfer.BandwidthSaved = 50.0
	output.Timing.StartTime = "2024-01-01T00:00:00Z"
	output.Timing.EndTime = "2024-01-01T00:01:00Z"
	output.Timing.DurationMs = 60000
	output.Timing.Duration = "1m 0s"
	output.Success = true

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal JSONOutput: %v", err)
	}

	var parsed JSONOutput
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSONOutput: %v", err)
	}

	if parsed.Image.Name != output.Image.Name {
		t.Errorf("Image.Name mismatch: got %q, want %q", parsed.Image.Name, output.Image.Name)
	}
	if parsed.Success != output.Success {
		t.Errorf("Success mismatch: got %v, want %v", parsed.Success, output.Success)
	}
}

func TestStats_PrintSummary_NoPanic(t *testing.T) {
	// Test that PrintSummary doesn't panic with various inputs
	testCases := []struct {
		name string
		s    *Stats
	}{
		{"empty stats", New()},
		{"with image info", func() *Stats {
			s := New()
			s.SetImageInfo("app", "v1")
			return s
		}()},
		{"full stats", func() *Stats {
			s := New()
			s.SetImageInfo("app", "v1")
			s.SetContainerInfo("container", "host")
			s.SetLayerStats(10, 5, 5)
			s.SetImageSize(1000000)
			s.SetTransferredBytes(500000)
			return s
		}()},
		{"zero layers", func() *Stats {
			s := New()
			s.SetLayerStats(0, 0, 0)
			return s
		}()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Capture output to prevent it from going to test output
			// Just verify it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PrintSummary() panicked: %v", r)
				}
			}()
			tc.s.PrintSummary()
		})
	}
}

func TestFormatBytes_EdgeCases(t *testing.T) {
	// Test boundary conditions
	tests := []struct {
		bytes    int64
		contains string
	}{
		{1023, "B"},       // Just under 1KB
		{1024, "KB"},      // Exactly 1KB
		{1025, "KB"},      // Just over 1KB
		{1048575, "KB"},   // Just under 1MB
		{1048576, "MB"},   // Exactly 1MB
		{1073741823, "MB"}, // Just under 1GB
		{1073741824, "GB"}, // Exactly 1GB
	}

	for _, tt := range tests {
		result := FormatBytes(tt.bytes)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("FormatBytes(%d) = %q, should contain %q", tt.bytes, result, tt.contains)
		}
	}
}
