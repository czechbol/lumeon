package constants

type i2cConstants struct {
	Devices  i2cDevices
	Commands i2cCommands
}

// i2cDevices tracks i2c addresses of connected devices.
type i2cDevices struct {
	// Daughter board responsible for the fan and power.
	Daughter uint16
	// Display controller managing the pixels.
	Display uint16
}

// i2cCommands holds i2c commands.
type i2cCommands struct {
	// Halt command.
	Halt []byte
}

// i2cAddresses holds i2c addresses of connected devices.

// I2C holds i2c addresses of connected devices.
var I2C = i2cConstants{
	Devices: i2cDevices{
		Daughter: 0x1a,
		Display:  0x3c,
	},
	Commands: i2cCommands{
		Halt: []byte{0xff},
	},
}
