package resources

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/czechbol/lumeon/core/resources/dto"
	"golang.org/x/sys/unix"
)

const (
	AttrTemperature          = 194
	AttrReallocatedSectors   = 5
	AttrStartStopCount       = 4
	AttrUncorrectableSectors = 198
	AttrPendingSectors       = 197
	AttrTotalLBAWritten      = 241
)

type HDDStats struct {
	DeviceName  string
	Temperature float64
	TotalSize   uint64
	Partitions  []Partition
	SmartStatus SmartStatus
}

type Partition struct {
	Name       string
	Mountpoint string
	FsType     string
	Total      uint64
	Free       uint64
}

type SmartStatus struct {
	HealthOK            bool
	PowerOnHours        int
	PowerCycleCount     int
	ReallocatedSectors  int
	UncorrectableErrors int
	PendingSectors      int
	TerabytesWritten    int
}

type HDD interface {
	GetAverageTemp() (float64, error)
	GetStats() ([]HDDStats, error)
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

	total := 0
	count := 0

	for _, device := range devices {
		info, err := getDeviceSMARTInfo(device)
		if err != nil {
			slog.Warn("error getting temperature for device", "device", device, "error", err)
			continue
		}

		total += info.Temperature.Current
		count++
	}

	if count == 0 {
		slog.Error("no valid temperature readings found")
		return 0, ErrTemperatureNotFound
	}

	averageTemp := float64(total) / float64(count)
	slog.Debug("average temperature calculated", "averageTemp", averageTemp)
	return averageTemp, nil
}

func getStorageDevices() ([]*dto.BlockDevice, error) {
	cmd := exec.Command("lsblk", "-b", "-J", "-o", "NAME,SIZE,TYPE,MOUNTPOINT")
	output, err := cmd.Output()
	if err != nil {
		slog.Error("error executing lsblk", "error", err)
		return nil, err
	}

	var lsblkOutput dto.BlockDevices
	err = json.Unmarshal(output, &lsblkOutput)
	if err != nil {
		slog.Error("error unmarshalling JSON output", "error", err)
		return nil, err
	}

	devices := make([]*dto.BlockDevice, 0, len(lsblkOutput.BlockDevices))
	for _, device := range lsblkOutput.BlockDevices {
		if device.Type == "disk" && device.Size > 0 {
			devices = append(devices, device)
		}
	}

	if len(devices) == 0 {
		slog.Error("no storage devices found")
		return nil, ErrDriveNotMounted
	}

	slog.Debug("storage devices found", "devices", devices)
	return devices, nil
}

func getDeviceSMARTInfo(device *dto.BlockDevice) (*dto.SmartctlOutput, error) {
	cmdDiskType := "sat"
	if strings.HasPrefix(device.Name, "nvme") {
		cmdDiskType = "nvme"
	}

	cmd := exec.Command(
		"smartctl",
		"-d",
		cmdDiskType,
		"-A",
		filepath.Join("/dev", device.Name),
		"-j",
		"--info",
		"--health",
		"--attributes",
		"--tolerance=verypermissive",
		"--nocheck=standby",
		"--format=brief",
		"--log=error",
	) //nolint:gosec
	output, err := cmd.Output()
	if err != nil {
		slog.Error("error executing smartctl for device", "device", device, "error", err)
		return nil, err
	}

	var smartctlOutput dto.SmartctlOutput

	err = json.Unmarshal(output, &smartctlOutput)
	if err != nil {
		slog.Error("error unmarshalling JSON output", "error", err)
		return nil, err
	}

	if smartctlOutput.JSONFormatVersion[0] != 1 {
		slog.Error("unsupported JSON format version", "version", smartctlOutput.JSONFormatVersion)
		return nil, ErrSmartOutputVersionIncompatible
	}

	slog.Debug("device info found", "device", device, "info", smartctlOutput)
	return &smartctlOutput, nil
}

func (h *hddImpl) GetStats() ([]HDDStats, error) {
	devices, err := getStorageDevices()
	if err != nil {
		return nil, fmt.Errorf("error getting storage devices: %w", err)
	}

	stats := make([]HDDStats, 0, len(devices))
	for _, device := range devices {
		deviceInfo, err := getDeviceSMARTInfo(device)
		if err != nil {
			slog.Error("error getting stats for device", "device", device, "error", err)
			continue
		}

		deviceStats := &HDDStats{
			DeviceName:  device.Name,
			Temperature: float64(deviceInfo.Temperature.Current),
			TotalSize:   device.Size,
		}

		populateSMART(deviceStats, deviceInfo)

		deviceStats.Partitions = populatePartitions(device)

		stats = append(stats, *deviceStats)
	}

	if len(stats) == 0 {
		return nil, fmt.Errorf("no valid device stats found")
	}

	return stats, nil
}

func populateSMART(stats *HDDStats, deviceInfo *dto.SmartctlOutput) {
	reallocSectorsAttr := getSMARTAttribute(deviceInfo, AttrReallocatedSectors)
	reallocatedSectors := 0
	if reallocSectorsAttr != nil {
		reallocatedSectors = reallocSectorsAttr.Raw.Value
	}

	uncorrSecAttr := getSMARTAttribute(deviceInfo, AttrUncorrectableSectors)
	uncorrectableSectors := 0
	if uncorrSecAttr != nil {
		uncorrectableSectors = uncorrSecAttr.Raw.Value
	}

	pendingSectorsAttr := getSMARTAttribute(deviceInfo, AttrPendingSectors)
	pendingSectors := 0
	if pendingSectorsAttr != nil {
		pendingSectors = pendingSectorsAttr.Raw.Value
	}

	lbaWrittenAttr := getSMARTAttribute(deviceInfo, AttrTotalLBAWritten)
	lbaWritten := 0
	if lbaWrittenAttr != nil {
		lbaWritten = lbaWrittenAttr.Raw.Value
	}

	tbw := lbaWritten * deviceInfo.LogicalBlockSize / 1_000_000_000_000

	stats.SmartStatus = SmartStatus{
		HealthOK:            deviceInfo.SmartStatus.Passed,
		PowerOnHours:        deviceInfo.PowerOnTime.Hours,
		PowerCycleCount:     deviceInfo.PowerCycleCount,
		ReallocatedSectors:  reallocatedSectors,
		UncorrectableErrors: uncorrectableSectors,
		PendingSectors:      pendingSectors,
		TerabytesWritten:    tbw,
	}

}

func getSMARTAttribute(deviceInfo *dto.SmartctlOutput, id int) *dto.Attribute {
	for _, attr := range deviceInfo.AtaSmartAttributes.Table {
		if attr.ID == id {
			return &attr
		}
	}

	return nil
}

func populatePartitions(device *dto.BlockDevice) []Partition {
	partitions := make([]Partition, 0, len(device.Children))
	for _, part := range device.Children {
		var stat unix.Statfs_t
		err := unix.Statfs(filepath.Join("/dev", part.Name), &stat)
		if err != nil {
			slog.Error("error getting partition stats", "partition", part, "error", err)
			continue
		}

		partition := Partition{
			Name:       part.Name,
			Mountpoint: part.MountPoint,
			FsType:     part.Type,
			Total:      part.Size,
			Free:       stat.Bfree * uint64(stat.Bsize),
		}

		partitions = append(partitions, partition)

	}

	return partitions
}
