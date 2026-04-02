package cli

import (
	"fmt"
	"os"

	"github.com/gookit/gcli/v3"
	"github.com/gookit/gcli/v3/show"
	"github.com/gookit/gcli/v3/show/table"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/gookit/slog"
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/listapp"
	"github.com/inhere/skillc/internal/domain/agent"
)

func buildSearchCommand() *gcli.Command {
	return &gcli.Command{
		Name: "search",
		Desc: "Search indexed skills",
		Config: func(c *gcli.Command) {
			c.AddArg("keyword", "search keyword")
		},
		Func: func(c *gcli.Command, _ []string) error {
			keyword := c.Arg("keyword").String()
			service := newSearchService()
			items, err := service.Search(keyword, "", "")
			if err != nil {
				slog.Error(err)
				return err
			}
			if len(items) == 0 {
				ccolor.Warnln("no skills found")
				return nil
			}

			tb := table.New("Search Result").SetHeads("Name", "Version", "Collection")
			for _, item := range items {
				tb.AddRow(item.Name, item.Version, item.Collection)
			}
			_, err = fmt.Fprint(os.Stdout, tb.Render())
			return err
		},
	}
}

func buildShowCommand() *gcli.Command {
	return &gcli.Command{
		Name: "show",
		Desc: "Show indexed skill details",
		Config: func(c *gcli.Command) {
			c.AddArg("skill-id", "skill id", true)
		},
		Func: func(c *gcli.Command, _ []string) error {
			skillID := c.Arg("skill-id").String()
			service := newSearchService()
			item, err := service.Show(skillID)
			if err != nil {
				return err
			}

			show.AList("Skill Details", item)
			return nil
		},
	}
}

type ManageOptions struct {
	Scope string
	Agent string
}

func (mo *ManageOptions) bindCommand(c *gcli.Command) {
	c.StrOpt(&mo.Scope, "scope", "s", string(agent.ScopeProject), "scope name")
	c.StrOpt(&mo.Agent, "agent", "a", agent.DefaultAgentDir, "agent name or directory")
}

func buildInstallCommand() *gcli.Command {
	var opts ManageOptions
	return &gcli.Command{
		Name:    "install",
		Desc:    "Install skills",
		Aliases: []string{"ins"},
		Config: func(c *gcli.Command) {
			opts.bindCommand(c)
			c.AddArg("skill-id", "skill id. if empty, restore from lock file")
		},
		Func: func(c *gcli.Command, _ []string) error {
			config, cwd, err := loadConfig()
			if err != nil {
				slog.Error(err)
				return err
			}

			req := installapp.InstallReq{
				SkillID: c.Arg("skill-id").String(),
				Agent:    opts.Agent,
				Scope:    opts.Scope,
				WorkDir:  cwd,
			}
			result, err := installapp.NewService(config.LockFile).Run(config, req, newSearchService())
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
	var opts ManageOptions
	return &gcli.Command{
		Name:    "uninstall",
		Desc:    "Uninstall skills",
		Aliases: []string{"uni"},
		Config: func(c *gcli.Command) {
			opts.bindCommand(c)
			c.AddArg("skill-id", "skill id, allow multiple", true, true)
		},
		Func: func(c *gcli.Command, _ []string) error {
			skillIDs := c.Arg("skill-id").Strings()
			config, _, err := loadConfig()
			if err != nil {
				slog.Error(err)
				return err
			}
			scope, err := parseScope(opts.Scope)
			if err != nil {
				return err
			}

			svc := installapp.NewService(config.LockFile)
			if err := svc.UninstallMulti(skillIDs, opts.Agent, scope); err != nil {
				slog.Error(err)
				return err
			}
			ccolor.Successln("uninstalled")
			return nil
		},
	}
}

func buildListCommand() *gcli.Command {
	var opts ManageOptions
	return &gcli.Command{
		Name:    "list",
		Desc:    "List installed skills",
		Aliases: []string{"ls"},
		Config: func(c *gcli.Command) {
			opts.bindCommand(c)
		},
		Func: func(c *gcli.Command, _ []string) error {
			config, _, err := loadConfig()
			if err != nil {
				slog.Error(err)
				return err
			}

			scope, err := parseScope(opts.Scope)
			if err != nil {
				return err
			}

			items, err := listapp.NewService(config.LockFile).List(opts.Agent, string(scope))
			if err != nil {
				slog.Error(err)
				return err
			}
			if len(items) == 0 {
				ccolor.Warnln("no skills found")
				return nil
			}

			tb := table.New("List Skills").SetHeads("Skill ID", "Agent", "Scope", "Status")
			for _, item := range items {
				tb.AddRow(item.SkillID, item.Agent, item.Scope, item.Status)
			}
			_, err = fmt.Fprint(os.Stdout, tb.Render())
			return err
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
