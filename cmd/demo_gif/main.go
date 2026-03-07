// Package main generates a demo GIF showing all OLED display pages with mock data.
// Run: go run ./cmd/demo_gif/ [-o demo.gif] [-scale 4]
//
// The tool starts the full DisplayService with a mock OLED that captures every
// DrawImage call as a frame. It runs for ~8 seconds (5 s animated splash +
// ~3 s for one full page cycle at 200 ms/page), then assembles the captured
// frames into a looping GIF, skipping the splash frames.
package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"log"
	"os"
	"sync"
	"time"

	"github.com/czechbol/lumeon/app/config"
	"github.com/czechbol/lumeon/core"
	"github.com/czechbol/lumeon/core/hardware"
	"github.com/czechbol/lumeon/core/hardware/types"
	"github.com/czechbol/lumeon/core/resources"
	resmock "github.com/czechbol/lumeon/core/resources/mock"
)

// ---- frame-capturing OLED -----------------------------------------------

type capturedFrame struct {
	img  image.Image
	when time.Time
}

type capturingOLED struct {
	mu     sync.Mutex
	frames []capturedFrame
}

// copyImage makes a deep copy so the source canvas can be reused.
func copyImage(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(x, y, src.At(x, y))
		}
	}
	return dst
}

func (c *capturingOLED) DrawImage(img image.Image) error {
	cp := copyImage(img)
	c.mu.Lock()
	c.frames = append(c.frames, capturedFrame{img: cp, when: time.Now()})
	c.mu.Unlock()
	return nil
}

func (c *capturingOLED) DrawGIF(g *gif.GIF) error {
	for _, frame := range g.Image {
		_ = c.DrawImage(frame)
	}
	return nil
}

func (c *capturingOLED) Invert(bool) error                                             { return nil }
func (c *capturingOLED) SetContrast(uint8) error                                       { return nil }
func (c *capturingOLED) Clear() error                                                  { return nil }
func (c *capturingOLED) DrawText(string, int, int) error                               { return nil }
func (c *capturingOLED) DrawLines([]string) error                                      { return nil }
func (c *capturingOLED) DrawImageWithText(image.Image, int, int, string) error         { return nil }
func (c *capturingOLED) DrawGIFWithText(*gif.GIF, int, int, string) error              { return nil }
func (c *capturingOLED) Scroll(types.ScrollDirection, types.FrameRate, int, int) error { return nil }
func (c *capturingOLED) StopScroll() error                                             { return nil }

var _ hardware.OLED = (*capturingOLED)(nil)

// ---- mock resources -------------------------------------------------------

func buildCPU() resources.CPU {
	return &resmock.CPUMock{
		GetAverageTempHandler: func() (float64, error) { return 52, nil },
		GetStatsHandler: func() (*resources.CPUStats, error) {
			return &resources.CPUStats{
				UsagePercent:   42,
				AvgTemperature: 52,
				CoreCount:      4,
				Cores: []resources.CoreStats{
					{ID: 0, UsagePercent: 38, MaxFrequency: 1800},
					{ID: 1, UsagePercent: 45, MaxFrequency: 1800},
					{ID: 2, UsagePercent: 52, MaxFrequency: 1800},
					{ID: 3, UsagePercent: 34, MaxFrequency: 1800},
				},
			}, nil
		},
	}
}

func buildMemory() resources.Memory {
	const gb = 1 << 30
	return &resmock.MemoryMock{
		GetStatsHandler: func() (*resources.MemoryStats, error) {
			total := uint64(8) * gb
			used := uint64(32) * gb / 10 // 3.2 GB
			avail := uint64(9) * gb / 2  // 4.5 GB
			swapTotal := uint64(4) * gb
			swapUsed := uint64(512) * (1 << 20)
			return &resources.MemoryStats{
				Total:        total,
				Used:         used,
				Available:    avail,
				SwapTotal:    swapTotal,
				SwapUsed:     swapUsed,
				UsagePercent: float64(used) / float64(total) * 100,
			}, nil
		},
	}
}

func buildNetwork() resources.Network {
	const mb = 1 << 20
	const gb = 1 << 30
	return &resmock.NetworkMock{
		GetInterfaceStatsHandler: func(_ string) (*resources.NetworkStats, error) {
			return nil, nil
		},
		GetAllInterfaceStatsHandler: func() (map[string]*resources.NetworkStats, error) {
			return map[string]*resources.NetworkStats{
				"eth0": {
					Interface:     "eth0",
					ReceiveSpeed:  1.2 * mb,
					SendSpeed:     28 * 1024,
					BytesReceived: uint64(6) * gb / 5, // 1.2 GB
					BytesSent:     234 * mb,
				},
				"wlan0": {
					Interface:     "wlan0",
					ReceiveSpeed:  4.5 * 1024,
					SendSpeed:     1.1 * 1024,
					BytesReceived: 456 * mb,
					BytesSent:     89 * mb,
				},
			}, nil
		},
	}
}

