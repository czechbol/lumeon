package mock

import "github.com/czechbol/lumeon/core/resources"

// CPUMock defines mocks for CPU.
type CPUMock struct {
	GetAverageTempHandler       func() (float64, error)
	GetAverageTempHandlerCalled int
}

var _ resources.CPU = (*CPUMock)(nil)

func (m *CPUMock) GetAverageTemp() (float64, error) {
	m.GetAverageTempHandlerCalled++
	return m.GetAverageTempHandler()
}
