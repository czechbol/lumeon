package hardware

import (
	"fmt"
	"log/slog"

	"github.com/anatol/smart.go"
	"github.com/jaypipes/ghw"
)

const (
	temperatureAttributeID = 194
)

type HDD interface {
	GetAverageTemp() (float64, error)
}

type hddImpl struct{}

func NewHDD() HDD {
	return &hddImpl{}
}

func (h *hddImpl) GetAverageTemp() (float64, error) {
	block, err := ghw.Block()
	if err != nil {
		return 0, fmt.Errorf("error getting block devices: %w", err)
	}

	total := 0.0
	count := 0

	for _, disk := range block.Disks {
		dev, err := smart.Open("/dev/" + disk.Name)
		if err != nil {
			slog.Debug("error opening device", "device", disk.Name, "error", err)
			continue
		}
		defer dev.Close()

		temp, err := getDeviceTemperature(dev)
		if err != nil {
			slog.Debug("error getting temperature", "device", disk.Name, "error", err)
			continue
		}

		total += temp
		count++
	}

	if count == 0 {
		return 0, ErrTemperatureNotFound
	}

	return total / float64(count), nil
}

func getDeviceTemperature(dev smart.Device) (float64, error) {
	attrs, err := dev.ReadGenericAttributes()
	if err != nil {
		return 0, err
	}

	if attrs.Temperature != 0 {
		return float64(attrs.Temperature), nil
	}

	// If generic attributes don't provide temperature, try device-specific methods
	switch d := dev.(type) {
	case *smart.SataDevice:
		data, err := d.ReadSMARTData()
		if err != nil {
			return 0, err
		}
		if attr, ok := data.Attrs[temperatureAttributeID]; ok {
			temp, _, _, _, err := attr.ParseAsTemperature()
			if err != nil {
				return 0, err
			}
			return float64(temp), nil
		}
	case *smart.NVMeDevice:
		sm, err := d.ReadSMART()
		if err != nil {
			return 0, err
		}
		return float64(sm.Temperature - 273), nil // Convert Kelvin to Celsius
	}

	return 0, ErrTemperatureNotFound
}
