package hardware

import (
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"log/slog"
	"time"
	"unicode/utf8"

	"github.com/czechbol/lumeon/core/hardware/i2c"
	"github.com/czechbol/lumeon/core/hardware/types"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"periph.io/x/conn/v3/display"
	"periph.io/x/devices/v3/ssd1306"
	"periph.io/x/devices/v3/ssd1306/image1bit"
)

type OLED interface {
	Invert(blackOnWhite bool) error
	SetContrast(brightness uint8) error
	Clear() error
	DrawImage(img image.Image) error
	DrawGIF(gif *gif.GIF) error
	DrawText(text string, x, y int) error
	DrawImageWithText(img image.Image, x, y int, text string) error
	DrawGIFWithText(gif *gif.GIF, x, y int, text string) error
	Scroll(direction types.ScrollDirection, rate types.FrameRate, startLine, endLine int) error
}

type oledI2cImpl struct {
	dev *ssd1306.Dev
}

func NewOLED(i2cBus i2c.I2CBus) (*oledI2cImpl, error) {
	slog.Info("Initializing OLED display")
	dev, err := ssd1306.NewI2C(i2cBus.GetBus(), &ssd1306.Opts{
		W:                displayWidth,
		H:                displayHeight,
		MirrorHorizontal: true,
	})
	if err != nil {
		return nil, err
	}

	return &oledI2cImpl{dev: dev}, nil
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

	return o.dev.Draw(o.dev.Bounds(), convertedImg, image.Point{})
}

func (o *oledI2cImpl) DrawGIF(gif *gif.GIF) error {
	slog.Debug("Preparing GIF")

	convertedGIF := convertGIF(o.dev, gif)

	slog.Debug("Drawing GIF")
	for i := 0; gif.LoopCount <= 0 || i < gif.LoopCount*len(gif.Image); i++ {
		index := i % len(gif.Image)
		c := time.After(time.Duration(10*gif.Delay[index]) * time.Millisecond)
		img := convertedGIF.Image[index]
		err := o.dev.Draw(o.dev.Bounds(), img, image.Point{})
		if err != nil {
			return err
		}
		<-c
	}
	return nil
}

func (o *oledI2cImpl) DrawText(text string, x, y int) error {
	slog.Debug("Drawing text", "text", text, "x", x, "y", y)
	img := image1bit.NewVerticalLSB(o.dev.Bounds())
	addLabel(img, x, y, text)

	return o.dev.Draw(o.dev.Bounds(), img, image.Point{})
}

func (o *oledI2cImpl) DrawImageWithText(img image.Image, x, y int, text string) error {
	slog.Debug("Drawing image with text", "text", text, "x", x, "y", y)
	convertedImg := convert(o.dev, img)
	addLabel(convertedImg, x, y, text)
	return o.dev.Draw(o.dev.Bounds(), convertedImg, image.Point{})
}

func (o *oledI2cImpl) DrawGIFWithText(gif *gif.GIF, x, y int, text string) error {
	slog.Debug("Preparing GIF with text", "text", text, "x", x, "y", y)

	// preprocess the GIF to save on resources during rendering
	convertedGIF := convertGIF(o.dev, gif)
	for i := range convertedGIF.Image {
		addLabel(convertedGIF.Image[i], x, y, text)
	}

	slog.Debug("Drawing GIF")
	for i := 0; gif.LoopCount <= 0 || i < gif.LoopCount*len(gif.Image); i++ {
		index := i % len(gif.Image)
		c := time.After(time.Duration(10*gif.Delay[index]) * time.Millisecond)
		img := convertedGIF.Image[index]
		err := o.dev.Draw(o.dev.Bounds(), img, image.Point{})
		if err != nil {
			return err
		}
		<-c
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
	screenBounds := disp.Bounds()
	size := screenBounds.Size()

	// Resize the image if necessary
	if !src.Bounds().Size().Eq(size) {
		src = resize(src, size)
	}

	// Convert the image to grayscale
	grayImg := image.NewGray(src.Bounds())
	draw.Draw(grayImg, src.Bounds(), src, image.Point{}, draw.Src)

	// Convert the grayscale image to monochrome
	monoImg := image1bit.NewVerticalLSB(screenBounds)
	for y := 0; y < grayImg.Bounds().Dy(); y++ {
		for x := 0; x < grayImg.Bounds().Dx(); x++ {
			grayColor := grayImg.GrayAt(x, y)
			if grayColor.Y > 128 {
				monoImg.Set(x, y, color.White)
			} else {
				monoImg.Set(x, y, color.Black)
			}
		}
	}

	return monoImg
}

func convertGIF(disp display.Drawer, g *gif.GIF) *gif.GIF {
	// Create a new GIF to store the fully rendered frames
	newGIF := &gif.GIF{
		LoopCount: g.LoopCount,
		Delay:     make([]int, len(g.Image)),
		Disposal:  make([]byte, len(g.Image)),
	}

	// Create a canvas to build up the frames
	canvas := image.NewGray(g.Image[0].Bounds())

	for i, srcImg := range g.Image {
		// Start with a clean canvas if disposal method is 2 (RestoreBGColor)
		if i > 0 && g.Disposal[i-1] == gif.DisposalBackground {
			draw.Draw(canvas, canvas.Bounds(), image.Transparent, image.Point{}, draw.Src)
		}

		// Draw this frame onto the canvas
		draw.Draw(canvas, srcImg.Bounds(), srcImg, srcImg.Bounds().Min, draw.Over)

		// Create a new paletted image for this frame
		palettedImage := image.NewPaletted(canvas.Bounds(), srcImg.Palette)
		draw.Draw(palettedImage, palettedImage.Bounds(), convert(disp, canvas), image.Point{}, draw.Src)

		// Add the new frame to our GIF
		newGIF.Image = append(newGIF.Image, palettedImage)
		newGIF.Delay[i] = g.Delay[i]
		newGIF.Disposal[i] = gif.DisposalNone // Since each frame is now complete
	}

	return newGIF
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
