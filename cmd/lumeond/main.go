package main

import (
	"os"

	"github.com/czechbol/lumeon/app"
	"github.com/czechbol/lumeon/app/config/settings"
)

func main() {
	application := app.NewApp(settings.GetConfig())
	exitCode := app.RunAndManageApp(application)

	os.Exit(exitCode)
}
