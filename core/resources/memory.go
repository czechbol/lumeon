package resources

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Memory monitoring
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
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		// Convert KB to bytes
		value *= 1024

		switch fields[0] {
		case "MemTotal:":
			stats.Total = value
		case "MemFree:":
			stats.Free = value
		case "MemAvailable:":
			stats.Available = value
		case "Buffers:":
			stats.Buffers = value
		case "Cached:":
			stats.Cached = value
		case "SwapTotal:":
			stats.SwapTotal = value
		case "SwapFree:":
			stats.SwapFree = value
		}
	}

	stats.Used = stats.Total - stats.Free - stats.Buffers - stats.Cached
	stats.SwapUsed = stats.SwapTotal - stats.SwapFree
	stats.UsagePercent = float64(stats.Used) / float64(stats.Total) * 100

	return stats, nil
}
