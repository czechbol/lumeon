// SPDX-License-Identifier: MPL-2.0
// Based on https://codeberg.org/pancake/neon (MPL-2.0)

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
	hddCacheTTL              = 20 * time.Second
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

type hddImpl struct {
	mu          sync.RWMutex
	cachedStats []HDDStats
	cacheTime   time.Time
	cacheTTL    time.Duration
}

func NewHDD() HDD {
	return &hddImpl{
		cacheTTL: hddCacheTTL,
	}
}

func (h *hddImpl) getOrRefresh() ([]HDDStats, error) {
	h.mu.RLock()
	if h.cachedStats != nil && time.Since(h.cacheTime) < h.cacheTTL {
		stats := make([]HDDStats, len(h.cachedStats))
		copy(stats, h.cachedStats)
		h.mu.RUnlock()
		return stats, nil
	}
	h.mu.RUnlock()

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
		return nil, ErrNoValidDeviceStats
	}

	h.mu.Lock()
	h.cachedStats = stats
	h.cacheTime = time.Now()
	h.mu.Unlock()

	result := make([]HDDStats, len(stats))
	copy(result, stats)
	return result, nil
}

func (h *hddImpl) GetAverageTemp() (float64, error) {
	stats, err := h.getOrRefresh()
	if err != nil {
		slog.Error("error getting storage stats", "error", err)
		return 0, err
	}

	total := 0.0
	count := 0

	for _, stat := range stats {
		if stat.Temperature > 0 {
			total += stat.Temperature
			count++
		}
	}

	if count == 0 {
		slog.Error("no valid temperature readings found")
		return 0, ErrTemperatureNotFound
	}

	averageTemp := total / float64(count)
	slog.Debug("average temperature calculated", "averageTemp", averageTemp)
	return averageTemp, nil
}

func getStorageDevices() ([]*dto.BlockDevice, error) {
	cmd := exec.CommandContext(context.Background(), "lsblk", "-b", "-J", "-o", "NAME,SIZE,TYPE,MOUNTPOINT")
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

	//nolint:gosec // cmdDiskType is hardcoded to "sat" or "nvme"; device.Name is from trusted lsblk output
	cmd := exec.CommandContext(
		context.Background(),
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
	)
	output, err := cmd.Output()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
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

	if smartctlOutput.Smartctl.ExitStatus&0x07 != 0 {
		return nil, fmt.Errorf("smartctl failed for device %s: exit status %d: %w",
			device.Name, smartctlOutput.Smartctl.ExitStatus, ErrSmartctlFailed)
	}

	slog.Debug("device info found", "device", device, "info", smartctlOutput)
	return &smartctlOutput, nil
}

func (h *hddImpl) GetStats() ([]HDDStats, error) {
	return h.getOrRefresh()
}

func populateSMART(stats *HDDStats, deviceInfo *dto.SmartctlOutput) {
	reallocSectorsAttr := getSMARTAttribute(deviceInfo, AttrReallocatedSectors)
	reallocatedSectors := 0
	if reallocSectorsAttr != nil {
		reallocatedSectors = int(reallocSectorsAttr.Raw.Value)
	}

	uncorrSecAttr := getSMARTAttribute(deviceInfo, AttrUncorrectableSectors)
	uncorrectableSectors := 0
	if uncorrSecAttr != nil {
		uncorrectableSectors = int(uncorrSecAttr.Raw.Value)
	}

	pendingSectorsAttr := getSMARTAttribute(deviceInfo, AttrPendingSectors)
	pendingSectors := 0
	if pendingSectorsAttr != nil {
		pendingSectors = int(pendingSectorsAttr.Raw.Value)
	}

	lbaWrittenAttr := getSMARTAttribute(deviceInfo, AttrTotalLBAWritten)
	lbaWritten := int64(0)
	if lbaWrittenAttr != nil {
		lbaWritten = lbaWrittenAttr.Raw.Value
	}

	tbw := int(lbaWritten * int64(deviceInfo.LogicalBlockSize) / 1_000_000_000_000)

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
		if part.MountPoint == "" {
			continue // skip unmounted partitions
		}
		var stat unix.Statfs_t
		err := unix.Statfs(part.MountPoint, &stat)
		if err != nil {
			slog.Error("error getting partition stats", "partition", part, "error", err)
			continue
		}

		partition := Partition{
			Name:       part.Name,
			Mountpoint: part.MountPoint,
			FsType:     part.Type,
			Total:      part.Size,
			Free:       stat.Bfree * uint64(stat.Bsize), //nolint:gosec // Bsize is a block size, always positive
		}

		partitions = append(partitions, partition)
	}

	return partitions
}
