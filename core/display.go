package core

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/czechbol/lumeon/app/config"
	"github.com/czechbol/lumeon/core/hardware"
	"github.com/czechbol/lumeon/core/resources"
)

const (
	displayPageCount = 4
	bytesPerMB       = 1 << 20
)

type DisplayService interface {
	IsRunning() bool
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

type displayServiceImpl struct {
	mutex         sync.RWMutex
	running       bool
	oled          hardware.OLED
	cpu           resources.CPU
	mem           resources.Memory
	net           resources.Network
	drives        resources.HDD
	displayConfig config.DisplayConfig
	ctx           context.Context
	cancel        context.CancelFunc
	shutdownChan  chan struct{}
}

func NewDisplayService(
	oled hardware.OLED,
	cpu resources.CPU,
	mem resources.Memory,
	net resources.Network,
	drives resources.HDD,
	displayConfig config.DisplayConfig,
) DisplayService {
	return &displayServiceImpl{
		oled:          oled,
		cpu:           cpu,
		mem:           mem,
		net:           net,
		drives:        drives,
		displayConfig: displayConfig,
		shutdownChan:  make(chan struct{}),
	}
}

func (ds *displayServiceImpl) IsRunning() bool {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()
	return ds.running
}

func (ds *displayServiceImpl) Start(ctx context.Context) error {
	ds.ctx, ds.cancel = context.WithCancel(ctx)
	ds.mutex.Lock()
	if ds.running {
		ds.mutex.Unlock()
		return nil
	}
	ds.running = true
	ds.mutex.Unlock()

	slog.Info("starting display loop")

	go ds.displayLoop()

	return nil
}

func (ds *displayServiceImpl) Shutdown(ctx context.Context) error {
	ds.cancel()

	select {
	case <-ds.shutdownChan:
		slog.Info("display loop stopped gracefully")
	case <-ctx.Done():
		slog.Warn("shutdown context expired before display loop could stop")
	}

	if err := ds.oled.Clear(); err != nil {
		slog.Error("failed to clear display on shutdown", "error", err)
	}

	ds.mutex.Lock()
	ds.running = false
	ds.mutex.Unlock()

	return nil
}

func (ds *displayServiceImpl) displayLoop() {
	defer close(ds.shutdownChan)

	page := 0
	ticker := time.NewTicker(ds.displayConfig.Interval())
	defer ticker.Stop()

	for {
		if err := ds.renderPage(page); err != nil {
			slog.Error("failed to render display page", "page", page, "error", err)
		}
		page = (page + 1) % displayPageCount

		select {
		case <-ds.ctx.Done():
			slog.Info("stopping display loop due to context cancellation")
			return
		case <-ticker.C:
			// Continue to the next page
		}
	}
}

func (ds *displayServiceImpl) renderPage(page int) error {
	switch page {
	case 0:
		return ds.renderCPUPage()
	case 1:
		return ds.renderMemoryPage()
	case 2:
		return ds.renderNetworkPage()
	case 3:
		return ds.renderHDDPage()
	}
	return nil
}

func (ds *displayServiceImpl) renderCPUPage() error {
	stats, err := ds.cpu.GetStats()
	if err != nil {
		return fmt.Errorf("getting cpu stats: %w", err)
	}

	lines := make([]string, 0, displayPageCount)
	lines = append(lines, fmt.Sprintf("CPU: %.1f%% %.0f\u00b0C", stats.UsagePercent, stats.AvgTemperature))
	for i, core := range stats.Cores {
		if i >= 3 {
			break
		}
		lines = append(lines, fmt.Sprintf(" C%d: %.1f%%", core.ID, core.UsagePercent))
	}

	return ds.oled.DrawLines(lines)
}

func (ds *displayServiceImpl) renderMemoryPage() error {
	stats, err := ds.mem.GetStats()
	if err != nil {
		return fmt.Errorf("getting memory stats: %w", err)
	}

	const gb = float64(1 << 30)
	usedGB := float64(stats.Used) / gb
	totalGB := float64(stats.Total) / gb
	swapUsedGB := float64(stats.SwapUsed) / gb
	swapTotalGB := float64(stats.SwapTotal) / gb

	lines := []string{
		fmt.Sprintf("RAM: %.1f/%.1f GB", usedGB, totalGB),
		fmt.Sprintf("Used: %.0f%%", stats.UsagePercent),
		fmt.Sprintf("Swap: %.1f/%.1f GB", swapUsedGB, swapTotalGB),
	}

	return ds.oled.DrawLines(lines)
}

func (ds *displayServiceImpl) renderNetworkPage() error {
	allStats, err := ds.net.GetAllInterfaceStats()
	if err != nil {
		return fmt.Errorf("getting network stats: %w", err)
	}

	lines := make([]string, 0, displayPageCount)
	for iface, stat := range allStats {
		if iface == "lo" {
			continue
		}
		rxMB := stat.ReceiveSpeed / bytesPerMB
		txMB := stat.SendSpeed / bytesPerMB
		lines = append(lines, fmt.Sprintf("%s \u2193%.1f \u2191%.1f MB/s", iface, rxMB, txMB))
	}

	if len(lines) == 0 {
		lines = append(lines, "No interfaces")
	}

	return ds.oled.DrawLines(lines)
}

func (ds *displayServiceImpl) renderHDDPage() error {
	allStats, err := ds.drives.GetStats()
	if err != nil {
		return fmt.Errorf("getting hdd stats: %w", err)
	}

	lines := make([]string, 0, len(allStats))
	for _, stat := range allStats {
		health := "OK"
		if !stat.SmartStatus.HealthOK {
			health = "!"
		}
		lines = append(lines, fmt.Sprintf("%s %.0f\u00b0C %s", stat.DeviceName, stat.Temperature, health))
	}

	if len(lines) == 0 {
		lines = append(lines, "No drives")
	}

	return ds.oled.DrawLines(lines)
}
