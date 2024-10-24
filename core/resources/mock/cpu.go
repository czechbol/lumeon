package mock

import "github.com/czechbol/lumeon/core/resources"

// CPUMock defines mocks for CPU.
type CPUMock struct {
	GetAverageTempHandler       func() (float64, error)
	GetAverageTempHandlerCalled int

	GetStatsHandler       func() (*resources.CPUStats, error)
	GetStatsHandlerCalled int
}

var _ resources.CPU = (*CPUMock)(nil)

func (m *CPUMock) GetAverageTemp() (float64, error) {
	m.GetAverageTempHandlerCalled++
	return m.GetAverageTempHandler()
}

func (m *CPUMock) GetStats() (*resources.CPUStats, error) {
	m.GetStatsHandlerCalled++
	return m.GetStatsHandler()
}
