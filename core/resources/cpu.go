package resources

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
)

const (
	thermalZonePath = "/sys/class/thermal"
	procStatPath    = "/proc/stat"
	procCPUInfo     = "/proc/cpuinfo"
)

type CPUStats struct {
	UsagePercent   float64
	AvgTemperature float64
	CoreCount      int
	Cores          []CoreStats
}

type CoreStats struct {
	ID           int
	UsagePercent float64
	MaxFrequency float64 // MHz
}

type CPU interface {
	GetAverageTemp() (float64, error)
	GetStats() (*CPUStats, error)
}

type cpuImpl struct {
}

func NewCPU() CPU {
	return &cpuImpl{}
}

func (c *cpuImpl) GetAverageTemp() (float64, error) {
	temps, err := c.getAllTemps()
	if err != nil {
		return 0, fmt.Errorf("error getting temperatures: %w", err)
	}

	if len(temps) == 0 {
		return 0, ErrNoValidTemperature
	}

	totalTemp := 0.0

	for _, temp := range temps {
		totalTemp += temp
	}

	return totalTemp / float64(len(temps)), nil
}

func (c *cpuImpl) getAllTemps() (map[string]float64, error) {
	pattern := filepath.Join(thermalZonePath, "thermal_zone*", "temp")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("error finding thermal zones: %w", err)
	}

	if len(matches) == 0 {
		return nil, ErrNoThermalZones
	}

	temps := make(map[string]float64)

	for _, match := range matches {
		temp, err := readTemperature(match)
		if err != nil {
			// Log the error but continue with other sensors
			slog.Error("error reading temperature from %s: %v\n", "target", match, "error", err)
			continue
		}
		slog.Debug("read temperature", "target", match, "temperature", temp)

		temps[match] = temp
	}

	return temps, nil
}

func readTemperature(path string) (float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	// Temperature is typically reported in milliseconds
	temp, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil {
		return 0, err
	}

	// Convert to degrees Celsius
	return temp / 1000.0, nil
}

// GetStats returns comprehensive CPU statistics
func (c *cpuImpl) GetStats() (*CPUStats, error) {
	percentages, err := cpu.Percent(time.Second*5, true)
	if err != nil {
		return nil, err
	}

	info, err := cpu.Info()
	if err != nil {
		return nil, err
	}

	var cores []CoreStats
	for i, percent := range percentages {
		cores = append(cores, CoreStats{
			ID:           i,
			UsagePercent: percent,
			MaxFrequency: info[i].Mhz,
		})
	}

	avgPercent, err := cpu.Percent(0, false)
	if err != nil {
		return nil, err
	}

	avgTemp, err := c.GetAverageTemp()
	if err != nil {
		return nil, err
	}

	return &CPUStats{
		UsagePercent:   avgPercent[0],
		AvgTemperature: avgTemp,
		CoreCount:      len(cores),
		Cores:          cores,
	}, nil
}
