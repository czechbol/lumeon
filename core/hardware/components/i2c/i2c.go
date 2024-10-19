package i2c

import (
	"fmt"
	"log/slog"

	"periph.io/x/conn/v3/driver/driverreg"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/host/v3"
)

type I2CBus interface {
	SendBytes(addr uint16, bytes []byte) error
}

type i2cBusImpl struct {
	bus i2c.Bus
}

func NewBus(busName string) (I2CBus, error) {
	if _, err := host.Init(); err != nil {
		slog.Error("cannot start host")
		return nil, err
	}

	if _, err := driverreg.Init(); err != nil {
		slog.Error("cannot inicialize i2c driver")
		return nil, err
	}
	i2cBus, err := i2creg.Open(busName)
	if err != nil {
		slog.Error("cannot open i2c bus")
		return nil, err
	}

	return i2cBusImpl{
		bus: i2cBus,
	}, nil
}

func (Bus i2cBusImpl) SendBytes(addr uint16, bytes []byte) error {
	slog.Debug(fmt.Sprintf("sending bytes to 0x%x: %v", addr, bytes))

	device := i2c.Dev{Bus: Bus.bus, Addr: uint16(addr)}

	_, err := device.Write(bytes)
	if err != nil {
		slog.Error(fmt.Sprintf("cannot send bytes to %d: %v", addr, err))
		return err
	}

	return nil
}
