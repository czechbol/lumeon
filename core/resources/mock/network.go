package mock

import "github.com/czechbol/lumeon/core/resources"

// NetworkMock defines mocks for Network.
type NetworkMock struct {
	GetInterfaceStatsHandler       func(iface string) (*resources.NetworkStats, error)
	GetInterfaceStatsHandlerCalled int

	GetAllInterfaceStatsHandler       func() (map[string]*resources.NetworkStats, error)
	GetAllInterfaceStatsHandlerCalled int
}

var _ resources.Network = (*NetworkMock)(nil)

func (m *NetworkMock) GetInterfaceStats(iface string) (*resources.NetworkStats, error) {
	m.GetInterfaceStatsHandlerCalled++
	return m.GetInterfaceStatsHandler(iface)
}

func (m *NetworkMock) GetAllInterfaceStats() (map[string]*resources.NetworkStats, error) {
	m.GetAllInterfaceStatsHandlerCalled++
	return m.GetAllInterfaceStatsHandler()
}
