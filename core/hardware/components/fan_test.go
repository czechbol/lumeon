package components

import (
	"fmt"
	"testing"

	"github.com/czechbol/lumeon/core/hardware"
	"github.com/czechbol/lumeon/core/hardware/components/i2c/mock"
	"github.com/czechbol/lumeon/core/hardware/constants"
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
	s.busMock.SendBytesHandler = func(addr uint16, bytes []byte) error {
		s.Equal(constants.I2C.Devices.Daughter, addr)
		s.Equal([]byte{50}, bytes)
		return nil
	}

	err := s.fan.SetSpeed(50)
	s.NoError(err)

	s.Equal(1, s.busMock.SendBytesHandlerCalled)
}

func (s *FanTestSuite) TestSetSpeedInvalid() {
	err := s.fan.SetSpeed(150)
	s.Error(err)
	s.Equal(fmt.Errorf("%w: speed is specified in percent: 0 to 100", hardware.ErrInvalidFanSpeed), err)
	s.Equal(0, s.busMock.SendBytesHandlerCalled)
}
