package main

import (
	"github.com/inhere/skillc/internal/cli"
)

func main() {
	app := cli.NewApp()
	app.Run(nil)
}
