package components

import (
	"testing"

	"github.com/czechbol/lumeon/core/hardware/components/i2c/mock"
	"github.com/czechbol/lumeon/core/hardware/constants"
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
	s.busMock.SendBytesHandler = func(addr uint16, bytes []byte) error {
		s.Equal(constants.I2C.Devices.Daughter, addr)
		s.Equal(constants.I2C.Commands.Halt, bytes)
		return nil
	}

	err := s.system.Halt()
	s.NoError(err)

	s.Equal(1, s.busMock.SendBytesHandlerCalled)
}
