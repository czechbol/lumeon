package resources

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Network monitoring
type NetworkStats struct {
	Interface       string
	BytesReceived   uint64
	BytesSent       uint64
	PacketsReceived uint64
	PacketsSent     uint64
	ReceiveSpeed    float64 // bytes per second
	SendSpeed       float64 // bytes per second
	Errors          uint64
	Dropped         uint64
}

type Network interface {
	GetInterfaceStats(iface string) (*NetworkStats, error)
	GetAllInterfaceStats() (map[string]*NetworkStats, error)
}

type networkImpl struct {
	prevStats map[string]*NetworkStats
	lastCheck time.Time
}

func NewNetwork() Network {
	return &networkImpl{
		prevStats: make(map[string]*NetworkStats),
		lastCheck: time.Now(),
	}
}

func (n *networkImpl) GetAllInterfaceStats() (map[string]*NetworkStats, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stats := make(map[string]*NetworkStats)
	scanner := bufio.NewScanner(file)

	// Skip header lines
	scanner.Scan()
	scanner.Scan()

	now := time.Now()
	timeDiff := now.Sub(n.lastCheck).Seconds()

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(strings.Split(line, ":")[1])
		if len(fields) < 16 {
			continue
		}

		iface := strings.TrimSpace(strings.Split(line, ":")[0])
		stat := &NetworkStats{
			Interface:       iface,
			BytesReceived:   parseUint64(fields[0]),
			PacketsReceived: parseUint64(fields[1]),
			Errors:          parseUint64(fields[2]),
			Dropped:         parseUint64(fields[3]),
			BytesSent:       parseUint64(fields[8]),
			PacketsSent:     parseUint64(fields[9]),
		}

		// Calculate speeds if we have previous measurements
		if prev, ok := n.prevStats[iface]; ok && timeDiff > 0 {
			stat.ReceiveSpeed = float64(stat.BytesReceived-prev.BytesReceived) / timeDiff
			stat.SendSpeed = float64(stat.BytesSent-prev.BytesSent) / timeDiff
		}

		stats[iface] = stat
	}

	n.prevStats = stats
	n.lastCheck = now

	return stats, nil
}

func (n *networkImpl) GetInterfaceStats(iface string) (*NetworkStats, error) {
	stats, err := n.GetAllInterfaceStats()
	if err != nil {
		return nil, err
	}

	if stat, ok := stats[iface]; ok {
		return stat, nil
	}
	return nil, fmt.Errorf("interface not found: %s", iface)
}

func parseUint64(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}
