package mock

import "github.com/czechbol/lumeon/core/hardware"

// HDDMock defines mocks for HDD.
type HDDMock struct {
	GetAverageTempHandler       func() (float64, error)
	GetAverageTempHandlerCalled int
}

var _ hardware.HDD = (*HDDMock)(nil)

func (m *HDDMock) GetAverageTemp() (float64, error) {
	m.GetAverageTempHandlerCalled++
	return m.GetAverageTempHandler()
}
