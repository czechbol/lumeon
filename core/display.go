package core

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
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
	displayPageCount      = 5
	displaySleepTimeout   = 2 * time.Minute
	displaySplashDuration = 5 * time.Second

	// Smooth-scroll animation for multi-subpage pages.
	scrollStep  = 4                     // pixels advanced per animation frame
	scrollDelay = 33 * time.Millisecond // ~30fps
)

type DisplayService interface {
	IsRunning() bool
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
	Wake()
}

// linesPerPage is the number of data rows that fit below the header.
const linesPerPage = (canvasH - headerHeight) / lineHeight // = 3

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

	// cpuCoreOffset scrolls core pairs on the CPU page.
	cpuCoreOffset int
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
			// Drain any tick that accumulated while a scrollPage-based page was
			// blocking internally. Without this, the buffered tick causes the next
			// page to render immediately with no visible dwell time.
			select {
			case <-ticker.C:
			default:
			}
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

// pageSlice returns the [start, end) window into a list of `total` items for the
// current scroll offset, and the next offset to use on the following render.
// If total <= pageSize all items are shown and the offset resets to 0.
func pageSlice(total, offset, pageSize int) (start, end, next int) {
	if total <= pageSize {
		return 0, total, 0
	}
	start = offset
	if start >= total {
		start = 0
	}
	end = start + pageSize
	if end > total {
		end = total
	}
	next = start + pageSize
	if next >= total {
		next = 0
	}
	return
}

// scrollFrame composites the fixed header with a scrolled content slice and sends it to the OLED.
// scrollOff is how many pixels of curr have scrolled off the top; next fills the gap from below.
func (ds *displayServiceImpl) scrollFrame(icon image.Image, title string, curr, next image.Image, scrollOff int) error {
	frame := newCanvas()
	drawIcon(frame, icon, 0, 0)
	drawText(frame, title, iconSize+2, 0)

	showCurr := contentH - scrollOff
	if showCurr > 0 {
		draw.Draw(frame,
			image.Rect(0, headerHeight, canvasW, headerHeight+showCurr),
			curr,
			image.Point{0, scrollOff},
			draw.Src)
	}
	if next != nil && scrollOff > 0 {
		draw.Draw(frame,
			image.Rect(0, headerHeight+showCurr, canvasW, headerHeight+contentH),
			next,
			image.Point{0, 0},
			draw.Src)
	}
	return ds.oled.DrawImage(frame)
}

// animateScroll smoothly scrolls the content area from curr to next over scrollStep-pixel increments.
func (ds *displayServiceImpl) animateScroll(icon image.Image, title string, curr, next image.Image) error {
	for off := scrollStep; off <= contentH; off += scrollStep {
		if err := ds.scrollFrame(icon, title, curr, next, off); err != nil {
			return err
		}
		select {
		case <-ds.ctx.Done():
			return nil
		case <-time.After(scrollDelay):
		}
	}
	return nil
}

