package resources

import "errors"

var (
	ErrNotImplemented = errors.New("not implemented")

	// HDD related errors.
	ErrDriveNotMounted                = errors.New("drive not mounted")
	ErrNoValidDeviceStats             = errors.New("no valid device stats found")
	ErrSmartctlFailed                 = errors.New("smartctl command failed")
	ErrSmartOutputVersionIncompatible = errors.New("smartctl output version incompatible")

	// Network related errors.
	ErrInterfaceNotFound = errors.New("interface not found")

	// Temperature related errors.
	ErrTemperatureNotFound = errors.New("temperature not found")
	ErrNoThermalZones      = errors.New("no thermal zones found")
	ErrNoValidTemperature  = errors.New("no valid temperature readings")
)
