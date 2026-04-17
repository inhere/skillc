package cli

import (
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

			ccolor.Successf("ok. config file: %s\n", service.ConfigFile())
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
		Config: func(c *gcli.Command) {
			c.AddArg("key", "config key", true)
		},
		Func: func(c *gcli.Command, args []string) error {
			key := c.Arg("key").String()
			service := newConfigService()
			value, err := service.Get(key)
			if err != nil {
				slog.Error(err)
				return err
			}
			ccolor.Infof("%s=%s\n", key, value)
			return nil
		},
	})

	cmd.Add(&gcli.Command{
		Name: "set",
		Desc: "Set config value by key",
		Config: func(c *gcli.Command) {
			c.AddArg("key", "config key", true)
			c.AddArg("value", "config value", true)
		},
		Func: func(c *gcli.Command, args []string) error {
			key := c.Arg("key").String()
			value := c.Arg("value").String()
			service := newConfigService()
			if err := service.Set(key, value); err != nil {
				slog.Error(err)
				return err
			}
			ccolor.Successln("ok")
			return nil
		},
	})

	return cmd
}
