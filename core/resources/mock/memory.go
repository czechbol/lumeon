package mock

import "github.com/czechbol/lumeon/core/resources"

// MemoryMock defines mocks for Memory.
type MemoryMock struct {
	GetStatsHandler       func() (*resources.MemoryStats, error)
	GetStatsHandlerCalled int
}

var _ resources.Memory = (*MemoryMock)(nil)

func (m *MemoryMock) GetStats() (*resources.MemoryStats, error) {
	m.GetStatsHandlerCalled++
	return m.GetStatsHandler()
}
