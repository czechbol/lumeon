package constants

// gpioDevices tracks GPIO pin addresses of connected devices.
type gpioDevices struct {
	// Button on top of the case.
	Button uint8
}

// GPIO holds GPIO pin addresses of connected devices.
var GPIO = gpioDevices{
	Button: 0x04,
}
