package mock

import (
	"context"

	"github.com/czechbol/lumeon/core/hardware"
)

// ButtonMock defines mocks for Button.
type ButtonMock struct {
	WaitForEventHandler       func(ctx context.Context) (hardware.ButtonEvent, error)
	WaitForEventHandlerCalled int
}

var _ hardware.Button = (*ButtonMock)(nil)

func (m *ButtonMock) WaitForEvent(ctx context.Context) (hardware.ButtonEvent, error) {
	m.WaitForEventHandlerCalled++
	return m.WaitForEventHandler(ctx)
}
