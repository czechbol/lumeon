package core

import (
	"context"
	"log/slog"
	"sync"

	"github.com/czechbol/lumeon/core/hardware"
)

type ButtonService interface {
	IsRunning() bool
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

type buttonServiceImpl struct {
	mutex        sync.RWMutex
	running      bool
	button       hardware.Button
	system       hardware.System
	ctx          context.Context
	cancel       context.CancelFunc
	shutdownChan chan struct{}
}

func NewButtonService(button hardware.Button, system hardware.System) ButtonService {
	return &buttonServiceImpl{
		button:       button,
		system:       system,
		shutdownChan: make(chan struct{}),
	}
}

func (bs *buttonServiceImpl) IsRunning() bool {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()
	return bs.running
}

func (bs *buttonServiceImpl) Start(ctx context.Context) error {
	bs.ctx, bs.cancel = context.WithCancel(ctx)
	bs.mutex.Lock()
	if bs.running {
		bs.mutex.Unlock()
		return nil
	}
	bs.running = true
	bs.mutex.Unlock()

	slog.Info("starting button service")

	go bs.buttonLoop()

	return nil
}

func (bs *buttonServiceImpl) Shutdown(ctx context.Context) error {
	bs.cancel()

	select {
	case <-bs.shutdownChan:
		slog.Info("button service stopped gracefully")
	case <-ctx.Done():
		slog.Warn("shutdown context expired before button service could stop")
	}

	bs.mutex.Lock()
	bs.running = false
	bs.mutex.Unlock()

	return nil
}

func (bs *buttonServiceImpl) buttonLoop() {
	defer close(bs.shutdownChan)

	for {
		event, err := bs.button.WaitForEvent(bs.ctx)
		if err != nil {
			if bs.ctx.Err() != nil {
				slog.Info("stopping button service due to context cancellation")
				return
			}
			slog.Error("error waiting for button event", "error", err)
			continue
		}

		switch event {
		case hardware.ButtonDoubleTap:
			slog.Info("button double tap detected")
		case hardware.ButtonShutdown:
			slog.Warn("button shutdown press detected, halting system")
			if err := bs.system.Halt(); err != nil {
				slog.Error("failed to halt system", "error", err)
			}
		}
	}
}
