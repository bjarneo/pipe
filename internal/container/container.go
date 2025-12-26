package container

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bjarneo/pipe/internal/config"
	"github.com/bjarneo/pipe/internal/format"
	"github.com/bjarneo/pipe/internal/logger"
	"github.com/bjarneo/pipe/internal/ssh"
)

// Display and formatting constants
const (
	// statsDisplayWidth is the width of the container stats display box
	statsDisplayWidth = 54

	// containerIDDisplayLen is the length to truncate container IDs to
	containerIDDisplayLen = 12
)

// Stats holds container statistics
type Stats struct {
	Name         string
	ID           string
	Status       string
	Health       string
	Image        string
	Created      string
	Uptime       string
	CPUPercent   string
	MemUsage     string
	MemPercent   string
	NetIO        string
	BlockIO      string
	PIDs         string
	RestartCount int
	Ports        []string
}

// StatsJSON is the JSON output format
type StatsJSON struct {
	Container struct {
		Name    string `json:"name"`
		ID      string `json:"id"`
		Image   string `json:"image"`
		Status  string `json:"status"`
		Health  string `json:"health,omitempty"`
		Created string `json:"created"`
		Uptime  string `json:"uptime"`
	} `json:"container"`
	Resources struct {
		CPUPercent string `json:"cpuPercent"`
		MemUsage   string `json:"memUsage"`
		MemPercent string `json:"memPercent"`
		PIDs       string `json:"pids"`
	} `json:"resources"`
	Network struct {
		IO    string   `json:"io"`
		Ports []string `json:"ports"`
	} `json:"network"`
	Storage struct {
		BlockIO string `json:"blockIO"`
	} `json:"storage"`
	RestartCount int `json:"restartCount"`
}

// GetStats retrieves container statistics from the remote host
func GetStats(cfg *config.Config, log *logger.Logger) (*Stats, error) {
	return GetStatsWithContext(context.Background(), cfg, log)
}

// GetStatsWithContext retrieves container statistics from the remote host with context support
func GetStatsWithContext(ctx context.Context, cfg *config.Config, log *logger.Logger) (*Stats, error) {
	stats := &Stats{Name: cfg.ContainerName}

	// Get container inspect data - health check is optional so we get it separately
	inspectCmd := fmt.Sprintf(`%s 'docker inspect --format "{{.Id}}|{{.State.Status}}|{{.Config.Image}}|{{.Created}}|{{.RestartCount}}" %s'`,
		ssh.GetCommand(cfg), cfg.ContainerName)

	result, err := ssh.ExecuteCommandContext(ctx, log, inspectCmd, "Getting container info")
	if err != nil {
		return nil, fmt.Errorf("container '%s' not found on %s", cfg.ContainerName, cfg.Host)
	}

	output := strings.TrimSpace(result.Stdout)
	if output == "" || strings.Contains(output, "No such object") || strings.Contains(output, "Error") {
		return nil, fmt.Errorf("container '%s' not found on %s", cfg.ContainerName, cfg.Host)
	}

	parts := strings.Split(output, "|")
	if len(parts) >= 5 {
		stats.ID = truncateID(parts[0])
		stats.Status = parts[1]
		stats.Image = parts[2]
		stats.Created = formatCreatedTime(parts[3])
		stats.Uptime = calculateUptime(parts[3])
		if restartCount, err := parseInt(parts[4]); err == nil {
			stats.RestartCount = restartCount
		}
	}

	// Get health status separately (may not exist)
	healthCmd := fmt.Sprintf(`%s 'docker inspect --format "{{if .State.Health}}{{.State.Health.Status}}{{else}}N/A{{end}}" %s'`,
		ssh.GetCommand(cfg), cfg.ContainerName)
	if healthResult, err := ssh.ExecuteCommandContext(ctx, log, healthCmd, "Getting health status"); err == nil {
		stats.Health = strings.TrimSpace(healthResult.Stdout)
		if stats.Health == "" {
			stats.Health = "N/A"
		}
	} else {
		stats.Health = "N/A"
	}

	// Get port mappings
	portsCmd := fmt.Sprintf(`%s 'docker port %s'`, ssh.GetCommand(cfg), cfg.ContainerName)
	if portResult, err := ssh.ExecuteCommandContext(ctx, log, portsCmd, "Getting port mappings"); err == nil {
		stats.Ports = parsePortMappings(portResult.Stdout)
	}

	// Get live stats (CPU, Memory, Network, Block I/O)
	statsCmd := fmt.Sprintf(`%s 'docker stats --no-stream --format "{{.CPUPerc}}|{{.MemUsage}}|{{.MemPerc}}|{{.NetIO}}|{{.BlockIO}}|{{.PIDs}}" %s'`,
		ssh.GetCommand(cfg), cfg.ContainerName)

	if statsResult, err := ssh.ExecuteCommandContext(ctx, log, statsCmd, "Getting container stats"); err == nil {
		statsParts := strings.Split(strings.TrimSpace(statsResult.Stdout), "|")
		if len(statsParts) >= 6 {
			stats.CPUPercent = statsParts[0]
			stats.MemUsage = statsParts[1]
			stats.MemPercent = statsParts[2]
			stats.NetIO = statsParts[3]
			stats.BlockIO = statsParts[4]
			stats.PIDs = statsParts[5]
		}
	}

	return stats, nil
}

