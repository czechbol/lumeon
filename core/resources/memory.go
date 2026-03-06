package resources

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Memory monitoring.
type MemoryStats struct {
	Total        uint64
	Used         uint64
	Free         uint64
	Available    uint64
	SwapTotal    uint64
	SwapUsed     uint64
	SwapFree     uint64
	Buffers      uint64
	Cached       uint64
	UsagePercent float64
}

type Memory interface {
	GetStats() (*MemoryStats, error)
}

type memoryImpl struct{}

func NewMemory() Memory {
	return &memoryImpl{}
}

func (m *memoryImpl) GetStats() (*MemoryStats, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stats := &MemoryStats{}
	fields := map[string]*uint64{
		"MemTotal:":     &stats.Total,
		"MemFree:":      &stats.Free,
		"MemAvailable:": &stats.Available,
		"Buffers:":      &stats.Buffers,
		"Cached:":       &stats.Cached,
		"SwapTotal:":    &stats.SwapTotal,
		"SwapFree:":     &stats.SwapFree,
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) < 2 {
			continue
		}

		ptr, ok := fields[parts[0]]
		if !ok {
			continue
		}

		value, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			continue
		}
		*ptr = value * 1024 // Convert KB to bytes
	}

	stats.Used = stats.Total - stats.Free - stats.Buffers - stats.Cached
	stats.SwapUsed = stats.SwapTotal - stats.SwapFree
	stats.UsagePercent = float64(stats.Used) / float64(stats.Total) * 100

	return stats, nil
}
