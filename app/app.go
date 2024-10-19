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
		TimeFormat: time.RFC3339,
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

	app.coreServices = &core.CoreServices{
		FanService: core.NewFanService(
			hardware.NewFan(i2cBus),
			hardware.NewCPU(),
			hardware.NewHDD(),
			app.config.FanConfig(),
		),
	}
}

// Run the App.
func (app *CoreApp) Run(ctx context.Context) error {
	err := app.coreServices.FanService.Start(ctx)
	if err != nil {
		return err
	}

	// Create a channel to signal when to stop
	stopChan := make(chan struct{})

	// Wait for a signal to stop
	<-stopChan

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

	return nil
}

func archCheck() {
	// Check if the architecture is supported
	if runtime.GOARCH != "arm64" && runtime.GOARCH != "arm" {
		slog.Error("this application is only supported on arm64 and arm architectures")
		os.Exit(1)
	}
}
