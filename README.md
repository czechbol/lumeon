# lumEON

![GitHub License](https://img.shields.io/github/license/czechbol/lumeon)
![GitHub Release Date](https://img.shields.io/github/release-date/czechbol/lumeon)
![GitHub Release (latest by date)](https://img.shields.io/github/v/release/czechbol/lumeon)

An alternative implementation for Argon40's EON case.
The [official repository](https://github.com/Argon40Tech/Argon40case) is a mix of shell and Python scripts that is controlled by a single shell loop.

The scope of this repository is to cover basics for the Argon EON case; I don't have their other products to test my implementation against.


## Installation

Ready-made packages are available for download in `.deb`, `.apk`, `.rpm` and `.pkg.tar.zst` formats. These packages automatically set up everything for you. You can find these packages in the [Releases](https://github.com/czechbol/lumeon/releases) section of this repository.

To install the package, download the appropriate file for your system from the Releases section and follow the standard installation procedure for your package manager.

For example, on a Debian-based system, you can install the `.deb` package using the following command:

```shell
$ sudo apt install ./lumeon.deb      # Debian and derivates
$ sudo apk add ./lumeon.apk          # Alpine and derivates
$ sudo dnf install ./lumeon.rpm      # Fedora and derivates
$ sudo pacman -U lumeon.pkg.tar.zst  # Arch and derivates
```

After installing the package, the service should be set up and running.
You can check its status using your system's service management commands.

```shell
$ sudo systemctl status lumeon
```

You can also use the following commands to manage the service using the following commands:

```shell
$ sudo systemctl enable lumeond # Enable the service to start on boot
$ sudo systemctl start lumeond
$ sudo systemctl stop lumeond
$ sudo systemctl disable lumeond
$ sudo systemctl restart lumeond
```


### i2c bus

The i2c bus is required to make the i2c bus available to the programs on the system.

On Raspberry Pi OS, you can do it via `raspi-config`.
Otherwise, make sure the following lines are present in `/boot/config.txt`:

```ini
dtparam=i2c_arm=on,i2c_arm_baudrate=400000 # baudrate is optional but recommended
```

Reboot after making the changes; after that the system should be able to detect
the case's daughter board on the bus `1a` and
the case's OLED display on the bus `3c`.
It should look something like this:

```shell
$ i2cdetect -y 1
     0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
00:                         -- -- -- -- -- -- -- --
10: -- -- -- -- -- -- -- -- -- -- 1a -- -- -- -- --
20: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --
30: -- -- -- -- -- -- -- -- -- -- -- -- 3c -- -- --
40: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --
50: -- 51 -- -- -- -- -- -- -- -- -- -- -- -- -- --
60: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --
70: -- -- -- -- -- -- -- --
```

## Development Prerequisites

Before you can start developing this project, you may want to to install the following tools:

- [GoReleaser](https://goreleaser.com/): Used for building and releasing the project. You can install it by following the instructions on the [GoReleaser installation page](https://goreleaser.com/install/).

- [golangci-lint](https://golangci-lint.run/): Used for linting the Go code. You can install it by following the instructions on the [golangci-lint installation page](https://golangci-lint.run/usage/install/).

After installing these tools, you can start developing the project by following the instructions in [HACKING.md](HACKING.md).


## Contributing

This source code of the project is published under the [Mozilla Public License 2.0](LICENSE).
