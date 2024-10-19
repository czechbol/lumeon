package mock

import "github.com/czechbol/lumeon/core/hardware/components"

// FanMock defines mocks for Fan.
type FanMock struct {
	SetSpeedHandler       func(speed uint8) error
	SetSpeedHandlerCalled int
}

var _ components.Fan = (*FanMock)(nil)

func (m *FanMock) SetSpeed(speed uint8) error {
	m.SetSpeedHandlerCalled++
	return m.SetSpeedHandler(speed)
}
