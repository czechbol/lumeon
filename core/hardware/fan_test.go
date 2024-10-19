package hardware

import (
	"testing"

	"github.com/czechbol/lumeon/core/hardware/i2c/mock"
	"github.com/stretchr/testify/suite"
)

type FanTestSuite struct {
	suite.Suite
	busMock *mock.I2CBus
	fan     Fan
}

func (s *FanTestSuite) SetupTest() {
	s.busMock = &mock.I2CBus{}
	s.fan = NewFan(s.busMock)
}

func TestFanTestSuite(t *testing.T) {
	suite.Run(t, new(FanTestSuite))
}

func (s *FanTestSuite) TestSetSpeed() {
	s.busMock.SendDataHandler = func(addr uint16, data ...byte) error {
		s.Equal(daughterboardAddress, addr)
		s.Equal([]byte{50}, data)
		return nil
	}

	err := s.fan.SetSpeed(50)
	s.NoError(err)

	s.Equal(1, s.busMock.SendDataHandlerCalled)
}

func (s *FanTestSuite) TestSetSpeedInvalid() {
	err := s.fan.SetSpeed(150)
	s.Error(err)
	s.ErrorIs(err, ErrInvalidFanSpeed)

	s.Equal(0, s.busMock.SendDataHandlerCalled)
}
