package stats

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bjarneo/pipe/internal/format"
)

// Display and formatting constants
const (
	// summaryDisplayWidth is the width of the deployment summary box
	summaryDisplayWidth = 50

	// bytesPerKilobyte is the base unit for byte size formatting
	bytesPerKilobyte = 1024
)

// Stats tracks deployment statistics
type Stats struct {
	StartTime        time.Time
	EndTime          time.Time
	ImageSize        int64
	TransferredBytes int64
	TotalLayers      int
	CachedLayers     int
	NewLayers        int
	ImageName        string
	ImageTag         string
	ContainerName    string
	Host             string
}

// New creates a new Stats instance
func New() *Stats {
	return &Stats{
		StartTime: time.Now(),
	}
}

// SetImageInfo sets the image information
func (s *Stats) SetImageInfo(name, tag string) {
	s.ImageName = name
	s.ImageTag = tag
}

// SetContainerInfo sets container and host information
func (s *Stats) SetContainerInfo(containerName, host string) {
	s.ContainerName = containerName
	s.Host = host
}

// SetLayerStats sets layer statistics
func (s *Stats) SetLayerStats(total, cached, new int) {
	s.TotalLayers = total
	s.CachedLayers = cached
	s.NewLayers = new
}

// SetImageSize sets the image size in bytes
func (s *Stats) SetImageSize(size int64) {
	s.ImageSize = size
}

// SetTransferredBytes sets the actual bytes transferred
func (s *Stats) SetTransferredBytes(bytes int64) {
	s.TransferredBytes = bytes
}

// Finish marks the end of the deployment
func (s *Stats) Finish() {
	s.EndTime = time.Now()
}

// Duration returns the deployment duration
func (s *Stats) Duration() time.Duration {
	if s.EndTime.IsZero() {
		return time.Since(s.StartTime)
	}
	return s.EndTime.Sub(s.StartTime)
}

// FormatBytes formats bytes into human readable format
func FormatBytes(bytes int64) string {
	if bytes < bytesPerKilobyte {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(bytesPerKilobyte), 0
	for n := bytes / bytesPerKilobyte; n >= bytesPerKilobyte; n /= bytesPerKilobyte {
		div *= bytesPerKilobyte
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration formats duration in a pretty way
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}

// JSONOutput represents the JSON output structure for stats
type JSONOutput struct {
	Image struct {
		Name string `json:"name"`
		Tag  string `json:"tag"`
	} `json:"image"`
	Container struct {
		Name string `json:"name"`
		Host string `json:"host"`
	} `json:"container"`
	Transfer struct {
		TotalLayers      int    `json:"totalLayers"`
		CachedLayers     int    `json:"cachedLayers"`
		NewLayers        int    `json:"newLayers"`
		CachePercent     float64 `json:"cachePercent"`
		ImageSizeBytes   int64  `json:"imageSizeBytes"`
		ImageSize        string `json:"imageSize"`
		TransferredBytes int64  `json:"transferredBytes"`
		Transferred      string `json:"transferred"`
		BandwidthSaved   float64 `json:"bandwidthSavedPercent"`
	} `json:"transfer"`
	Timing struct {
		StartTime   string  `json:"startTime"`
		EndTime     string  `json:"endTime"`
		DurationMs  int64   `json:"durationMs"`
		Duration    string  `json:"duration"`
	} `json:"timing"`
	Success bool `json:"success"`
}

// ToJSON returns the stats as a JSON string
func (s *Stats) ToJSON() (string, error) {
	s.Finish()

	output := JSONOutput{}
	output.Image.Name = s.ImageName
	output.Image.Tag = s.ImageTag
	output.Container.Name = s.ContainerName
	output.Container.Host = s.Host
	output.Transfer.TotalLayers = s.TotalLayers
	output.Transfer.CachedLayers = s.CachedLayers
	output.Transfer.NewLayers = s.NewLayers
	if s.TotalLayers > 0 {
		output.Transfer.CachePercent = float64(s.CachedLayers) / float64(s.TotalLayers) * 100
	}
	output.Transfer.ImageSizeBytes = s.ImageSize
	output.Transfer.ImageSize = FormatBytes(s.ImageSize)
	output.Transfer.TransferredBytes = s.TransferredBytes
	output.Transfer.Transferred = FormatBytes(s.TransferredBytes)
	if s.ImageSize > 0 && s.TransferredBytes > 0 {
		output.Transfer.BandwidthSaved = float64(s.ImageSize-s.TransferredBytes) / float64(s.ImageSize) * 100
	}
	output.Timing.StartTime = s.StartTime.Format(time.RFC3339)
	output.Timing.EndTime = s.EndTime.Format(time.RFC3339)
	output.Timing.DurationMs = s.Duration().Milliseconds()
	output.Timing.Duration = FormatDuration(s.Duration())
	output.Success = true

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// PrintSummary prints a pretty deployment summary
func (s *Stats) PrintSummary() {
	s.Finish()

	width := summaryDisplayWidth
	line := strings.Repeat("─", width)
	doubleLine := strings.Repeat("═", width)

	fmt.Println()
	fmt.Printf("╔%s╗\n", doubleLine)
	fmt.Printf("║%s║\n", format.CenterText("DEPLOYMENT SUMMARY", width))
	fmt.Printf("╠%s╣\n", doubleLine)

	// Image info
	fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  Image: %s:%s", s.ImageName, s.ImageTag), width))
	fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  Container: %s", s.ContainerName), width))
	fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  Host: %s", s.Host), width))

	fmt.Printf("╟%s╢\n", line)

	// Transfer stats
	fmt.Printf("║%s║\n", format.CenterText("Transfer Statistics", width))
	fmt.Printf("╟%s╢\n", line)

	if s.TotalLayers > 0 {
		cachePercent := float64(s.CachedLayers) / float64(s.TotalLayers) * 100
		fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  Total Layers: %d", s.TotalLayers), width))
		fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  Cached Layers: %d (%.0f%%)", s.CachedLayers, cachePercent), width))
		fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  New Layers: %d", s.NewLayers), width))
	}

	if s.ImageSize > 0 {
		fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  Image Size: %s", FormatBytes(s.ImageSize)), width))
	}

	if s.TransferredBytes > 0 {
		fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  Data Transferred: %s", FormatBytes(s.TransferredBytes)), width))
		if s.ImageSize > 0 {
			savings := float64(s.ImageSize-s.TransferredBytes) / float64(s.ImageSize) * 100
			if savings > 0 {
				fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  Bandwidth Saved: %.1f%%", savings), width))
			}
		}
	}

	fmt.Printf("╟%s╢\n", line)

	// Time stats
	fmt.Printf("║%s║\n", format.CenterText("Timing", width))
	fmt.Printf("╟%s╢\n", line)
	fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  Duration: %s", FormatDuration(s.Duration())), width))
	fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  Completed: %s", s.EndTime.Format("15:04:05")), width))

	fmt.Printf("╚%s╝\n", doubleLine)
	fmt.Println()
}
