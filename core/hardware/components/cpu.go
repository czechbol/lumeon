package components

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	thermalZonePath = "/sys/class/thermal"
)

type CPU interface {
	GetAverageTemp() (float64, error)
}

type cpuImpl struct{}

func NewCPU() CPU {
	return &cpuImpl{}
}

func (c *cpuImpl) GetAverageTemp() (float64, error) {
	pattern := filepath.Join(thermalZonePath, "thermal_zone*", "temp")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return 0, fmt.Errorf("error finding thermal zones: %w", err)
	}

	if len(matches) == 0 {
		return 0, fmt.Errorf("no thermal zones found")
	}

	var totalTemp float64
	var count int

	for _, match := range matches {
		temp, err := readTemperature(match)
		if err != nil {
			// Log the error but continue with other sensors
			slog.Error("error reading temperature from %s: %v\n", "target", match, "error", err)
			continue
		}
		slog.Debug("read temperature", "target", match, "temperature", temp)

		totalTemp += temp
		count++
	}

	if count == 0 {
		return 0, fmt.Errorf("no valid temperature readings")
	}

	return totalTemp / float64(count), nil
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
