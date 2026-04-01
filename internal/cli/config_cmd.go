package cli

import (
	"fmt"

	"github.com/gookit/gcli/v3"
	"github.com/gookit/gcli/v3/show"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/gookit/slog"
)


func buildConfigCommand() *gcli.Command {
	cmd := &gcli.Command{
		Name: "config",
		Desc: "Manage Skillc configuration",
		Aliases: []string{"cfg"},
	}

	cmd.Add(&gcli.Command{
		Name: "init",
		Desc: "Initialize config file",
		Func: func(c *gcli.Command, args []string) error {
			service := newConfigService()
			_, err := service.Init()
			if err != nil {
				return err
			}

			ccolor.Successln("ok. config file: %s", service.ConfigFile())
			return nil
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

			cfg.Sources = nil
			show.AList("config", cfg)
			return nil
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
			ccolor.Infof("%s=%s\n", args[0], value)
			return nil
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
			ccolor.Successln("ok")
			return nil
		},
	})

	return cmd
}
