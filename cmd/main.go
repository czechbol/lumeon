package main

import (
	"github.com/czechbol/lumeon/app"
	"github.com/czechbol/lumeon/app/config/settings"
)

func main() {
	application := app.NewApp(settings.GetConfig())
	app.RunAndManageApp(application)
}
