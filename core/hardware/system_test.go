package hardware

import (
	"testing"

	"github.com/czechbol/lumeon/core/hardware/i2c/mock"
	"github.com/stretchr/testify/suite"
)

type SystemTestSuite struct {
	suite.Suite
	busMock *mock.I2CBus
	system  System
}

func (s *SystemTestSuite) SetupTest() {
	s.busMock = &mock.I2CBus{}
	s.system = NewSystem(s.busMock)
}

func TestSystemTestSuite(t *testing.T) {
	suite.Run(t, new(SystemTestSuite))
}

func (s *SystemTestSuite) TestHalt() {
	s.busMock.SendDataHandler = func(addr uint16, data ...byte) error {
		s.Equal(daughterboardAddress, addr)
		s.Equal([]byte{cmdSystemHalt}, data)
		return nil
	}

	err := s.system.Halt()
	s.NoError(err)

	s.Equal(1, s.busMock.SendDataHandlerCalled)
}
