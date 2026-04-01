package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gookit/gcli/v3"
	"github.com/gookit/goutil/x/ccolor"
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

func NewApp(version, gitHash, buildTime string) *gcli.App {
	app := gcli.NewApp()
	app.Name = "skillc"
	app.Desc = "Skill manager for multi-agent ecosystems"
	app.Version = fmt.Sprintf("%s (Git Hash: %s, Build Time: %s)", version, gitHash, buildTime)
	app.Add(buildConfigCommand())
	app.Add(buildSourceCommand())
	app.Add(buildSearchCommand())
	app.Add(buildShowCommand())
	app.Add(buildInstallCommand())
	app.Add(buildUninstallCommand())
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
	add.Add(buildSourceAddLocalCommand())
	add.Add(buildSourceAddGitCommand())
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
		Config: func(c *gcli.Command) {
			c.AddArg("id", "source id", true)
		},
		Func: func(c *gcli.Command, args []string) error {
			sourceID := c.Arg("id").String()
			for _, arg := range args {
				if sourceID == "" {
					sourceID = arg
				}
			}
			if sourceID == "" {
				return fmt.Errorf("source id is required")
			}
			service := newSourceService()
			if err := service.Sync(sourceID); err != nil {
				slog.Error(err)
				return err
			}
			list, err := service.List()
			if err != nil {
				slog.Error(err)
				return err
			}
			for _, src := range list {
				if src.ID == sourceID {
					return WriteLine(os.Stdout, fmt.Sprintf("synced %s %s", src.ID, src.Status))
				}
			}
			return WriteLine(os.Stdout, fmt.Sprintf("synced %s", sourceID))
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

func buildSourceAddLocalCommand() *gcli.Command {
	var syncNow bool
	return &gcli.Command{
		Name: "local",
		Desc: "Add a local source",
		Config: func(c *gcli.Command) {
			c.BoolOpt(&syncNow, "sync", "", false, "sync source after adding")
			c.AddArg("path", "local source path", true)
		},
		Func: func(c *gcli.Command, _ []string) error {
			pathArg := c.Arg("path").String()
			if pathArg == "" {
				return fmt.Errorf("local source path is required")
			}

			service := newSourceService()
			src, err := service.AddLocal(pathArg)
			if err != nil {
				slog.Error(err)
				return err
			}
			ccolor.Infof("%s path=%s added", src.ID, src.Path)

			if syncNow {
				if err := service.Sync(src.ID); err != nil {
					slog.Error(err)
					return err
				}
				return nil
			}
			ccolor.Infof("Next, please run > skillc source sync %s", src.ID)
			return nil
		},
	}
}

func buildSourceAddGitCommand() *gcli.Command {
	var syncNow bool
	return &gcli.Command{
		Name: "git",
		Desc: "Add a git source",
		Config: func(c *gcli.Command) {
			c.AddArg("url", "git source url", true)
			c.AddArg("ref", "git ref", false)
			c.BoolOpt(&syncNow, "sync", "", false, "sync source after adding")
		},
		Func: func(c *gcli.Command, args []string) error {
			urlArg := c.Arg("url").String()
			ref := c.Arg("ref").String()
			parsedSync := syncNow
			for _, arg := range args {
				if arg == "--sync" {
					parsedSync = true
					continue
				}
				if urlArg == "" {
					urlArg = arg
					continue
				}
				if ref == "" {
					ref = arg
				}
			}
			if urlArg == "" {
				return fmt.Errorf("git source url is required")
			}
			service := newSourceService()
			src, err := service.AddGit(urlArg, ref)
			if err != nil {
				slog.Error(err)
				return err
			}
			if err := WriteLine(os.Stdout, fmt.Sprintf("%s %s %s", src.ID, src.URL, src.Ref)); err != nil {
				return err
			}
			if parsedSync {
				if err := service.Sync(src.ID); err != nil {
					slog.Error(err)
					return err
				}
				return nil
			}
			return WriteLine(os.Stdout, fmt.Sprintf("next: skillc source sync %s", src.ID))
		},
	}
}

func buildSearchCommand() *gcli.Command {
	return &gcli.Command{
		Name: "search",
		Desc: "Search indexed skills",
		Config: func(c *gcli.Command) {
			c.AddArg("keyword", "search keyword", false)
		},
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
			if len(items) == 0 {
				return WriteLine(os.Stdout, "no skills found")
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
			for _, record := range result.Installed {
				if err := WriteLine(os.Stdout, fmt.Sprintf("%s %s", record.SkillID, record.InstalledPath)); err != nil {
					return err
				}
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

func buildUninstallCommand() *gcli.Command {
	return &gcli.Command{
		Name: "uninstall",
		Desc: "Uninstall skills",
		Config: func(c *gcli.Command) {
			c.AddArg("skill-id", "skill id", true)
			c.AddArg("agent", "agent name", true)
			c.AddArg("scope", "install scope", true)
		},
		Func: func(c *gcli.Command, args []string) error {
			if len(args) < 3 {
				return fmt.Errorf("skill id, agent, and scope are required")
			}
			config, _, err := loadConfig()
			if err != nil {
				slog.Error(err)
				return err
			}
			scope, err := parseScope(args[2])
			if err != nil {
				return err
			}
			if err := installapp.NewService(config.LockFile).Uninstall(args[0], args[1], scope); err != nil {
				slog.Error(err)
				return err
			}
			return WriteLine(os.Stdout, "ok")
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
	name := item.ID
	if item.QualifiedName != "" {
		name = item.QualifiedName
	}
	return fmt.Sprintf("%s %s %s", name, item.Name, item.Version)
}

func newConfigService() *configapp.Service {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return configapp.NewService(defaultConfigFile(cwd), cwd)
}

func newSearchService() *searchapp.Service {
	config, _, err := loadConfig()
	if err != nil {
		cwd, getwdErr := os.Getwd()
		if getwdErr != nil {
			cwd = "."
		}
		return searchapp.NewService(filepath.Join(cwd, "skillc-index.json"))
	}
	return searchapp.NewService(config.IndexFile)
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
