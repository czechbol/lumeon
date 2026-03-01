/*
Package app initializes core application.
Application is managed by RunAndManageApp.
*/
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/czechbol/lumeon/app/config"
	"github.com/czechbol/lumeon/core"
	"github.com/czechbol/lumeon/core/hardware"
	"github.com/czechbol/lumeon/core/hardware/i2c"
	"github.com/czechbol/lumeon/core/resources"
	"gitlab.com/greyxor/slogor"
)

const (
	shutdownTimeoutSec        = 10
	traceFinishTimeoutSeconds = 4
	asciiArt                  = "    __                    __________  _   __\n   / /   __  ______ ___  / ____/ __ \\/ | / /\n  / /   / / / / __ `__ \\/ __/ / / / /  |/ / \n / /___/ /_/ / / / / / / /___/ /_/ / /|  /  \n/_____/\\__,_/_/ /_/ /_/_____/\\____/_/ |_/   \n\n"
)

var ErrShutdownFailed = errors.New("app shutdown failed")

// App interface.
type App interface {
	Init()
	Run(context.Context) error
	Shutdown(ctx context.Context) error
}

// CoreApp implements App interface.
type CoreApp struct {
	config       config.Config
	coreServices *core.CoreServices
}

// NewCoreApp constructs App.
func NewApp(config config.Config) *CoreApp {
	return &CoreApp{
		config: config,
	}
}

// Init initializes the App.
func (app *CoreApp) Init() {
	// set logger
	logger := slog.New(slogor.NewHandler(os.Stderr, slogor.Options{
		TimeFormat: "2006-01-02 15:04:05.000",
		Level:      app.config.LogLevel(),
		ShowSource: false,
	}))

	slog.SetDefault(logger)

	fmt.Print(asciiArt)

	slog.Info(fmt.Sprintf("starting %s", serviceName), "version", version, "commit", gitCommit, "buildDate", buildDate)

	archCheck()

	i2cBus, err := i2c.NewBus("")
	if err != nil {
		slog.Error("failed to initialize i2c bus", "error", err)
		os.Exit(1)
	}

	oled, err := hardware.NewOLED(i2cBus)
	if err != nil {
		slog.Error("failed to initialize OLED display", "error", err)
		os.Exit(1)
	}

	cpu := resources.NewCPU()
	drives := resources.NewHDD()

	services := &core.CoreServices{
		FanService: core.NewFanService(
			hardware.NewFan(i2cBus),
			cpu,
			drives,
			app.config.FanConfig(),
		),
		DisplayService: core.NewDisplayService(
			oled,
			cpu,
			resources.NewMemory(),
			resources.NewNetwork(),
			drives,
			app.config.DisplayConfig(),
		),
	}

	button, err := hardware.NewButton()
	if err != nil {
		slog.Warn("button not available, skipping button service", "error", err)
	} else {
		services.ButtonService = core.NewButtonService(button, services.DisplayService)
	}

	app.coreServices = services
}

// Run the App.
func (app *CoreApp) Run(ctx context.Context) error {
	if err := app.coreServices.FanService.Start(ctx); err != nil {
		return err
	}
	if err := app.coreServices.DisplayService.Start(ctx); err != nil {
		return err
	}
	if app.coreServices.ButtonService != nil {
		if err := app.coreServices.ButtonService.Start(ctx); err != nil {
			return err
		}
	}

	<-ctx.Done()

	return nil
}

// Shutdown the App.
func (app *CoreApp) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, shutdownTimeoutSec*time.Second)
	defer cancel()

	slog.Info("stopping fan loop")
	if err := app.coreServices.FanService.Shutdown(ctx); err != nil {
		slog.Error("failed to stop fan loop", "error", err)
	}

	slog.Info("stopping display service")
	if err := app.coreServices.DisplayService.Shutdown(ctx); err != nil {
		slog.Error("failed to stop display service", "error", err)
	}

	if app.coreServices.ButtonService != nil {
		slog.Info("stopping button service")
		if err := app.coreServices.ButtonService.Shutdown(ctx); err != nil {
			slog.Error("failed to stop button service", "error", err)
		}
	}

	return nil
}

func archCheck() {
	// Check if the architecture is supported
	if runtime.GOARCH != "arm64" && runtime.GOARCH != "arm" {
		slog.Error("this application is only supported on arm64 and arm architectures")
		os.Exit(1)
	}
}
