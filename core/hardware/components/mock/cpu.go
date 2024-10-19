package mock

import "github.com/czechbol/lumeon/core/hardware/components"

// CPUMock defines mocks for CPU.
type CPUMock struct {
	GetAverageTempHandler       func() (float64, error)
	GetAverageTempHandlerCalled int
}

var _ components.CPU = (*CPUMock)(nil)

func (m *CPUMock) GetAverageTemp() (float64, error) {
	m.GetAverageTempHandlerCalled++
	return m.GetAverageTempHandler()
}