// scrollPage renders a page with a fixed header and a vertically-scrolling content area.
// subpages is a list of draw functions, each filling a 128×48 content canvas.
// The first subpage is shown immediately; subsequent subpages scroll in from below
// with a smooth animation. The method blocks until all subpages have been displayed,
// advancing one subpage per display interval, or until the context is cancelled.
func (ds *displayServiceImpl) scrollPage(iconData []byte, title string, subpages []func(draw.Image)) error {
	if len(subpages) == 0 {
		return nil
	}

	contents := make([]image.Image, len(subpages))
	for i, fn := range subpages {
		c := newContentCanvas()
		fn(c)
		contents[i] = c
	}

	icon := decodeIcon(iconData)

	if err := ds.scrollFrame(icon, title, contents[0], nil, 0); err != nil {
		return err
	}

	for i := range contents {
		select {
		case <-ds.ctx.Done():
			return nil
		case <-time.After(ds.displayConfig.Interval()):
		}

		if i+1 < len(contents) {
			if err := ds.animateScroll(icon, title, contents[i], contents[i+1]); err != nil {
				return err
			}
		}
	}

	return nil
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
		return ds.renderHDDSMARTPage()
	case 4:
		return ds.renderHDDSpacePage()
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

	// Usage bar + percentage on the same row
	pctText := fmt.Sprintf(" %.0f%%", stats.UsagePercent)
	barW := canvasW - textWidth(pctText) - 2
	drawProgressBar(canvas, 0, y, barW, stats.UsagePercent)
	drawText(canvas, pctText, barW+2, y)
	y += lineHeight

	// Core usage pairs (2 per line); scroll when pairs exceed available lines.
	// One row is consumed by the bar, so linesPerPage-1 pairs fit.
	const coreLinesPerPage = linesPerPage - 1
	pairs := (len(stats.Cores) + 1) / 2
	offset, end, next := pageSlice(pairs, ds.cpuCoreOffset, coreLinesPerPage)
	ds.cpuCoreOffset = next

	for p := offset; p < end; p++ {
		i := p * 2
		left := fmt.Sprintf(
			"C%d:%.0f%%%.1fG",
			stats.Cores[i].ID,
			stats.Cores[i].UsagePercent,
			stats.Cores[i].MaxFrequency/1000,
		)
		drawText(canvas, left, 0, y)
		if i+1 < len(stats.Cores) {
			right := fmt.Sprintf(
				"C%d:%.0f%%%.1fG",
				stats.Cores[i+1].ID,
				stats.Cores[i+1].UsagePercent,
				stats.Cores[i+1].MaxFrequency/1000,
			)
			drawText(canvas, right, canvasW/2, y)
		}
		y += lineHeight
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
	availGB := float64(stats.Available) / gb
	swapUsedGB := float64(stats.SwapUsed) / gb
	swapTotalGB := float64(stats.SwapTotal) / gb

	canvas := newCanvas()
	y := drawHeader(canvas, iconMemoryPNG, "Memory")

	// RAM bar + percentage on the same row
	pctText := fmt.Sprintf(" %.0f%%", stats.UsagePercent)
	barW := canvasW - textWidth(pctText) - 2
	drawProgressBar(canvas, 0, y, barW, stats.UsagePercent)
	drawText(canvas, pctText, barW+2, y)
	y += lineHeight

	// RAM: used by apps + reclaimable-inclusive available
	drawText(canvas, fmt.Sprintf("Used %.1f  Avail %.1fG", usedGB, availGB), 0, y)
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

	if len(ifaces) == 0 {
		canvas := newCanvas()
		drawHeader(canvas, iconNetworkPNG, "Network")
		drawText(canvas, "No interfaces", 0, headerHeight)
		return ds.oled.DrawImage(canvas)
	}

	// One subpage per interface: row1 speeds, row2 cumulative totals, row3 errors.
	subpages := make([]func(draw.Image), len(ifaces))
	for i, iface := range ifaces {
		stat := allStats[iface]
		subpages[i] = func(content draw.Image) {
			y := 0
			speeds := fmt.Sprintf("\u2193%s \u2191%s", formatSpeed(stat.ReceiveSpeed), formatSpeed(stat.SendSpeed))
			speedsX := rightAlignX(speeds)
			drawText(content, truncateToFit(iface, speedsX-6), 0, y)
			drawText(content, speeds, speedsX, y)
			y += lineHeight

			totals := fmt.Sprintf("\u2193%s \u2191%s", formatBytes(stat.BytesReceived), formatBytes(stat.BytesSent))
			totalsX := rightAlignX(totals)
			drawText(content, "tot", 0, y)
			drawText(content, totals, totalsX, y)
			y += lineHeight

			drawText(content, fmt.Sprintf("err:%d drop:%d", stat.Errors, stat.Dropped), 0, y)
		}
	}
	return ds.scrollPage(iconNetworkPNG, "Network", subpages)
}

func (ds *displayServiceImpl) renderHDDSMARTPage() error {
	allStats, err := ds.drives.GetStats()
	if err != nil {
		return fmt.Errorf("getting hdd stats: %w", err)
	}

	if len(allStats) == 0 {
		canvas := newCanvas()
		drawHeader(canvas, iconHDDPNG, "Storage")
		drawText(canvas, "No drives", 0, headerHeight)
		return ds.oled.DrawImage(canvas)
	}

	// One subpage per drive: row1 name+temp+health, row2 POH+TBW, row3 error counters.
	subpages := make([]func(draw.Image), len(allStats))
	for i, stat := range allStats {
		subpages[i] = func(content draw.Image) {
			y := 0
			health := "PASS"
			if !stat.SmartStatus.HealthOK {
				health = "FAIL"
			}
			detail := fmt.Sprintf("%.0f\u00b0C %s", stat.Temperature, health)
			detailX := rightAlignX(detail)
			drawText(content, truncateToFit(stat.DeviceName, detailX-6), 0, y)
			drawText(content, detail, detailX, y)
			y += lineHeight

			drawText(
				content,
				fmt.Sprintf("POH:%dh TBW:%dT", stat.SmartStatus.PowerOnHours, stat.SmartStatus.TerabytesWritten),
				0,
				y,
			)
			y += lineHeight

			drawText(
				content,
				fmt.Sprintf(
					"RS:%d UE:%d PS:%d",
					stat.SmartStatus.ReallocatedSectors,
					stat.SmartStatus.UncorrectableErrors,
					stat.SmartStatus.PendingSectors,
				),
				0,
				y,
			)
		}
	}
	return ds.scrollPage(iconHDDPNG, "Storage", subpages)
}

func (ds *displayServiceImpl) renderHDDSpacePage() error {
	allStats, err := ds.drives.GetStats()
	if err != nil {
		return fmt.Errorf("getting hdd stats: %w", err)
	}

	// Collect all mounted partitions across all drives.
	var allParts []resources.Partition
	for _, stat := range allStats {
		allParts = append(allParts, stat.Partitions...)
	}

	if len(allParts) == 0 {
		canvas := newCanvas()
		drawHeader(canvas, iconHDDPNG, "Disk Space")
		drawText(canvas, "No partitions", 0, headerHeight)
		return ds.oled.DrawImage(canvas)
	}

	// One subpage per partition: row1 mountpoint, row2 usage bar+%, row3 free/total.
	subpages := make([]func(draw.Image), len(allParts))
	for i, part := range allParts {
		subpages[i] = func(content draw.Image) {
			y := 0
			drawText(content, truncateToFit(part.Mountpoint, canvasW), 0, y)
			y += lineHeight

			usedPct := 100.0 * float64(part.Total-part.Free) / float64(part.Total)
			pctText := fmt.Sprintf(" %.0f%%", usedPct)
			barW := canvasW - textWidth(pctText) - 2
			drawProgressBar(content, 0, y, barW, usedPct)
			drawText(content, pctText, barW+2, y)
			y += lineHeight

			drawText(content, fmt.Sprintf("%s free / %s", formatBytes(part.Free), formatBytes(part.Total)), 0, y)
		}
	}
	return ds.scrollPage(iconHDDPNG, "Disk Space", subpages)
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
