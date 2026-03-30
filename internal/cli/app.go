package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gookit/gcli/v3"
	"github.com/gookit/slog"
	"github.com/inhere/skillc/internal/app/configapp"
)

func NewApp() *gcli.App {
	app := gcli.NewApp()
	app.Name = "skillc"
	app.Desc = "Skill manager for multi-agent ecosystems"
	app.Version = "dev"
	app.Add(buildConfigCommand())
	return app
}

func buildConfigCommand() *gcli.Command {
	cmd := &gcli.Command{
		Name: "config",
		Desc: "Manage Skillc configuration",
	}

	cmd.Add(&gcli.Command{
		Name: "init",
		Desc: "Initialize config file",
		Func: func(c *gcli.Command, args []string) error {
			service := newConfigService()
			cfg, err := service.Init()
			if err != nil {
				slog.Error(err)
				return err
			}
			return WriteLine(os.Stdout, cfg.LockFile)
		},
	})

	cmd.Add(&gcli.Command{
		Name: "show",
		Desc: "Show current config",
		Func: func(c *gcli.Command, args []string) error {
			service := newConfigService()
			cfg, err := service.Show()
			if err != nil {
				slog.Error(err)
				return err
			}
			return WriteLine(os.Stdout, fmt.Sprintf("lock_file=%s", cfg.LockFile))
		},
	})

	cmd.Add(&gcli.Command{
		Name: "get",
		Desc: "Get config value by key",
		Func: func(c *gcli.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("config key is required")
			}
			service := newConfigService()
			value, err := service.Get(args[0])
			if err != nil {
				slog.Error(err)
				return err
			}
			return WriteLine(os.Stdout, value)
		},
	})

	cmd.Add(&gcli.Command{
		Name: "set",
		Desc: "Set config value by key",
		Func: func(c *gcli.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("config key and value are required")
			}
			service := newConfigService()
			if err := service.Set(args[0], args[1]); err != nil {
				slog.Error(err)
				return err
			}
			return WriteLine(os.Stdout, "ok")
		},
	})

	return cmd
}

func newConfigService() *configapp.Service {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return configapp.NewService(defaultConfigFile(cwd), cwd)
}

func defaultConfigFile(baseDir string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(baseDir, "skillc.yaml")
	}
	return filepath.Join(home, ".config", "skillc", "config.yaml")
}
