package mock

import "github.com/czechbol/lumeon/core/hardware/components/i2c"

// I2CBus defines mocks for I2CBus.
type I2CBus struct {
	SendBytesHandler       func(addr uint16, bytes []byte) error
	SendBytesHandlerCalled int
}

var _ i2c.I2CBus = (*I2CBus)(nil)

func (m *I2CBus) SendBytes(addr uint16, bytes []byte) error {
	m.SendBytesHandlerCalled++
	return m.SendBytesHandler(addr, bytes)
}
