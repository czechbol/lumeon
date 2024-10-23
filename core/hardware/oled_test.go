package hardware

import (
	"image"
	"image/gif"
	"testing"

	"github.com/czechbol/lumeon/core/hardware/i2c/mock"
	"github.com/czechbol/lumeon/core/hardware/types"
	"github.com/stretchr/testify/suite"
	i2clib "periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2ctest"
)

type OLEDTestSuite struct {
	suite.Suite
	busMock   *mock.I2CBus
	oled      *oledI2cImpl
	recordBus *i2ctest.Record
}

func (s *OLEDTestSuite) SetupTest() {
	s.busMock = &mock.I2CBus{}
	s.recordBus = &i2ctest.Record{}

	s.busMock.GetBusHandler = func() i2clib.Bus {
		return s.recordBus
	}

	oled, err := NewOLED(s.busMock)
	s.NoError(err)
	s.oled = oled

	s.Equal(1, s.busMock.GetBusCalled)
}

func TestOLEDTestSuite(t *testing.T) {
	suite.Run(t, new(OLEDTestSuite))
}

func (s *OLEDTestSuite) TestInvert() {
	err := s.oled.Invert(true)
	s.NoError(err)

	s.Len(s.recordBus.Ops, 2)
}

func (s *OLEDTestSuite) TestSetContrast() {
	err := s.oled.SetContrast(0x7F)
	s.NoError(err)

	s.Len(s.recordBus.Ops, 2)
}

func (s *OLEDTestSuite) TestClear() {
	err := s.oled.Clear()
	s.NoError(err)

	s.Len(s.recordBus.Ops, 2)
}

func (s *OLEDTestSuite) TestDrawImage() {
	img := image.NewGray(image.Rect(0, 0, 128, 64))

	err := s.oled.DrawImage(img)
	s.NoError(err)

	s.Len(s.recordBus.Ops, 17)
}

func (s *OLEDTestSuite) TestDrawGIF() {
	g := &gif.GIF{
		Image: []*image.Paletted{
			image.NewPaletted(image.Rect(0, 0, 128, 64), nil),
			image.NewPaletted(image.Rect(0, 0, 128, 64), nil),
		},
		Delay: []int{10},
	}

	err := s.oled.DrawGIF(g)
	s.NoError(err)
}

func (s *OLEDTestSuite) TestDrawText() {
	err := s.oled.DrawText("Hello", 0, 0)
	s.NoError(err)
}

func (s *OLEDTestSuite) TestDrawImageWithText() {
	img := image.NewGray(image.Rect(0, 0, 128, 64))
	err := s.oled.DrawImageWithText(img, 0, 0, "Hello")
	s.NoError(err)
}

func (s *OLEDTestSuite) TestDrawGIFWithText() {
	g := &gif.GIF{
		Image: []*image.Paletted{
			image.NewPaletted(image.Rect(0, 0, 128, 64), nil),
		},
		Delay: []int{10},
	}

	err := s.oled.DrawGIFWithText(g, 0, 0, "Hello")
	s.NoError(err)
}

func (s *OLEDTestSuite) TestScroll() {
	err := s.oled.Scroll(types.ScrollRight, types.FrameRate5, 0, 63)
	s.NoError(err)
}

func (s *OLEDTestSuite) TestStopScroll() {
	err := s.oled.StopScroll()
	s.NoError(err)
}
