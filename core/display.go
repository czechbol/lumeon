package core

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/gif"
	_ "image/png" // register PNG decoder
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/czechbol/lumeon/app/config"
	"github.com/czechbol/lumeon/core/hardware"
	"github.com/czechbol/lumeon/core/resources"
)

const (
	displayPageCount      = 4
	displaySleepTimeout   = 2 * time.Minute
	displaySplashDuration = 5 * time.Second
)

type DisplayService interface {
	IsRunning() bool
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
	Wake()
}

type displayServiceImpl struct {
	mutex         sync.RWMutex
	running       bool
	sleeping      bool
	oled          hardware.OLED
	cpu           resources.CPU
	mem           resources.Memory
	net           resources.Network
	drives        resources.HDD
	displayConfig config.DisplayConfig
	ctx           context.Context
	cancel        context.CancelFunc
	shutdownChan  chan struct{}
	wakeChan      chan struct{}
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
		wakeChan:      make(chan struct{}, 1),
	}
}

func (ds *displayServiceImpl) Wake() {
	select {
	case ds.wakeChan <- struct{}{}:
	default:
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

	if !ds.showStartupSplash() {
		return
	}

	page := 0
	ticker := time.NewTicker(ds.displayConfig.Interval())
	defer ticker.Stop()

	sleepTimer := time.NewTimer(displaySleepTimeout)
	defer sleepTimer.Stop()

	// Render first page immediately after splash
	if err := ds.renderPage(page); err != nil {
		slog.Error("failed to render display page", "page", page, "error", err)
	}
	page = (page + 1) % displayPageCount

	for {
		select {
		case <-ds.ctx.Done():
			slog.Info("stopping display loop due to context cancellation")
			return
		case <-ticker.C:
			page = ds.handleTick(page)
		case <-sleepTimer.C:
			ds.handleSleep()
		case <-ds.wakeChan:
			ds.handleWake(ticker, sleepTimer)
		}
	}
}

// showStartupSplash renders the animated splash, warms the CPU cache, and waits
// for the splash duration. Returns false if the context was cancelled.
func (ds *displayServiceImpl) showStartupSplash() bool {
	slog.Info("showing startup splash")
	go func() {
		if _, err := ds.cpu.GetStats(); err != nil {
			slog.Warn("failed to warm CPU stats cache", "error", err)
		}
	}()

	start := time.Now()
	if err := ds.renderAnimatedSplash(); err != nil {
		slog.Error("failed to render animated splash, trying static", "error", err)
	}
	// Show static splash for the remainder of the splash duration.
	if err := ds.renderSplash(); err != nil {
		slog.Error("failed to render startup splash", "error", err)
	}
	remaining := displaySplashDuration - time.Since(start)
	if remaining <= 0 {
		return true
	}
	select {
	case <-ds.ctx.Done():
		return false
	case <-time.After(remaining):
		return true
	}
}

func (ds *displayServiceImpl) handleTick(page int) int {
	ds.mutex.RLock()
	sleeping := ds.sleeping
	ds.mutex.RUnlock()

	if !sleeping {
		if err := ds.renderPage(page); err != nil {
			slog.Error("failed to render display page", "page", page, "error", err)
		}
		page = (page + 1) % displayPageCount
	}

	return page
}

func (ds *displayServiceImpl) handleSleep() {
	slog.Info("display going to sleep")
	ds.mutex.Lock()
	ds.sleeping = true
	ds.mutex.Unlock()

	if err := ds.oled.Clear(); err != nil {
		slog.Error("failed to clear display for sleep", "error", err)
	}
}

func (ds *displayServiceImpl) handleWake(ticker *time.Ticker, sleepTimer *time.Timer) {
	ds.mutex.Lock()
	wasSleeping := ds.sleeping
	ds.sleeping = false
	ds.mutex.Unlock()

	if !sleepTimer.Stop() {
		select {
		case <-sleepTimer.C:
		default:
		}
	}
	sleepTimer.Reset(displaySleepTimeout)

	if wasSleeping {
		slog.Info("display waking up")
		if err := ds.renderSplash(); err != nil {
			slog.Error("failed to render splash on wake", "error", err)
		}
		// Reset ticker so the splash is visible for a full interval
		// before data pages begin rendering.
		ticker.Reset(ds.displayConfig.Interval())
		select {
		case <-ticker.C:
		default:
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

	canvas := newCanvas()
	y := drawHeader(canvas, iconCPUPNG,
		fmt.Sprintf("CPU %.0f\u00b0C", stats.AvgTemperature))

	// Usage bar
	drawProgressBar(canvas, 0, y, canvasW-textWidth(" 100%")-2, stats.UsagePercent)
	drawText(canvas, fmt.Sprintf(" %.0f%%", stats.UsagePercent), canvasW-textWidth(" 100%")-2, y-2)
	y += barHeight + 2

	// Core usage pairs (2 per line)
	for i := 0; i < len(stats.Cores); i += 2 {
		left := fmt.Sprintf("C%d:%.0f%%", stats.Cores[i].ID, stats.Cores[i].UsagePercent)
		if i+1 < len(stats.Cores) {
			right := fmt.Sprintf("C%d:%.0f%%", stats.Cores[i+1].ID, stats.Cores[i+1].UsagePercent)
			drawText(canvas, left, 0, y)
			drawText(canvas, right, canvasW/2, y)
		} else {
			drawText(canvas, left, 0, y)
		}
		y += lineHeight
		if y+lineHeight > canvasH {
			break
		}
	}

	return ds.oled.DrawImage(canvas)
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

	canvas := newCanvas()
	y := drawHeader(canvas, iconMemoryPNG, "Memory")

	// RAM bar + percentage
	pctText := fmt.Sprintf(" %.0f%%", stats.UsagePercent)
	barW := canvasW - textWidth(pctText) - 2
	drawProgressBar(canvas, 0, y, barW, stats.UsagePercent)
	drawText(canvas, pctText, barW+2, y-2)
	y += barHeight + 2

	// RAM usage detail
	drawText(canvas, fmt.Sprintf("RAM  %.1f / %.1f GB", usedGB, totalGB), 0, y)
	y += lineHeight

	// Swap usage
	drawText(canvas, fmt.Sprintf("Swap %.1f / %.1f GB", swapUsedGB, swapTotalGB), 0, y)

	return ds.oled.DrawImage(canvas)
}

func (ds *displayServiceImpl) renderNetworkPage() error {
	allStats, err := ds.net.GetAllInterfaceStats()
	if err != nil {
		return fmt.Errorf("getting network stats: %w", err)
	}

	ifaces := make([]string, 0, len(allStats))
	for iface := range allStats {
		if iface == "lo" ||
			strings.HasPrefix(iface, "veth") ||
			strings.HasPrefix(iface, "br-") {
			continue
		}
		ifaces = append(ifaces, iface)
	}
	sort.Strings(ifaces)

	canvas := newCanvas()
	y := drawHeader(canvas, iconNetworkPNG, "Network")

	if len(ifaces) == 0 {
		drawText(canvas, "No interfaces", 0, y)
		return ds.oled.DrawImage(canvas)
	}

	for _, iface := range ifaces {
		stat := allStats[iface]
		speeds := fmt.Sprintf("\u2193%s \u2191%s", formatSpeed(stat.ReceiveSpeed), formatSpeed(stat.SendSpeed))
		speedsX := rightAlignX(speeds)
		name := truncateToFit(iface, speedsX-7) // 7px gap before speeds
		drawText(canvas, name, 0, y)
		drawText(canvas, speeds, speedsX, y)
		y += lineHeight
		if y+lineHeight > canvasH {
			break
		}
	}

	return ds.oled.DrawImage(canvas)
}

func (ds *displayServiceImpl) renderHDDPage() error {
	allStats, err := ds.drives.GetStats()
	if err != nil {
		return fmt.Errorf("getting hdd stats: %w", err)
	}

	canvas := newCanvas()
	y := drawHeader(canvas, iconHDDPNG, "Storage")

	if len(allStats) == 0 {
		drawText(canvas, "No drives", 0, y)
		return ds.oled.DrawImage(canvas)
	}

	for _, stat := range allStats {
		health := "OK"
		if !stat.SmartStatus.HealthOK {
			health = "!"
		}
		detail := fmt.Sprintf("%.0f\u00b0C %s", stat.Temperature, health)
		detailX := rightAlignX(detail)
		name := truncateToFit(stat.DeviceName, detailX-7) // 7px gap before detail
		drawText(canvas, name, 0, y)
		drawText(canvas, detail, detailX, y)
		y += lineHeight
		if y+lineHeight > canvasH {
			break
		}
	}

	return ds.oled.DrawImage(canvas)
}

// renderSplash draws the embedded splash onto the display.
// Uses the animated GIF on first boot, static PNG on wake.
func (ds *displayServiceImpl) renderSplash() error {
	img, _, err := image.Decode(bytes.NewReader(splashPNG))
	if err != nil {
		return fmt.Errorf("decoding splash image: %w", err)
	}
	return ds.oled.DrawImage(img)
}

// renderAnimatedSplash draws the animated GIF splash once.
func (ds *displayServiceImpl) renderAnimatedSplash() error {
	g, err := gif.DecodeAll(bytes.NewReader(splashGIF))
	if err != nil {
		slog.Warn("failed to decode animated splash, falling back to static", "error", err)
		return ds.renderSplash()
	}
	g.LoopCount = -1 // play once
	return ds.oled.DrawGIF(g)
}
