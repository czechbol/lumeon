package mock

import (
	"image"
	"image/gif"

	"github.com/czechbol/lumeon/core/hardware"
	"github.com/czechbol/lumeon/core/hardware/types"
)

// OLEDMock defines mocks for OLED.
type OLEDMock struct {
	InvertHandler                  func(blackOnWhite bool) error
	InvertHandlerCalled            int
	SetContrastHandler             func(brightness uint8) error
	SetContrastHandlerCalled       int
	ClearHandler                   func() error
	ClearHandlerCalled             int
	DrawImageHandler               func(img image.Image) error
	DrawImageHandlerCalled         int
	DrawGIFHandler                 func(gif *gif.GIF) error
	DrawGIFHandlerCalled           int
	DrawTextHandler                func(text string, x, y int) error
	DrawTextHandlerCalled          int
	DrawImageWithTextHandler       func(img image.Image, x, y int, text string) error
	DrawImageWithTextHandlerCalled int
	DrawGIFWithTextHandler         func(gif *gif.GIF, x, y int, text string) error
	DrawGIFWithTextHandlerCalled   int
	ScrollHandler                  func(direction types.ScrollDirection, rate types.FrameRate, startLine, endLine int) error
	ScrollHandlerCalled            int
}

var _ hardware.OLED = (*OLEDMock)(nil)

func (m *OLEDMock) Invert(blackOnWhite bool) error {
	m.InvertHandlerCalled++
	return m.InvertHandler(blackOnWhite)
}

func (m *OLEDMock) SetContrast(brightness uint8) error {
	m.SetContrastHandlerCalled++
	return m.SetContrastHandler(brightness)
}

func (m *OLEDMock) Clear() error {
	m.ClearHandlerCalled++
	return m.ClearHandler()
}

func (m *OLEDMock) DrawImage(img image.Image) error {
	m.DrawImageHandlerCalled++
	return m.DrawImageHandler(img)
}

func (m *OLEDMock) DrawGIF(gif *gif.GIF) error {
	m.DrawGIFHandlerCalled++
	return m.DrawGIFHandler(gif)
}

func (m *OLEDMock) DrawText(text string, x, y int) error {
	m.DrawTextHandlerCalled++
	return m.DrawTextHandler(text, x, y)
}

func (m *OLEDMock) DrawImageWithText(img image.Image, x, y int, text string) error {
	m.DrawImageWithTextHandlerCalled++
	return m.DrawImageWithTextHandler(img, x, y, text)
}

func (m *OLEDMock) DrawGIFWithText(gif *gif.GIF, x, y int, text string) error {
	m.DrawGIFWithTextHandlerCalled++
	return m.DrawGIFWithTextHandler(gif, x, y, text)
}

func (m *OLEDMock) Scroll(direction types.ScrollDirection, rate types.FrameRate, startLine, endLine int) error {
	m.ScrollHandlerCalled++
	return m.ScrollHandler(direction, rate, startLine, endLine)
}
