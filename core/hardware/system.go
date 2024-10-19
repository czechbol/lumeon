package hardware

import (
	"log/slog"
	"os/exec"

	"github.com/czechbol/lumeon/core/hardware/i2c"
)

type System interface {
	Shutdown() error
	Halt() error
}

type systemImpl struct {
	bus i2c.I2CBus
}

func NewSystem(bus i2c.I2CBus) System {
	return systemImpl{
		bus: bus,
	}
}

// Shutdown cuts down the power to the system, hard drives and the whole case.
func (s systemImpl) Shutdown() error {
	slog.Warn("shutting down the system")

	return exec.Command("shutdown", "now").Run()
}

func (s systemImpl) Halt() error {
	slog.Warn("halting the system")

	return s.bus.SendData(daughterboardAddress, cmdSystemHalt)
}
