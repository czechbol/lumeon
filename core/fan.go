package core

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/czechbol/lumeon/app/config"
	"github.com/czechbol/lumeon/core/hardware"
	"github.com/czechbol/lumeon/core/resources"
)

type FanService interface {
	IsRunning() bool
	Start(ctx context.Context) error
	Shutdown(context.Context) error
}

type fanServiceImpl struct {
	mutex        sync.RWMutex
	running      bool
	fan          hardware.Fan
	cpu          resources.CPU
	drives       resources.HDD
	fanConfig    config.FanConfig
	ctx          context.Context
	cancel       context.CancelFunc
	shutdownChan chan struct{}
}

func NewFanService(fan hardware.Fan, cpu resources.CPU, drives resources.HDD, fanConfig config.FanConfig) FanService {
	return &fanServiceImpl{
		fan:          fan,
		cpu:          cpu,
		drives:       drives,
		fanConfig:    fanConfig,
		shutdownChan: make(chan struct{}),
	}
}

func (fs *fanServiceImpl) IsRunning() bool {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()
	return fs.running
}

func (fs *fanServiceImpl) Start(ctx context.Context) error {
	fs.ctx, fs.cancel = context.WithCancel(ctx)
	fs.mutex.Lock()
	if fs.running {
		fs.mutex.Unlock()
		return nil
	}
	fs.running = true
	fs.mutex.Unlock()

	slog.Info("starting fan loop")

	go fs.fanLoop()

	return nil
}

func (fs *fanServiceImpl) Shutdown(ctx context.Context) error {
	fs.cancel()

	select {
	case <-fs.shutdownChan:
		slog.Info("fan loop stopped gracefully")
	case <-ctx.Done():
		slog.Warn("shutdown context expired before fan loop could stop")
	}

	fs.mutex.Lock()
	fs.running = false
	fs.mutex.Unlock()

	return nil
}

func (fs *fanServiceImpl) fanLoop() {
	defer close(fs.shutdownChan)

	var currentSpeed uint8
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		var err error
		currentSpeed, err = fs.adjustFanSpeed(currentSpeed)
		if err != nil {
			slog.Error("failed to adjust fan speed", "error", err)
		}

		select {
		case <-fs.ctx.Done():
			slog.Info("stopping fan loop due to context cancellation")
			return
		case <-ticker.C:
			// Continue to the next iteration
		}
	}
}

func (fs *fanServiceImpl) adjustFanSpeed(currentSpeed uint8) (uint8, error) {
	tempRequestedByCPU := fs.getCPUFanSpeed()
	tempRequestedByDrives := fs.getDriveFanSpeed()

	speed := max(tempRequestedByCPU, tempRequestedByDrives)

	if speed != currentSpeed {
		slog.Info("altering fan speed", "speed", speed)
		if err := fs.fan.SetSpeed(speed); err != nil {
			slog.Error("Failed to set fan speed", "error", err)
			return currentSpeed, err
		}
		currentSpeed = speed
	} else {
		slog.Info("requested fan speed did not change", "current", currentSpeed)
	}

	return currentSpeed, nil
}

func (fs *fanServiceImpl) getCPUFanSpeed() uint8 {
	slog.Debug("obtaining fan speed from CPU temp curve")

	temp, err := fs.cpu.GetAverageTemp()
	if err != nil {
		slog.Error("Failed to get CPU temperature", "error", err)
		return 100
	}

	var speed uint8
	for _, point := range fs.fanConfig.CPUCurve() {
		if temp < 0 {
			slog.Warn("CPU temperature is less than zero", "temperature", temp)
		}
		if uint8(temp) > point.Temperature {
			speed = point.Speed
		} else {
			break
		}
	}
	return speed
}

func (fs *fanServiceImpl) getDriveFanSpeed() uint8 {
	slog.Debug("obtaining fan speed from HDD temp curve")

	temp, err := fs.drives.GetAverageTemp()
	if err != nil {
		slog.Error("Failed to get drive temperature", "error", err)
		return 100
	}

	var speed uint8
	for _, point := range fs.fanConfig.HDDCurve() {
		if temp < 0 {
			slog.Warn("Drive temperature is less than zero", "temperature", temp)
		}
		if uint8(temp) > point.Temperature {
			speed = point.Speed
		} else {
			break
		}
	}
	return speed
}
