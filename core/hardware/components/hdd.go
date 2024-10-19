package components

import (
	"encoding/json"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/czechbol/lumeon/core/hardware"
	"github.com/czechbol/lumeon/core/hardware/dto"
)

type HDD interface {
	GetAverageTemp() (float64, error)
}

type hddImpl struct{}

func NewHDD() HDD {
	return &hddImpl{}
}

func (h *hddImpl) GetAverageTemp() (float64, error) {
	// Get a list of all storage devices
	devices, err := getStorageDevices()
	if err != nil {
		slog.Error("error getting storage devices", "error", err)
		return 0, err
	}

	total := 0.0
	count := 0

	for _, device := range devices {
		temp, err := getDeviceTemperature(device)
		if err != nil {
			slog.Warn("error getting temperature for device", "device", device, "error", err)
			continue
		}

		total += temp
		count++
	}

	if count == 0 {
		slog.Error("no valid temperature readings found")
		return 0, hardware.ErrTemperatureNotFound
	}

	averageTemp := total / float64(count)
	slog.Debug("average temperature calculated", "averageTemp", averageTemp)
	return averageTemp, nil
}

func getStorageDevices() ([]string, error) {
	cmd := exec.Command("lsblk", "-ndo", "NAME")
	output, err := cmd.Output()
	if err != nil {
		slog.Error("error executing lsblk", "error", err)
		return nil, err
	}

	devices := strings.Split(strings.TrimSpace(string(output)), "\n")
	slog.Debug("storage devices found", "devices", devices)
	return devices, nil
}

func getDeviceTemperature(device string) (float64, error) {
	cmd := exec.Command("smartctl", "-d", "sat", "-A", "/dev/"+device, "-j")
	output, err := cmd.Output()
	if err != nil {
		slog.Error("error executing smartctl for device", "device", device, "error", err)
		return 0, err
	}

	var smartctlOutput dto.SmartctlOutput

	err = json.Unmarshal(output, &smartctlOutput)
	if err != nil {
		slog.Error("error unmarshalling JSON output", "error", err)
		return 0, err
	}

	temp := smartctlOutput.Temperature.Current

	slog.Debug("device temperature found", "device", device, "temperature", temp)
	return float64(temp), nil
}
