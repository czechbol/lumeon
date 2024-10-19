package mock

import "github.com/czechbol/lumeon/core/hardware/components"

// SystemMock defines mocks for System.
type SystemMock struct {
	ShutdownHandler       func() error
	ShutdownHandlerCalled int
	HaltHandler           func() error
	HaltHandlerCalled     int
}

var _ components.System = (*SystemMock)(nil)

func (m *SystemMock) Shutdown() error {
	m.ShutdownHandlerCalled++
	return m.ShutdownHandler()
}

func (m *SystemMock) Halt() error {
	m.HaltHandlerCalled++
	return m.HaltHandler()
}
