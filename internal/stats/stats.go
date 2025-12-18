package stats

import (
	"fmt"
	"strings"
	"time"
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
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
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

// PrintSummary prints a pretty deployment summary
func (s *Stats) PrintSummary() {
	s.Finish()

	width := 50
	line := strings.Repeat("─", width)
	doubleLine := strings.Repeat("═", width)

	fmt.Println()
	fmt.Printf("╔%s╗\n", doubleLine)
	fmt.Printf("║%s║\n", centerText("DEPLOYMENT SUMMARY", width))
	fmt.Printf("╠%s╣\n", doubleLine)

	// Image info
	fmt.Printf("║%s║\n", padRight(fmt.Sprintf("  Image: %s:%s", s.ImageName, s.ImageTag), width))
	fmt.Printf("║%s║\n", padRight(fmt.Sprintf("  Container: %s", s.ContainerName), width))
	fmt.Printf("║%s║\n", padRight(fmt.Sprintf("  Host: %s", s.Host), width))

	fmt.Printf("╟%s╢\n", line)

	// Transfer stats
	fmt.Printf("║%s║\n", centerText("Transfer Statistics", width))
	fmt.Printf("╟%s╢\n", line)

	if s.TotalLayers > 0 {
		cachePercent := float64(s.CachedLayers) / float64(s.TotalLayers) * 100
		fmt.Printf("║%s║\n", padRight(fmt.Sprintf("  Total Layers: %d", s.TotalLayers), width))
		fmt.Printf("║%s║\n", padRight(fmt.Sprintf("  Cached Layers: %d (%.0f%%)", s.CachedLayers, cachePercent), width))
		fmt.Printf("║%s║\n", padRight(fmt.Sprintf("  New Layers: %d", s.NewLayers), width))
	}

	if s.ImageSize > 0 {
		fmt.Printf("║%s║\n", padRight(fmt.Sprintf("  Image Size: %s", FormatBytes(s.ImageSize)), width))
	}

	if s.TransferredBytes > 0 {
		fmt.Printf("║%s║\n", padRight(fmt.Sprintf("  Data Transferred: %s", FormatBytes(s.TransferredBytes)), width))
		if s.ImageSize > 0 {
			savings := float64(s.ImageSize-s.TransferredBytes) / float64(s.ImageSize) * 100
			if savings > 0 {
				fmt.Printf("║%s║\n", padRight(fmt.Sprintf("  Bandwidth Saved: %.1f%%", savings), width))
			}
		}
	}

	fmt.Printf("╟%s╢\n", line)

	// Time stats
	fmt.Printf("║%s║\n", centerText("Timing", width))
	fmt.Printf("╟%s╢\n", line)
	fmt.Printf("║%s║\n", padRight(fmt.Sprintf("  Duration: %s", FormatDuration(s.Duration())), width))
	fmt.Printf("║%s║\n", padRight(fmt.Sprintf("  Completed: %s", s.EndTime.Format("15:04:05")), width))

	fmt.Printf("╚%s╝\n", doubleLine)
	fmt.Println()
}

// centerText centers text within a given width
func centerText(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text + strings.Repeat(" ", width-len(text)-padding)
}

// padRight pads text to the right to fill width
func padRight(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	return text + strings.Repeat(" ", width-len(text))
}
