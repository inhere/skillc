package main

import (
	"github.com/inhere/skillc/internal/cli"
)

// Build-time variables injected via -ldflags
var (
	Version   = "dev"
	GitHash   = "unknown"
	BuildTime = "unknown"
)

func main() {
	app := cli.NewApp(Version, GitHash, BuildTime)
	app.Run(nil)
}
