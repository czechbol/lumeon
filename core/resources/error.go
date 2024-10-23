package resources

import "errors"

var (
	ErrNotImplemented = errors.New("not implemented")

	// Temperature related errors.
	ErrTemperatureNotFound = errors.New("temperature not found")
	ErrNoThermalZones      = errors.New("no thermal zones found")
	ErrNoValidTemperature  = errors.New("no valid temperature readings")
)
