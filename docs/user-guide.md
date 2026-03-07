# lumEON User Guide

lumEON is a daemon that runs in the background on your Raspberry Pi and does two things: it keeps the Argon EON case fan at the right speed based on temperature, and it drives the case OLED display with live system stats. Both are fully configurable.

## Table of contents

- [Service management](#service-management)
- [Configuration](#configuration)
- [Display pages](#display-pages)
- [Button behaviour](#button-behaviour)
- [Verbosity flags](#verbosity-flags)
- [Troubleshooting](#troubleshooting)

---

## Service management

lumEON runs as a systemd service called `lumeond`. It starts automatically on boot after installation.

```sh
sudo systemctl status lumeond     # check current status
sudo systemctl start lumeond      # start
sudo systemctl stop lumeond       # stop
sudo systemctl restart lumeond    # restart (required after config changes)
sudo systemctl enable lumeond     # start automatically on boot
sudo systemctl disable lumeond    # remove from boot
```

Logs go to the journal:

```sh
sudo journalctl -u lumeond -f     # follow live logs
sudo journalctl -u lumeond -n 50  # last 50 lines
```

---

## Configuration

The config file lives at `/etc/lumeon/lumeon.toml`. Changes take effect after restarting the service.

### logLevel

Controls how much the daemon logs to the journal.

```toml
logLevel = "info"
```

| Value   | What you see                                        |
|---------|-----------------------------------------------------|
| `error` | Only errors                                         |
| `warn`  | Errors and warnings                                 |
| `info`  | Normal operational messages (default)               |
| `debug` | Everything, including per-cycle fan and display data |

You can also override this temporarily at launch with the `-v` / `-vv` flags (see [Verbosity flags](#verbosity-flags)).

---

### fan.enabled

Enables or disables fan control entirely. When disabled, the fan runs at whatever speed it was last set to before lumEON started (or the hardware default).

```toml
[fan]
enabled = true
```

---

### fan.cpuCurve and fan.hddCurve

These define how fan speed maps to temperature. Each entry is `"temperature_celsius" = "fan_speed_percent"`.

```toml
cpuCurve = { "0" = "20", "30" = "25", "40" = "35", "50" = "50", "60" = "70", "70" = "90", "75" = "100" }
hddCurve = { "0" = "20", "30" = "25", "40" = "50", "50" = "85", "60" = "100" }
```

**How it works:** lumEON checks both the average CPU temperature and the average drive temperature every 30 seconds. For each, it walks the curve from lowest to highest temperature threshold and applies the speed from the last threshold that the current temperature exceeds. The fan is then set to whichever of the two resulting speeds is higher.

For example, with the default CPU curve above, a CPU at 55°C would trigger the `"50" = "50"` entry, giving 50% fan speed.

**Guidelines:**
- Temperature values: integers in °C, 0–255
- Speed values: integers as percentage, 0–100
- Add as many points as you like; they are sorted automatically

> [!WARNING]
> If a temperature reading fails (e.g. `smartmontools` is not installed or a drive is unreadable), lumEON defaults that channel to 100% fan speed as a fail-safe. If your fan is always running at full speed, check the troubleshooting section.

---

### display.enabled

Enables or disables the OLED display.

```toml
[display]
enabled = true
```

---

### display.interval

How many seconds each display page is shown before advancing to the next. For pages with multiple subpages (Network, Storage SMART, Disk Space), each subpage is shown for this duration.

```toml
interval = 5   # seconds, minimum 1
```

> [!NOTE]
> If set to 0 or a negative value, the interval silently defaults to 5 seconds.

---

## Display pages

The display cycles through five pages in order. Each page has a small icon and title in a header row, with content below.

### Page 1 — CPU

Shows average CPU temperature in the header, overall usage as a progress bar with percentage, and per-core usage and maximum frequency in pairs (e.g. `C0:12%4.1G C1:8%4.1G`). If there are more core pairs than fit on screen, they scroll through on each cycle.

### Page 2 — Memory

Shows overall RAM usage as a progress bar with percentage, used and available RAM in GB, and swap usage (used / total GB).

### Page 3 — Network

Shows one subpage per non-loopback, non-virtual network interface. Each subpage shows the interface name and current receive/transmit speeds, cumulative bytes received and sent since boot, and error and drop counters. Interfaces named `lo`, starting with `veth`, or starting with `br-` are filtered out.

### Page 4 — Storage SMART

Shows one subpage per detected drive. Each subpage shows the drive name, temperature, and SMART health status (PASS/FAIL), power-on hours, terabytes written, and reallocated sector, uncorrectable error, and pending sector counts.

Requires `smartmontools` to be installed (it is installed automatically with the lumEON package).

### Page 5 — Disk Space

Shows one subpage per mounted partition across all drives. Each subpage shows the mount point, a usage bar with percentage, and free / total space.

---

## Button behaviour

The physical button on the Argon EON case wakes the OLED display if it has gone to sleep. After waking, the display resets its sleep timer and resumes from the current page.

The display automatically goes to sleep after 2 minutes of no button activity to prevent OLED burn-in.

---

## Verbosity flags

You can pass `-v` or `-vv` to `lumeond` to temporarily override the log level in the config file. This is useful for debugging without editing the config.

```sh
lumeond -v    # info level
lumeond -vv   # debug level
```

When running as a systemd service, add flags to the `ExecStart` line in the service file (`/etc/systemd/system/lumeond.service`), then reload and restart:

```sh
sudo systemctl daemon-reload
sudo systemctl restart lumeond
```

---

## Troubleshooting

**The service fails to start**

Check `sudo journalctl -u lumeond -n 30` for the error. Common causes:

- i2c is not enabled — see [i2c Setup](../README.md#i2c-setup)
- The i2c devices are not detected — run `i2cdetect -y 1` and verify `0x1a` and `0x3c` appear
- Config file is missing or has a syntax error — the log will say `failed to read config file`

**The fan is always at 100%**

This means lumEON could not read a temperature. Check that `smartmontools` is installed (`smartctl --version`) and that your drives are visible (`smartctl -a /dev/sdX`). If only the HDD reading fails and your setup has no SMART-capable drives, set `hddCurve` to a flat curve like `{ "0" = "0" }` to ignore drive temperature.

**The OLED display is blank**

Check that `0x3c` appears in `i2cdetect -y 1`. If the device is detected but the screen stays dark, try restarting the service. If `display.enabled` is `false` in the config, set it to `true` and restart.

**The button does nothing**

The button service starts only if the hardware is detected. If it failed, you will see `button not available, skipping button service` in the logs. This does not affect fan or display operation.
