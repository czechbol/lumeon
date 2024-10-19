package config

import "log/slog"

type Config interface {
	LogLevel() slog.Level
	FanConfig() FanConfig
}

type configImpl struct {
	logLevel  slog.Level
	fanConfig FanConfig
}

func NewConfig(logLevel slog.Level, fanConfig FanConfig) Config {
	return &configImpl{
		logLevel:  logLevel,
		fanConfig: fanConfig,
	}
}

func (c *configImpl) LogLevel() slog.Level {
	return c.logLevel
}

func (c *configImpl) FanConfig() FanConfig {
	return c.fanConfig
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
