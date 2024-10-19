package settings

import (
	"log/slog"
	"os"
	"sort"
	"strconv"

	"github.com/czechbol/lumeon/app/config"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Settings is the struct that holds the configuration for the application.
type Settings struct {
	LogLevel    string
	FanSettings FanSettings
}

// FanSettings is the struct that holds the configuration for the fan.
type FanSettings struct {
	Enabled  bool
	CPUCurve map[uint8]uint8
	HDDCurve map[uint8]uint8
}

func init() {
	viper.SetConfigName("lumeon")
	viper.SetConfigType("toml")
	viper.AddConfigPath("/etc/lumeon/")
	viper.AddConfigPath(".")
}

// LoadSettings loads the settings from the configuration file.
func GetConfig() config.Config {
	verbosityFlag := pflag.CountP("verbosity", "v", "verbosity level")
	pflag.Parse()

	err := viper.ReadInConfig()
	if err != nil {
		slog.Error("failed to read config file", "error", err)
		os.Exit(1)
	}

	// Get log level from config file
	logLevel := viper.GetString("logLevel")

	// Override log level if verbosity flag is set
	switch *verbosityFlag {
	case 1:
		logLevel = "warn"
	case 2:
		logLevel = "info"
	default:
		if *verbosityFlag > 2 {
			logLevel = "debug"
		}
	}

	return config.NewConfig(
		convertLogLevel(logLevel),
		config.NewFanConfig(
			viper.GetBool("fan.enabled"),
			stringMapStringToPointSlice(viper.GetStringMapString("fan.cpuCurve")),
			stringMapStringToPointSlice(viper.GetStringMapString("fan.hddCurve")),
		),
	)
}

func stringMapStringToPointSlice(input map[string]string) []config.FanCurvePoint {
	output := make([]config.FanCurvePoint, 0, len(input))
	for k, v := range input {
		key, err := strconv.Atoi(k)
		if err != nil {
			slog.Error("invalid temperature", "temp", k)
			os.Exit(1)
		}

		value, err := strconv.Atoi(v)
		if err != nil {
			slog.Error("invalid fan speed", "speed", v)
			os.Exit(1)
		}

		if key < 0 || key > 255 {
			slog.Error("temperatures must be between 0 and 255", "temp", key)
			os.Exit(1)
		}

		if value < 0 || value > 100 {
			slog.Error("fan speeds must be between 0 and 100", "speed", value)
			os.Exit(1)
		}

		output = append(output, config.FanCurvePoint{Temperature: uint8(key), Speed: uint8(value)})
	}

	sort.Slice(output, func(i, j int) bool {
		return output[i].Temperature < output[j].Temperature
	})

	return output
}

func convertLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
