package mock

import (
	"github.com/czechbol/lumeon/core/hardware/i2c"
	i2clib "periph.io/x/conn/v3/i2c"
)

// I2CBus defines mocks for I2CBus.
type I2CBus struct {
	GetBusHandler         func() i2clib.Bus
	GetBusCalled          int
	SendDataHandler       func(addr uint16, data ...byte) error
	SendDataHandlerCalled int
}

var _ i2c.I2CBus = (*I2CBus)(nil)

func (m *I2CBus) GetBus() i2clib.Bus {
	m.GetBusCalled++
	return m.GetBusHandler()
}

func (m *I2CBus) SendData(addr uint16, data ...byte) error {
	m.SendDataHandlerCalled++
	return m.SendDataHandler(addr, data...)
}
