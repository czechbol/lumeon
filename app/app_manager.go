package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// RunAndManageApp manages App lifecycle and handles signals.
func RunAndManageApp(app App) int {
	var exitCode int = 0
	app.Init()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server
	go func() {
		if err := app.Run(ctx); err != nil {
			slog.Error("application ended with error", slog.String("err", err.Error()))
			exitCode = 3
			cancel() // Cancel the context to signal termination
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for interrupt signal to gracefully shutdown the app
	select {
	case sig := <-quit:
		slog.Info(fmt.Sprintf("signal '%s' received, shutting down", sig))
		cancel() // Cancel the context to signal termination
	case <-ctx.Done():
		slog.Info("application terminated")
	}

	if err := app.Shutdown(ctx); err != nil {
		slog.Error(err.Error())
		exitCode = 3
	}

	slog.Info("bye")
	return exitCode
}
