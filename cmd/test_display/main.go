package main

import (
	"image"
	"image/color"
	"image/jpeg"
	"log/slog"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/czechbol/lumeon/core/hardware"
	"github.com/czechbol/lumeon/core/hardware/i2c"
	"gitlab.com/greyxor/slogor"
)

type img struct {
	img  *image.Gray
	name string
	file string
}

func main() {
	// Initialize logger
	logger := slog.New(slogor.NewHandler(os.Stderr, slogor.Options{
		TimeFormat: time.RFC3339,
		Level:      slog.LevelDebug,
		ShowSource: false,
	}))
	slog.SetDefault(logger)

	// Generate test images
	images := []img{
		{generateCheckerboardImage(128, 64), "checkerboard", "image1.jpg"},
		{generateVerticalStripesImage(128, 64), "v-stripes", "image2.jpg"},
		{generateHorizontalStripesImage(128, 64), "h-stripes", "image3.jpg"},
		{generateRandomImage(128, 64), "rand-dots", "image4.jpg"},
	}
	black := image.NewGray(image.Rect(0, 0, 128, 64))

	// Check if the runtime architecture is amd64
	if runtime.GOARCH == "amd64" {
		wd, err := os.Getwd()
		if err != nil {
			slog.Error("failed to get working directory", "error", err)
			os.Exit(1)
		}

		slog.Warn("Running on amd64 architecture, saving images to JPEG files")
		slog.Info("saving images to", "path", wd)

		saveImages(images)
		os.Exit(0)
	}

	// Initialize I2C bus
	i2cBus, err := i2c.NewBus("")
	if err != nil {
		slog.Error("failed to initialize i2c bus", "error", err)
		os.Exit(1)
	}

	// Initialize OLED display
	display, err := hardware.NewOLED(i2cBus)
	if err != nil {
		slog.Error("failed to initialize OLED display", "error", err)
		os.Exit(1)
	}

	// Display images in a loop for 20 seconds
	displayImages(display, images, 20*time.Second)

	// Clear the display
	if err := display.DrawImage(black); err != nil {
		slog.Error("failed to clear display", "error", err)
		os.Exit(1)
	}
}

func generateCheckerboardImage(width, height int) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if (x/8+y/8)%2 == 0 {
				img.Set(x, y, color.White)
			} else {
				img.Set(x, y, color.Black)
			}
		}
	}
	return img
}

func generateVerticalStripesImage(width, height int) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if (x/8)%2 == 0 {
				img.Set(x, y, color.White)
			} else {
				img.Set(x, y, color.Black)
			}
		}
	}
	return img
}

func generateHorizontalStripesImage(width, height int) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if (y/8)%2 == 0 {
				img.Set(x, y, color.White)
			} else {
				img.Set(x, y, color.Black)
			}
		}
	}
	return img
}

//nolint:gosec
func generateRandomImage(width, height int) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Randomly set the pixel to black or white
			if rand.Intn(2) == 0 {
				img.Set(x, y, color.White)
			} else {
				img.Set(x, y, color.Black)
			}
		}
	}
	return img
}

func saveImages(images []img) {
	for _, img := range images {
		if err := saveJPEG(img.img, img.name); err != nil {
			slog.Error("failed to save image", "name", img.name, "error", err)
		}
	}
}

func saveJPEG(img *image.Gray, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
}

//nolint:revive
func displayImages(display hardware.OLED, images []img, duration time.Duration) {
	startTime := time.Now()
	for time.Since(startTime) < duration {
		for _, img := range images {
			// Calculate the position to center the text
			textWidth := len(img.name) * 7
			x := (128 - textWidth) / 2

			if err := display.DrawImageWithText(img.img, x, 24, img.name); err != nil {
				slog.Error("failed to draw image", "name", img.name, "error", err)
				return
			}
			time.Sleep(1 * time.Second)
		}
	}
}
