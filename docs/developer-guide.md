# lumEON Developer Guide

This guide covers the architecture, code structure, and development workflow for lumEON. It assumes you are a Go developer comfortable with goroutines, interfaces, and i2c at a conceptual level. You do not need prior experience with the periph.io library or the Argon40 hardware.

## Table of contents

- [lumEON Developer Guide](#lumeon-developer-guide)
  - [Table of contents](#table-of-contents)
  - [Architecture overview](#architecture-overview)
  - [Directory structure](#directory-structure)
  - [App lifecycle](#app-lifecycle)
  - [Services](#services)
    - [FanService (`core/fan.go`)](#fanservice-corefango)
    - [DisplayService (`core/display.go`)](#displayservice-coredisplaygo)
    - [ButtonService (`core/button.go`)](#buttonservice-corebuttongo)
  - [Hardware drivers](#hardware-drivers)
    - [i2c bus (`core/hardware/i2c/`)](#i2c-bus-corehardwarei2c)
    - [Fan driver (`core/hardware/fan.go`)](#fan-driver-corehardwarefango)
    - [OLED driver (`core/hardware/oled.go`)](#oled-driver-corehardwareoledgo)
    - [Button driver (`core/hardware/button.go`)](#button-driver-corehardwarebuttongo)
  - [Resource probers](#resource-probers)
  - [Display rendering](#display-rendering)
  - [Configuration loading](#configuration-loading)
  - [Build](#build)
  - [Test](#test)
  - [Utility commands](#utility-commands)

---

## Architecture overview

lumEON is structured as three independent services that share read-only access to hardware resource data:

```
cmd/lumeond/main.go
    └── app.RunAndManageApp
            ├── FanService      ← polls CPU + HDD temps every 30s, sets fan speed via i2c
            ├── DisplayService  ← cycles OLED pages on a configurable interval
            └── ButtonService   ← watches the physical button, wakes the display on press
```

Each service runs in its own goroutine, communicates via channels and a shared context, and is shut down gracefully on SIGINT or SIGTERM.

There is no network interface and no RPC. The only external communication is over the i2c bus to the Argon EON daughterboard (fan + button at `0x1A`) and SSD1306 OLED display (`0x3C`).

---

## Directory structure

```
cmd/
  lumeond/          — main entry point (binary: lumeond)
  demo_gif/         — utility: renders the OLED demo.gif for the README

app/
  app.go            — App interface, CoreApp implementation, Init/Run/Shutdown
  app_manager.go    — RunAndManageApp: lifecycle, signal handling
  version.go        — version/gitCommit/buildDate vars (overwritten by ldflags)
  config/
    config.go       — Config, FanConfig, DisplayConfig interfaces + implementations
    settings/
      settings.go   — viper + pflag wiring to load lumeon.toml

core/
  core.go           — CoreServices struct (FanService + DisplayService + ButtonService)
  fan.go            — FanService: interface + implementation
  display.go        — DisplayService: interface, display loop, page rendering logic
  display_render.go — Low-level canvas/drawing helpers (text, progress bars, icons)
  button.go         — ButtonService: interface + implementation
  icon_embed.go     — Embedded icon PNGs (CPU, memory, network, HDD)
  splash_embed.go   — Embedded splash GIF + PNG assets

  hardware/
    fan.go          — Fan hardware driver (i2c writes to daughterboard)
    oled.go         — OLED hardware driver (SSD1306 via periph.io/devices)
    button.go       — Button hardware driver (GPIO via periph.io)
    system.go       — System utilities (architecture check helpers)
    constants.go    — i2c addresses and command bytes
    error.go        — Sentinel hardware errors
    i2c/
      bus.go        — i2c bus abstraction (wraps periph.io host)
      mock/         — Mock i2c bus for testing

  resources/
    cpu.go          — CPU temperature + usage stats via gopsutil
    hdd.go          — Drive temperature + SMART data via smartctl
    memory.go       — RAM + swap stats via gopsutil
    network.go      — Network interface stats via gopsutil
    error.go        — Sentinel resource errors

  assets/
    splash.gif      — Animated startup splash (embedded)
    splash.png      — Static splash for wake-from-sleep (embedded)

packaging/
  etc/
    lumeon.toml     — Default config installed to /etc/lumeon/lumeon.toml
    lumeond.service — systemd service unit
  scripts/          — Pre/post install scripts for package managers

.github/workflows/  — CI (test + lint) and release workflows
```

---

## App lifecycle

```
main()
  → settings.GetConfig()      reads lumeon.toml, parses CLI flags
  → app.NewApp(config)        creates CoreApp with config
  → app.RunAndManageApp(app)
        → app.Init()          sets up logger, checks arch, opens i2c bus,
                              constructs hardware drivers and resource probers,
                              wires them into CoreServices
        → app.Run(ctx)        starts each service goroutine, blocks on ctx.Done()
        ← SIGINT / SIGTERM    cancel() called, ctx.Done() fires
        → app.Shutdown(ctx)   stops each service with a 10s timeout, clears display
```

`RunAndManageApp` handles the signal plumbing and returns an exit code. All services receive the same context; cancelling it is the signal for all goroutines to stop.

If `Init` encounters a fatal error (i2c bus unavailable, OLED not found, wrong architecture), it calls `os.Exit(1)` directly. This is intentional — there is nothing sensible to do without the hardware.

---

## Services

All three services follow the same pattern:

```go
type XxxService interface {
    IsRunning() bool
    Start(ctx context.Context) error
    Shutdown(ctx context.Context) error
}
```

`Start` launches a goroutine and returns immediately. `Shutdown` signals the goroutine via `cancel()` and waits for it to confirm via a `shutdownChan`, with a timeout from the passed context. `IsRunning` is protected by a `sync.RWMutex`.

### FanService (`core/fan.go`)

Runs `fanLoop` in a goroutine. On each iteration it:

1. Gets average CPU temperature from the `CPU` resource prober
2. Gets average drive temperature from the `HDD` resource prober
3. Walks each configured curve to find the appropriate fan speed
4. Takes the maximum of the two speeds
5. Calls `fan.SetSpeed(speed)` only if the speed changed
6. Waits 30 seconds via `time.NewTicker`

If either temperature read fails, that channel defaults to 100% fan speed as a fail-safe.

### DisplayService (`core/display.go`)

Runs `displayLoop` in a goroutine. On start:

1. Begins continuous CPU polling in the background (`cpu.Poll(ctx)`) so the cache is warm before the first render
2. Shows an animated splash (GIF, plays once), then a static splash, for ~5 seconds total
3. Renders page 0 immediately, then creates a ticker for subsequent pages

The loop advances through 5 pages (CPU → Memory → Network → Storage SMART → Disk Space) in a cycle. Pages with multiple subpages (Network, SMART, Disk Space) block the loop for multiple ticks while displaying each subpage with a smooth scroll animation between them.

The display sleeps after 2 minutes of inactivity (no button presses), clearing the screen to prevent OLED burn-in. Pressing the button sends to `wakeChan`, which wakes the display and resets the sleep timer.

### ButtonService (`core/button.go`)

Runs `buttonLoop` in a goroutine. Calls `button.WaitForEvent(ctx)` in a blocking loop. On a `ButtonPress` event, calls `display.Wake()`.

---

## Hardware drivers

### i2c bus (`core/hardware/i2c/`)

Wraps `periph.io/x/host/v3` initialization and `periph.io/x/conn/v3/i2c` bus access. The `Bus` interface has a single method `Dev(addr uint16) i2c.Dev`, which returns a device handle for a given address. The mock in `i2c/mock/` implements this interface for testing without real hardware.

### Fan driver (`core/hardware/fan.go`)

Writes a single speed byte to the daughterboard at `0x1A`. Speed is a value from 0–100 (percent). The `cmdSystemHalt` byte (`0xFF`) is reserved and must not be sent as a speed value.

### OLED driver (`core/hardware/oled.go`)

Wraps `periph.io/x/devices/v3/ssd1306` to drive the 128×64 OLED at `0x3C`. Exposes three methods: `DrawImage(image.Image)`, `DrawGIF(*gif.GIF)`, and `Clear()`. The image is expected to be 128×64 pixels; the driver handles the SSD1306 page-addressing protocol internally.

### Button driver (`core/hardware/button.go`)

Reads button press events from the daughterboard via `periph.io`. `WaitForEvent(ctx)` blocks until a press is detected or the context is cancelled.

---

## Resource probers

All probers use [gopsutil](https://github.com/shirou/gopsutil) except HDD SMART data, which shells out to `smartctl`.

| Prober    | File                        | What it provides                                                                          |
| --------- | --------------------------- | ----------------------------------------------------------------------------------------- |
| `CPU`     | `core/resources/cpu.go`     | Average temperature, overall usage %, per-core usage and max frequency                    |
| `HDD`     | `core/resources/hdd.go`     | Per-drive temperature, SMART health, power-on hours, TBW, error counters, partition usage |
| `Memory`  | `core/resources/memory.go`  | RAM used/available/total, usage %, swap used/total                                        |
| `Network` | `core/resources/network.go` | Per-interface receive/transmit speeds and cumulative byte counters, errors, drops         |

`CPU` has an additional `Poll(ctx context.Context)` method that starts a background goroutine to continuously sample CPU usage. gopsutil requires two samples to calculate a CPU usage percentage; calling `Poll` ensures the display always has a fresh reading without blocking on the first render.

---

## Display rendering

The rendering pipeline lives in `core/display.go` and `core/display_render.go`.

The display canvas is a 128×64 `image.RGBA`. Every page is composed by:

1. Drawing a 16px-tall header with a small icon and title text
2. Drawing content in the remaining 48px using `drawText`, `drawProgressBar`, and similar helpers

For pages with multiple subpages (Network, SMART, Disk Space), `scrollPage` is used. It pre-renders all subpages into separate 128×48 content canvases, then displays them one at a time with `animateScroll` providing a smooth vertical scroll transition at ~30fps.

The font is `github.com/hajimehoshi/bitmapfont/v3` — a small pixel font that renders cleanly on the 128×64 display without anti-aliasing. Icons are small PNG images embedded at compile time via `go:embed` in `core/icon_embed.go`.

---

## Configuration loading

Config is loaded by `app/config/settings/settings.go` at startup using:

- **[viper](https://github.com/spf13/viper)** to read `lumeon.toml` from `/etc/lumeon/` or the working directory
- **[pflag](https://github.com/spf13/pflag)** for the `-v` / `-vv` verbosity flags

The raw TOML values are parsed into a `Settings` struct, then converted to the `config.Config` interface (defined in `app/config/config.go`). The interface is what the rest of the application uses; it is intentionally separated from the loading mechanism so config can be provided differently in tests.

Fan curves are stored in TOML as `map[string]string` (because TOML keys must be strings) and are converted to `[]config.FanCurvePoint` sorted by temperature ascending.

---

## Build

```sh
# Local build (development only — exits on non-arm hardware at runtime)
go build ./cmd/lumeond/

# Release packages for arm + arm64 (requires goreleaser)
goreleaser build --snapshot --clean
```

GoReleaser builds with `CGO_ENABLED=0` and `-trimpath`, injects version/commit/date via ldflags, and compresses the binary with UPX. Output packages are `.deb`, `.rpm`, `.apk`, and `.pkg.tar.zst`.

The binary name is `lumeond` (daemon convention). The module path is `github.com/czechbol/lumeon`.

---

## Test

```sh
# Run all tests
go test ./...

# Run a specific test suite
go test ./core/hardware/ -run TestFanTestSuite
go test ./core/hardware/ -run TestFanTestSuite/TestSetSpeed

# With verbose output
go test -v ./...
```

Tests use [testify](https://github.com/stretchr/testify) for assertions. Hardware tests use the mock i2c bus from `core/hardware/i2c/mock/` so they run without real hardware.

---

## Utility commands

```sh
# Generate the demo GIF for the README (requires the OLED to be connected)
go run ./cmd/demo_gif/
```
