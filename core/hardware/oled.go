package hardware

import (
	"image"
	"image/draw"
	"image/gif"
	_ "image/png"
	"log/slog"
	"time"
	"unicode/utf8"

	"github.com/czechbol/lumeon/core/hardware/i2c"
	"github.com/czechbol/lumeon/core/hardware/types"
	"periph.io/x/conn/v3/display"
	"periph.io/x/devices/v3/ssd1306"
	"periph.io/x/devices/v3/ssd1306/image1bit"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type OLED interface {
	Invert(blackOnWhite bool) error
	SetContrast(brightness uint8) error
	Clear() error
	DrawImage(img image.Image) error
	DrawGIF(gif *gif.GIF, fps int) error
	DrawText(text string, x, y int) error
	DrawImageWithText(img image.Image, x, y int, text string) error
	DrawGIFWithText(gif *gif.GIF, rate types.FrameRate, x, y int, text string) error
	Scroll(direction types.ScrollDirection, rate types.FrameRate, startLine, endLine int) error
}

type oledI2cImpl struct {
	dev *ssd1306.Dev
}

func NewOLED(i2cBus i2c.I2CBus) (*oledI2cImpl, error) {
	slog.Info("Initializing OLED display")
	dev, err := ssd1306.NewI2C(i2cBus.GetBus(), &ssd1306.Opts{
		W: displayWidth,
		H: displayHeight,
	})
	if err != nil {
		return nil, err
	}

	return &oledI2cImpl{dev: dev}, nil
}

func (o *oledI2cImpl) drawImage(img image.Image) error {
	mirroredImg := mirrorImage(img)
	return o.dev.Draw(o.dev.Bounds(), mirroredImg, image.Point{})
}

func (o *oledI2cImpl) Invert(blackOnWhite bool) error {
	slog.Debug("Inverting display", "blackOnWhite", blackOnWhite)
	return o.dev.Invert(blackOnWhite)
}

// SetContrast sets the display brightness.
func (o *oledI2cImpl) SetContrast(brightness uint8) error {
	slog.Debug("Setting contrast", "brightness", brightness)
	return o.dev.SetContrast(brightness)
}

func (o *oledI2cImpl) Clear() error {
	slog.Debug("Clearing display")
	return o.dev.Halt()
}

func (o *oledI2cImpl) DrawImage(img image.Image) error {
	slog.Debug("Drawing image")
	convertedImg := convert(o.dev, img)

	return o.drawImage(convertedImg)
}

func (o *oledI2cImpl) DrawGIF(gif *gif.GIF, fps int) error {
	slog.Debug("Drawing GIF", "fps", fps)
	img := image1bit.NewVerticalLSB(o.dev.Bounds())
	for _, frame := range gif.Image {
		convertedImg := convert(o.dev, frame)

		draw.Draw(img, img.Bounds(), convertedImg, image.Point{}, draw.Over)
		err := o.drawImage(img)
		if err != nil {
			return err
		}

		time.Sleep(time.Second / time.Duration(fps))
	}
	return nil
}

func (o *oledI2cImpl) DrawText(text string, x, y int) error {
	slog.Debug("Drawing text", "text", text, "x", x, "y", y)
	img := image1bit.NewVerticalLSB(o.dev.Bounds())
	addLabel(img, x, y, text)

	return o.drawImage(img)
}

func (o *oledI2cImpl) DrawImageWithText(img image.Image, x, y int, text string) error {
	slog.Debug("Drawing image with text", "text", text, "x", x, "y", y)
	convertedImg := convert(o.dev, img)
	addLabel(convertedImg, x, y, text)
	return o.drawImage(convertedImg)
}

func (o *oledI2cImpl) DrawGIFWithText(gif *gif.GIF, rate types.FrameRate, x, y int, text string) error {
	slog.Debug("Drawing GIF with text", "text", text, "rate", rate, "x", x, "y", y)
	img := image1bit.NewVerticalLSB(o.dev.Bounds())
	for _, frame := range gif.Image {
		convertedImg := convert(o.dev, frame)
		addLabel(convertedImg, x, y, text)

		draw.Draw(img, img.Bounds(), convertedImg, image.Point{}, draw.Over)

		err := o.drawImage(img)
		if err != nil {
			return err
		}

		time.Sleep(time.Second / time.Duration(frameRateToFPS(rate)))
	}
	return nil
}

func (o *oledI2cImpl) Scroll(direction types.ScrollDirection, rate types.FrameRate, startLine, endLine int) error {
	slog.Debug("Scrolling display", "direction", direction, "rate", rate, "startLine", startLine, "endLine", endLine)
	return o.dev.Scroll(ssd1306.Orientation(direction), ssd1306.FrameRate(rate), startLine, endLine)
}

func (o *oledI2cImpl) StopScroll() error {
	slog.Debug("Stopping scroll")
	return o.dev.StopScroll()
}

// resize scales the source image to the given size.
func resize(src image.Image, size image.Point) *image.NRGBA {
	srcMax := src.Bounds().Max
	dst := image.NewNRGBA(image.Rectangle{Max: size})
	for y := 0; y < size.Y; y++ {
		sY := (y*srcMax.Y + size.Y/2) / size.Y
		for x := 0; x < size.X; x++ {
			dst.Set(x, y, src.At((x*srcMax.X+size.X/2)/size.X, sY))
		}
	}
	return dst
}

// convert prepares the image for the OLED display format.
func convert(disp display.Drawer, src image.Image) *image1bit.VerticalLSB {
	slog.Debug("Converting image for display")
	screenBounds := disp.Bounds()
	size := screenBounds.Size()
	src = resize(src, size)
	img := image1bit.NewVerticalLSB(screenBounds)
	r := src.Bounds()
	r = r.Add(image.Point{(size.X - r.Max.X) / 2, (size.Y - r.Max.Y) / 2})
	draw.Draw(img, r, src, image.Point{}, draw.Src)
	return img
}

// addLabel draws text onto the image at the specified coordinates.
func addLabel(img draw.Image, x, y int, label string) {
	slog.Debug("Adding label to image", "label", label, "x", x, "y", y)
	col := image1bit.On
	bgCol := image1bit.Off

	face := basicfont.Face7x13
	textWidth := utf8.RuneCountInString(label) * 7 // Approximate width per character
	textHeight := face.Metrics().Height.Ceil()

	// Draw black rectangle as background
	draw.Draw(img, image.Rect(x, y, x+textWidth, y+textHeight), image.NewUniform(bgCol), image.Point{}, draw.Src)

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y + face.Metrics().Ascent.Ceil())},
	}
	d.DrawString(label)
}

func mirrorImage(img image.Image) image.Image {
	bounds := img.Bounds()
	mirrored := image.NewGray(bounds)
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			mirrored.Set(bounds.Dx()-x, y, img.At(x, y))
		}
	}
	return mirrored
}

//nolint:revive
func frameRateToFPS(rate types.FrameRate) int {
	switch rate {
	case types.FrameRate2:
		return 2
	case types.FrameRate3:
		return 3
	case types.FrameRate4:
		return 4
	case types.FrameRate5:
		return 5
	case types.FrameRate25:
		return 25
	case types.FrameRate64:
		return 64
	case types.FrameRate128:
		return 128
	case types.FrameRate256:
		return 256
	default:
		return 25
	}
}
