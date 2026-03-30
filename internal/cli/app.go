package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gookit/gcli/v3"
	"github.com/gookit/slog"
	"github.com/inhere/skillc/internal/app/configapp"
	"github.com/inhere/skillc/internal/app/doctorapp"
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/listapp"
	"github.com/inhere/skillc/internal/app/searchapp"
	"github.com/inhere/skillc/internal/app/sourceapp"
	"github.com/inhere/skillc/internal/domain/agent"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/skill"
)

func NewApp() *gcli.App {
	app := gcli.NewApp()
	app.Name = "skillc"
	app.Desc = "Skill manager for multi-agent ecosystems"
	app.Version = "dev"
	app.Add(buildConfigCommand())
	app.Add(buildSourceCommand())
	app.Add(buildSearchCommand())
	app.Add(buildShowCommand())
	app.Add(buildInstallCommand())
	app.Add(buildListCommand())
	app.Add(buildDoctorCommand())
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

func buildSourceCommand() *gcli.Command {
	cmd := &gcli.Command{
		Name: "source",
		Desc: "Manage Skillc sources",
	}

	add := &gcli.Command{
		Name: "add",
		Desc: "Add a source",
	}
	add.Add(&gcli.Command{
		Name: "local",
		Desc: "Add a local source",
		Func: func(c *gcli.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("local source path is required")
			}
			service := newSourceService()
			src, err := service.AddLocal(args[0])
			if err != nil {
				slog.Error(err)
				return err
			}
			return WriteLine(os.Stdout, fmt.Sprintf("%s %s", src.ID, src.Path))
		},
	})
	add.Add(&gcli.Command{
		Name: "git",
		Desc: "Add a git source",
		Func: func(c *gcli.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("git source url is required")
			}
			ref := ""
			if len(args) > 1 {
				ref = args[1]
			}
			service := newSourceService()
			src, err := service.AddGit(args[0], ref)
			if err != nil {
				slog.Error(err)
				return err
			}
			return WriteLine(os.Stdout, fmt.Sprintf("%s %s %s", src.ID, src.URL, src.Ref))
		},
	})
	cmd.Add(add)

	cmd.Add(&gcli.Command{
		Name: "list",
		Desc: "List sources",
		Func: func(c *gcli.Command, args []string) error {
			service := newSourceService()
			list, err := service.List()
			if err != nil {
				slog.Error(err)
				return err
			}
			for _, src := range list {
				if err := WriteLine(os.Stdout, fmt.Sprintf("%s %s %s %s", src.ID, src.Type, src.Status, src.Path)); err != nil {
					return err
				}
			}
			return nil
		},
	})

	cmd.Add(&gcli.Command{
		Name: "sync",
		Desc: "Sync source by id",
		Func: func(c *gcli.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("source id is required")
			}
			service := newSourceService()
			if err := service.Sync(args[0]); err != nil {
				slog.Error(err)
				return err
			}
			return WriteLine(os.Stdout, "ok")
		},
	})

	cmd.Add(&gcli.Command{
		Name: "status",
		Desc: "Show source status",
		Func: func(c *gcli.Command, args []string) error {
			service := newSourceService()
			list, err := service.List()
			if err != nil {
				slog.Error(err)
				return err
			}
			for _, src := range list {
				if err := WriteLine(os.Stdout, fmt.Sprintf("%s %s %s", src.ID, src.Status, src.ErrorMessage)); err != nil {
					return err
				}
			}
			return nil
		},
	})

	cmd.Add(&gcli.Command{
		Name: "remove",
		Desc: "Remove source by id",
		Func: func(c *gcli.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("source id is required")
			}
			service := newSourceService()
			if err := service.Remove(args[0]); err != nil {
				slog.Error(err)
				return err
			}
			return WriteLine(os.Stdout, "ok")
		},
	})

	return cmd
}

