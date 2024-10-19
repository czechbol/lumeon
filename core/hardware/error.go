package hardware

import "errors"

var (
	ErrNotImplemented = errors.New("not implemented")

	// Fan related errors.
	ErrInvalidFanSpeed = errors.New("invalid fan speed")

	// Temperature related errors.
	ErrTemperatureNotFound = errors.New("temperature not found")
	ErrNoThermalZones      = errors.New("no thermal zones found")
	ErrNoValidTemperature  = errors.New("no valid temperature readings")

	// Display related errors.
	ErrInvalidImageSize        = errors.New("invalid image size")
	ErrInvalidMemoryMode       = errors.New("invalid memory mode")
	ErrInvalidPageStart        = errors.New("invalid page start address")
	ErrInvalidHorizontalOffset = errors.New("invalid horizontal offset")
)
