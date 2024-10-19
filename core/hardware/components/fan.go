package components

import (
	"fmt"

	"github.com/czechbol/lumeon/core/hardware"
	"github.com/czechbol/lumeon/core/hardware/components/i2c"
	"github.com/czechbol/lumeon/core/hardware/constants"
)

type Fan interface {
	SetSpeed(speed uint8) error
}

type fanImpl struct {
	bus i2c.I2CBus
}

func NewFan(bus i2c.I2CBus) Fan {
	return &fanImpl{bus: bus}
}

func (f *fanImpl) SetSpeed(speed uint8) error {
	if speed > 100 {
		return fmt.Errorf("%w: speed is specified in percent: 0 to 100", hardware.ErrInvalidFanSpeed)
	}

	return f.bus.SendBytes(constants.I2C.Devices.Daughter, []byte{speed})
}