func buildHDD() resources.HDD {
	const gb = 1 << 30
	return &resmock.HDDMock{
		GetAverageTempHandler: func() (float64, error) { return 32, nil },
		GetStatsHandler: func() ([]resources.HDDStats, error) {
			return []resources.HDDStats{
				{
					DeviceName:  "sda",
					Temperature: 32,
					TotalSize:   uint64(1000) * gb,
					SmartStatus: resources.SmartStatus{
						HealthOK:            true,
						PowerOnHours:        8760,
						TerabytesWritten:    2,
						ReallocatedSectors:  0,
						UncorrectableErrors: 0,
						PendingSectors:      0,
					},
					Partitions: []resources.Partition{
						{
							Name:       "sda1",
							Mountpoint: "/",
							Total:      uint64(120) * gb,
							Free:       uint64(90) * gb,
						},
						{
							Name:       "sda2",
							Mountpoint: "/home",
							Total:      uint64(800) * gb,
							Free:       uint64(400) * gb,
						},
					},
				},
			}, nil
		},
	}
}

// ---- GIF assembly ---------------------------------------------------------

var oledPalette = color.Palette{
	color.RGBA{R: 0, G: 0, B: 0, A: 255},
	color.RGBA{R: 255, G: 255, B: 255, A: 255},
}

// scaleAndPalette scales src by scale factor (nearest-neighbour) and converts
// to a 2-colour paletted image.
func scaleAndPalette(src image.Image, scale int) *image.Paletted {
	b := src.Bounds()
	w := b.Dx() * scale
	h := b.Dy() * scale
	dst := image.NewPaletted(image.Rect(0, 0, w, h), oledPalette)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, _, _, _ := src.At(x/scale, y/scale).RGBA()
			if r > 0x8000 {
				dst.SetColorIndex(x, y, 1)
			}
		}
	}
	return dst
}

// buildGIF converts captured frames into a looping GIF:
//   - Splash animation frames (captured near-instantly) get animUnits delay
//   - Static splash and data page frames (gap to next > animThresh) get dwellUnits
//   - Scroll-animation frames (gap ≤ animThresh) get animUnits
//
// The 5 s splash-wait gap has no frames, so the last splash frame naturally
// gets dwellUnits just like a data page frame.
func buildGIF(frames []capturedFrame, outputPath string, scale int) {
	const animThresh = 50 * time.Millisecond
	const dwellUnits = 300 // 3 s (GIF delay unit = 10 ms)
	const animUnits = 8    // 80 ms

	data := frames
	if len(data) == 0 {
		log.Fatal("no frames captured")
	}

	gifFrames := make([]*image.Paletted, 0, len(data))
	delays := make([]int, 0, len(data))

	for i, f := range data {
		var delay int
		if i+1 < len(data) {
			delta := data[i+1].when.Sub(f.when)
			if delta > animThresh {
				delay = dwellUnits
			} else {
				delay = animUnits
			}
		} else {
			delay = dwellUnits
		}
		gifFrames = append(gifFrames, scaleAndPalette(f.img, scale))
		delays = append(delays, delay)
	}

	g := &gif.GIF{
		Image:     gifFrames,
		Delay:     delays,
		LoopCount: 0, // loop forever
	}

	f, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("creating %s: %v", outputPath, err)
	}

	if err := gif.EncodeAll(f, g); err != nil {
		_ = f.Close()
		log.Fatalf("encoding GIF: %v", err)
	}

	if err := f.Close(); err != nil {
		log.Fatalf("closing %s: %v", outputPath, err)
	}

	fmt.Printf("wrote %d frames to %s (%dx%d pixels)\n",
		len(gifFrames), outputPath, 128*scale, 64*scale)
}

// ---- main -----------------------------------------------------------------

func main() {
	output := flag.String("o", "docs/demo.gif", "output GIF file path")
	scale := flag.Int("scale", 6, "pixel scale factor (default 6 → 768×384)")
	flag.Parse()

	oled := &capturingOLED{}
	dispCfg := config.NewDisplayConfig(true, 200*time.Millisecond)

	svc := core.NewDisplayService(
		oled,
		buildCPU(),
		buildMemory(),
		buildNetwork(),
		buildHDD(),
		dispCfg,
	)

	// Run long enough for the 5 s animated splash and one full page cycle.
	ctx, cancel := context.WithTimeout(context.Background(), 9*time.Second)

	if err := svc.Start(ctx); err != nil {
		cancel()
		log.Fatalf("starting display service: %v", err)
	}

	<-ctx.Done()
	cancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), time.Second)
	defer shutCancel()
	_ = svc.Shutdown(shutCtx)

	oled.mu.Lock()
	frames := make([]capturedFrame, len(oled.frames))
	copy(frames, oled.frames)
	oled.mu.Unlock()

	fmt.Printf("captured %d total frames\n", len(frames))
	buildGIF(frames, *output, *scale)
}