func buildSearchCommand() *gcli.Command {
	return &gcli.Command{
		Name: "search",
		Desc: "Search indexed skills",
		Func: func(c *gcli.Command, args []string) error {
			keyword := ""
			if len(args) > 0 {
				keyword = args[0]
			}
			service := newSearchService()
			items, err := service.Search(keyword, "", "")
			if err != nil {
				slog.Error(err)
				return err
			}
			for _, item := range items {
				if err := WriteLine(os.Stdout, formatSkillLine(item)); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func buildShowCommand() *gcli.Command {
	return &gcli.Command{
		Name: "show",
		Desc: "Show indexed skill details",
		Func: func(c *gcli.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("skill id is required")
			}
			service := newSearchService()
			item, err := service.Show(args[0])
			if err != nil {
				slog.Error(err)
				return err
			}
			return WriteLine(os.Stdout, formatSkillLine(item))
		},
	}
}

func buildInstallCommand() *gcli.Command {
	return &gcli.Command{
		Name: "install",
		Desc: "Install skills",
		Func: func(c *gcli.Command, args []string) error {
			config, cwd, err := loadConfig()
			if err != nil {
				slog.Error(err)
				return err
			}

			result, err := installapp.NewService(config.LockFile).Run(config, cwd, args, newSearchService())
			if err != nil {
				slog.Error(err)
				return err
			}
			if result.Installed != nil {
				return WriteLine(os.Stdout, fmt.Sprintf("%s %s", result.Installed.SkillID, result.Installed.InstalledPath))
			}
			for _, record := range result.Restored {
				if err := WriteLine(os.Stdout, fmt.Sprintf("%s %s %s", record.SkillID, record.Agent, record.Scope)); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func buildListCommand() *gcli.Command {
	return &gcli.Command{
		Name: "list",
		Desc: "List installed skills",
		Func: func(c *gcli.Command, args []string) error {
			config, _, err := loadConfig()
			if err != nil {
				slog.Error(err)
				return err
			}

			agentName := ""
			scope := ""
			if len(args) > 0 {
				agentName = args[0]
			}
			if len(args) > 1 {
				scope = args[1]
			}
			if len(args) > 2 {
				return fmt.Errorf("too many arguments")
			}
			if scope != "" {
				if _, err := parseScope(scope); err != nil {
					return err
				}
			}

			items, err := listapp.NewService(config.LockFile).List(agentName, scope)
			if err != nil {
				slog.Error(err)
				return err
			}
			for _, item := range items {
				if err := WriteLine(os.Stdout, fmt.Sprintf("%s %s %s %s", item.SkillID, item.Agent, item.Scope, item.Status)); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func buildDoctorCommand() *gcli.Command {
	return &gcli.Command{
		Name: "doctor",
		Desc: "Check environment health",
		Func: func(c *gcli.Command, args []string) error {
			service := newDoctorService()
			result, err := service.Check()
			if err != nil {
				slog.Error(err)
				return err
			}
			lines := []string{
				fmt.Sprintf("git_available=%t", result.GitAvailable),
				fmt.Sprintf("config_ok=%t", result.ConfigOK),
				fmt.Sprintf("lock_file=%s", result.LockFile),
				fmt.Sprintf("repo_cache_dir=%s", result.RepoCacheDir),
			}
			for _, line := range lines {
				if err := WriteLine(os.Stdout, line); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func formatSkillLine(item skill.Skill) string {
	return fmt.Sprintf("%s %s %s", item.ID, item.Name, item.Version)
}

func newConfigService() *configapp.Service {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return configapp.NewService(defaultConfigFile(cwd), cwd)
}

func newSearchService() *searchapp.Service {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return searchapp.NewService(filepath.Join(cwd, "skillc-index.json"))
}

func newSourceService() *sourceapp.Service {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return sourceapp.NewService(defaultConfigFile(cwd), cwd)
}

func newDoctorService() *doctorapp.Service {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return doctorapp.NewService(defaultConfigFile(cwd), cwd)
}

func loadConfig() (cfg.Config, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return cfg.Config{}, "", err
	}
	config, err := configapp.NewService(defaultConfigFile(cwd), cwd).Show()
	if err != nil {
		return cfg.Config{}, "", err
	}
	return config, cwd, nil
}

func parseScope(value string) (agent.Scope, error) {
	scope := agent.Scope(value)
	switch scope {
	case agent.ScopeUser, agent.ScopeProject:
		return scope, nil
	default:
		return "", fmt.Errorf("unsupported scope: %s", value)
	}
}

func defaultConfigFile(baseDir string) string {
	localPath := filepath.Join(baseDir, "skillc.yaml")
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return localPath
	}
	return filepath.Join(home, ".config", "skillc", "config.yaml")
}
