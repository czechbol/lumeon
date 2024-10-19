package types

import "periph.io/x/devices/v3/ssd1306"

type ScrollDirection ssd1306.Orientation

const (
	ScrollRight   ScrollDirection = ScrollDirection(ssd1306.Right)
	ScrollLeft    ScrollDirection = ScrollDirection(ssd1306.Left)
	ScrollUpLeft  ScrollDirection = ScrollDirection(ssd1306.UpLeft)
	ScrollUpRight ScrollDirection = ScrollDirection(ssd1306.UpRight)
)

type FrameRate ssd1306.FrameRate

const (
	FrameRate2   FrameRate = FrameRate(ssd1306.FrameRate2)
	FrameRate3   FrameRate = FrameRate(ssd1306.FrameRate3)
	FrameRate4   FrameRate = FrameRate(ssd1306.FrameRate4)
	FrameRate5   FrameRate = FrameRate(ssd1306.FrameRate5)
	FrameRate25  FrameRate = FrameRate(ssd1306.FrameRate25)
	FrameRate64  FrameRate = FrameRate(ssd1306.FrameRate64)
	FrameRate128 FrameRate = FrameRate(ssd1306.FrameRate128)
	FrameRate256 FrameRate = FrameRate(ssd1306.FrameRate256)
)
