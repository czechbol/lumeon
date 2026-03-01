package config

import (
	"log/slog"
	"time"
)

type Config interface {
	LogLevel() slog.Level
	FanConfig() FanConfig
	DisplayConfig() DisplayConfig
}

type configImpl struct {
	logLevel      slog.Level
	fanConfig     FanConfig
	displayConfig DisplayConfig
}

func NewConfig(logLevel slog.Level, fanConfig FanConfig, displayConfig DisplayConfig) Config {
	return &configImpl{
		logLevel:      logLevel,
		fanConfig:     fanConfig,
		displayConfig: displayConfig,
	}
}

func (c *configImpl) LogLevel() slog.Level {
	return c.logLevel
}

func (c *configImpl) FanConfig() FanConfig {
	return c.fanConfig
}

func (c *configImpl) DisplayConfig() DisplayConfig {
	return c.displayConfig
}

type DisplayConfig interface {
	Enabled() bool
	Interval() time.Duration
}

type displayConfigImpl struct {
	enabled  bool
	interval time.Duration
}

func NewDisplayConfig(enabled bool, interval time.Duration) DisplayConfig {
	return &displayConfigImpl{
		enabled:  enabled,
		interval: interval,
	}
}

func (d *displayConfigImpl) Enabled() bool {
	return d.enabled
}

func (d *displayConfigImpl) Interval() time.Duration {
	return d.interval
}

type FanConfig interface {
	Enabled() bool
	CPUCurve() []FanCurvePoint
	HDDCurve() []FanCurvePoint
}

type fanConfigImpl struct {
	enabled  bool
	cpuCurve []FanCurvePoint
	hddCurve []FanCurvePoint
}

func NewFanConfig(enabled bool, cpuCurve, hddCurve []FanCurvePoint) FanConfig {
	return &fanConfigImpl{
		enabled:  enabled,
		cpuCurve: cpuCurve,
		hddCurve: hddCurve,
	}
}

func (f *fanConfigImpl) Enabled() bool {
	return f.enabled
}

func (f *fanConfigImpl) CPUCurve() []FanCurvePoint {
	return f.cpuCurve
}

func (f *fanConfigImpl) HDDCurve() []FanCurvePoint {
	return f.hddCurve
}

type FanCurvePoint struct {
	Temperature uint8
	Speed       uint8
}

func NewFanCurvePoint(temperature, speed uint8) FanCurvePoint {
	return FanCurvePoint{
		Temperature: temperature,
		Speed:       speed,
	}
}
