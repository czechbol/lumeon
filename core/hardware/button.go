package hardware

import (
	"context"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
)

// ButtonEvent represents a button press event type.
type ButtonEvent int

const (
	// ButtonDoubleTap is a short 20–30ms pulse.
	ButtonDoubleTap ButtonEvent = iota
	// ButtonShutdown is a longer 40–50ms pulse.
	ButtonShutdown
)

const (
	pulsePollTimeout       = 200 * time.Millisecond
	doubleTapMinMS   int64 = 20
	doubleTapMaxMS   int64 = 30
	shutdownMinMS    int64 = 40
	shutdownMaxMS    int64 = 50
)

// Button is the interface for the power button.
type Button interface {
	WaitForEvent(ctx context.Context) (ButtonEvent, error)
}

type buttonImpl struct {
	pin gpio.PinIn
}

// NewButton initialises the GPIO4 pin as a pull-up input and returns a Button.
func NewButton() (Button, error) {
	pin := gpioreg.ByName("GPIO4")
	if pin == nil {
		return nil, ErrButtonPinNotFound
	}

	pinIn, ok := pin.(gpio.PinIn)
	if !ok {
		return nil, ErrButtonPinNotFound
	}

	if err := pinIn.In(gpio.PullUp, gpio.FallingEdge); err != nil {
		return nil, err
	}

	return &buttonImpl{pin: pinIn}, nil
}

// WaitForEvent blocks until a button press event is detected or ctx is cancelled.
func (b *buttonImpl) WaitForEvent(ctx context.Context) (ButtonEvent, error) {
	for {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}

		// Wait up to 100ms for a falling edge, then re-check ctx.
		if !b.pin.WaitForEdge(100 * time.Millisecond) {
			continue
		}

		event, ok := b.classifyPulse()
		if ok {
			return event, nil
		}
		// Unknown pulse duration — ignore and wait for the next event.
	}
}

// classifyPulse measures the button pulse duration and returns the event type.
func (b *buttonImpl) classifyPulse() (ButtonEvent, bool) {
	start := time.Now()

	// Poll until the pin goes high again (button released).
	for b.pin.Read() == gpio.Low {
		if time.Since(start) > pulsePollTimeout {
			break
		}
		time.Sleep(time.Millisecond)
	}

	ms := time.Since(start).Milliseconds()

	switch {
	case ms >= doubleTapMinMS && ms <= doubleTapMaxMS:
		return ButtonDoubleTap, true
	case ms >= shutdownMinMS && ms <= shutdownMaxMS:
		return ButtonShutdown, true
	default:
		return 0, false
	}
}