// ToJSON converts stats to JSON string
func (s *Stats) ToJSON() (string, error) {
	output := StatsJSON{}
	output.Container.Name = s.Name
	output.Container.ID = s.ID
	output.Container.Image = s.Image
	output.Container.Status = s.Status
	output.Container.Health = s.Health
	output.Container.Created = s.Created
	output.Container.Uptime = s.Uptime
	output.Resources.CPUPercent = s.CPUPercent
	output.Resources.MemUsage = s.MemUsage
	output.Resources.MemPercent = s.MemPercent
	output.Resources.PIDs = s.PIDs
	output.Network.IO = s.NetIO
	output.Network.Ports = s.Ports
	output.Storage.BlockIO = s.BlockIO
	output.RestartCount = s.RestartCount

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// PrintStats displays container stats in a nice format
func (s *Stats) PrintStats() {
	width := statsDisplayWidth
	line := strings.Repeat("─", width)
	doubleLine := strings.Repeat("═", width)

	fmt.Println()
	fmt.Printf("╔%s╗\n", doubleLine)
	fmt.Printf("║%s║\n", format.CenterText("CONTAINER STATS", width))
	fmt.Printf("╠%s╣\n", doubleLine)

	// Container info
	fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  Name: %s", s.Name), width))
	fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  ID: %s", s.ID), width))
	fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  Image: %s", s.Image), width))
	fmt.Printf("║%s║\n", format.PadRightWithEmoji(fmt.Sprintf("  Status: %s", colorizeStatus(s.Status)), width))
	if s.Health != "N/A" {
		fmt.Printf("║%s║\n", format.PadRightWithEmoji(fmt.Sprintf("  Health: %s", colorizeHealth(s.Health)), width))
	}
	fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  Created: %s", s.Created), width))
	fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  Uptime: %s", s.Uptime), width))
	if s.RestartCount > 0 {
		fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  Restarts: %d", s.RestartCount), width))
	}

	fmt.Printf("╟%s╢\n", line)

	// Resource usage
	fmt.Printf("║%s║\n", format.CenterText("Resource Usage", width))
	fmt.Printf("╟%s╢\n", line)
	fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  CPU: %s", s.CPUPercent), width))
	fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  Memory: %s (%s)", s.MemUsage, s.MemPercent), width))
	fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  PIDs: %s", s.PIDs), width))

	fmt.Printf("╟%s╢\n", line)

	// Network
	fmt.Printf("║%s║\n", format.CenterText("Network", width))
	fmt.Printf("╟%s╢\n", line)
	fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  I/O: %s", s.NetIO), width))
	if len(s.Ports) > 0 {
		fmt.Printf("║%s║\n", format.PadRight("  Ports:", width))
		for _, port := range s.Ports {
			fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("    %s", port), width))
		}
	}

	fmt.Printf("╟%s╢\n", line)

	// Storage
	fmt.Printf("║%s║\n", format.CenterText("Storage", width))
	fmt.Printf("╟%s╢\n", line)
	fmt.Printf("║%s║\n", format.PadRight(fmt.Sprintf("  Block I/O: %s", s.BlockIO), width))

	fmt.Printf("╚%s╝\n", doubleLine)
	fmt.Println()
}

// Helper functions

func truncateID(id string) string {
	if len(id) > containerIDDisplayLen {
		return id[:containerIDDisplayLen]
	}
	return id
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

func formatCreatedTime(created string) string {
	t, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return created
	}
	return t.Format("2006-01-02 15:04:05")
}

func calculateUptime(created string) string {
	t, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return "unknown"
	}

	duration := time.Since(t)

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func parsePortMappings(output string) []string {
	var ports []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			ports = append(ports, line)
		}
	}
	return ports
}

func colorizeStatus(status string) string {
	if strings.Contains(strings.ToLower(status), "running") {
		return "🟢 " + status
	}
	if strings.Contains(strings.ToLower(status), "exited") {
		return "🔴 " + status
	}
	return "🟡 " + status
}

func colorizeHealth(health string) string {
	switch strings.ToLower(health) {
	case "healthy":
		return "🟢 " + health
	case "unhealthy":
		return "🔴 " + health
	case "starting":
		return "🟡 " + health
	default:
		return health
	}
}

