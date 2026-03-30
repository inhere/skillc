package main

import (
	"os"

	"github.com/inhere/skillc/internal/cli"
)

func main() {
	app := cli.NewApp()
	app.Run(os.Args)
}
