package hardware

import "errors"

var (
	// Image related errors.
	ErrInvalidImageSize         = errors.New("invalid image size")
	ErrUnsupportedImagePosition = errors.New("unsupported image position")

	// Pixel related errors.
	ErrPixelConversion = errors.New("could not convert pixel to color.Gray16")

	// Text related errors.
	ErrIndexFileFormat = errors.New("index file must have format 'name-XxY.bin'")

	// XY related errors.
	ErrInvalidXYFormat = errors.New("invalid XY format")

	// Fan related errors.
	ErrInvalidFanSpeed = errors.New("invalid fan speed")

	// Temperature related errors.
	ErrTemperatureNotFound = errors.New("temperature not found")

	// Smartctl related errors.
	ErrTemperatureAttributeNotFound = errors.New("temperature attribute was not present in smartctl output")

	ErrNotImplemented = errors.New("not implemented")
)
