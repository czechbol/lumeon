package mock

import "github.com/czechbol/lumeon/core/hardware"

// FanMock defines mocks for Fan.
type FanMock struct {
	SetSpeedHandler       func(speed uint8) error
	SetSpeedHandlerCalled int
}

var _ hardware.Fan = (*FanMock)(nil)

func (m *FanMock) SetSpeed(speed uint8) error {
	m.SetSpeedHandlerCalled++
	return m.SetSpeedHandler(speed)
}
