package mock

import "github.com/czechbol/lumeon/core/resources"

// HDDMock defines mocks for HDD.
type HDDMock struct {
	GetAverageTempHandler       func() (float64, error)
	GetAverageTempHandlerCalled int

	GetStatsHandler       func() ([]resources.HDDStats, error)
	GetStatsHandlerCalled int
}

var _ resources.HDD = (*HDDMock)(nil)

func (m *HDDMock) GetAverageTemp() (float64, error) {
	m.GetAverageTempHandlerCalled++
	return m.GetAverageTempHandler()
}

func (m *HDDMock) GetStats() ([]resources.HDDStats, error) {
	m.GetStatsHandlerCalled++
	return m.GetStatsHandler()
}
