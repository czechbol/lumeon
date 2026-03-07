# lumEON

![GitHub License](https://img.shields.io/github/license/czechbol/lumeon)
![GitHub Release (latest by date)](https://img.shields.io/github/v/release/czechbol/lumeon)
![GitHub Release Date](https://img.shields.io/github/release-date/czechbol/lumeon)

An alternative daemon for the [Argon40 EON](https://www.argon40.com/products/argon-eon-pi-nas) Raspberry Pi NAS case. It controls the case fan via configurable temperature curves and drives the OLED display with live system stats over i2c.

The [official software](https://github.com/Argon40Tech/Argon40case) is a mix of shell and Python scripts running in a single loop. lumEON replaces it with a single self-contained Go binary.

<p align="center">
  <img src="docs/demo.gif" alt="OLED display demo" />
</p>

> [!NOTE]
> lumEON targets the Argon EON case only. Other Argon40 products are untested.

## Requirements

- Raspberry Pi with **arm** or **arm64** architecture
- i2c enabled (see [i2c setup](#i2c-setup) below)
- One of: Debian/Ubuntu, Fedora/RHEL, Alpine, or Arch Linux

## Installation

Packages are available in `.deb`, `.rpm`, `.apk`, and `.pkg.tar.zst` formats. They install the binary, systemd service, and default config in one step.

### Debian / Ubuntu

```sh
curl -LO https://github.com/czechbol/lumeon/releases/latest/download/lumeon_linux_arm64.deb
sudo apt install ./lumeon_linux_arm64.deb
```

### Fedora / RHEL

```sh
curl -LO https://github.com/czechbol/lumeon/releases/latest/download/lumeon_linux_arm64.rpm
sudo dnf install ./lumeon_linux_arm64.rpm
```

### Alpine

```sh
curl -LO https://github.com/czechbol/lumeon/releases/latest/download/lumeon_linux_arm64.apk
sudo apk add --allow-untrusted ./lumeon_linux_arm64.apk
```

### Arch Linux

```sh
curl -LO https://github.com/czechbol/lumeon/releases/latest/download/lumeon_linux_arm64.pkg.tar.zst
sudo pacman -U lumeon_linux_arm64.pkg.tar.zst
```

> For 32-bit Raspberry Pi OS, replace `arm64` with `armv7` in the filename.

After installation the service starts automatically. Check its status with:

```sh
sudo systemctl status lumeond
```

Common service commands:

```sh
sudo systemctl enable lumeond   # start on boot
sudo systemctl start lumeond
sudo systemctl stop lumeond
sudo systemctl restart lumeond
```

## i2c Setup

i2c must be enabled for lumEON to communicate with the fan controller and OLED display.

**Raspberry Pi OS:** run `sudo raspi-config` → Interface Options → I2C → Enable.

**Other distros:** add the following to `/boot/config.txt` and reboot:

```ini
dtparam=i2c_arm=on,i2c_arm_baudrate=400000
```

After rebooting, verify the devices are detected:

```sh
$ i2cdetect -y 1
     0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
00:                         -- -- -- -- -- -- -- --
10: -- -- -- -- -- -- -- -- -- -- 1a -- -- -- -- --
20: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --
30: -- -- -- -- -- -- -- -- -- -- -- -- 3c -- -- --
```

- `0x1a` — daughterboard (fan + button controller)
- `0x3c` — OLED display

## Configuration

The default config is installed at `/etc/lumeon/lumeon.toml`. Edit it to adjust fan curves, display interval, and log level, then restart the service.

```toml
logLevel = "info"   # debug | info | warn | error

[fan]
enabled = true
cpuCurve = { "0" = "20", "30" = "25", "50" = "50", "70" = "90", "75" = "100" }
hddCurve = { "0" = "20", "30" = "25", "40" = "50", "50" = "85", "60" = "100" }

[display]
enabled = true
interval = 5   # seconds per page
```

See the [User Guide](docs/user-guide.md) for a full explanation of every option.

## Documentation

- [User Guide](docs/user-guide.md) — configuration reference, display pages, button and sleep behaviour
- [Developer Guide](docs/developer-guide.md) — architecture, code structure, build and test instructions
- [Contributing Guide](docs/contributing.md) — how to contribute: setup, style, PR process

## Acknowledgements

lumEON was initially inspired by and built upon [neon](https://codeberg.org/pancake/neon) by pancake, also licensed [MPL-2.0](https://codeberg.org/pancake/neon/raw/branch/main/LICENSE).

## Contributing

Source code is published under the [Mozilla Public License 2.0](LICENSE). See the [Contributing Guide](docs/contributing.md) for how to get started.
