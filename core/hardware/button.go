package hardware

import (
	"context"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/host/v3"
	"periph.io/x/host/v3/bcm283x"
)

// ButtonEvent represents a button press event type.
type ButtonEvent int

const (
	// ButtonPress is fired each time the power button is pressed.
	// The Argon40 EON hardware reports both edges only on button release,
	// so duration-based classification is not possible.
	ButtonPress ButtonEvent = iota
)

// Button is the interface for the power button.
type Button interface {
	WaitForEvent(ctx context.Context) (ButtonEvent, error)
}

type buttonImpl struct {
	pin gpio.PinIn
}

// NewButton initialises GPIO4 as a pull-down falling-edge input and returns a Button.
func NewButton() (Button, error) {
	if _, err := host.Init(); err != nil {
		return nil, err
	}

	pin := bcm283x.GPIO4
	if pin == nil {
		return nil, ErrButtonPinNotFound
	}

	if err := pin.In(gpio.PullDown, gpio.FallingEdge); err != nil {
		return nil, err
	}

	return &buttonImpl{pin: pin}, nil
}

// WaitForEvent blocks until a button press is detected or ctx is cancelled.
// It polls with a short timeout so context cancellation is handled promptly.
func (b *buttonImpl) WaitForEvent(ctx context.Context) (ButtonEvent, error) {
	for {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}

		if b.pin.WaitForEdge(100 * time.Millisecond) {
			return ButtonPress, nil
		}
	}
}
