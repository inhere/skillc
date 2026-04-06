package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gookit/gcli/v3"
	"github.com/gookit/gcli/v3/show"
	"github.com/gookit/gcli/v3/show/table"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/gookit/slog"
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/listapp"
	"github.com/inhere/skillc/internal/app/updateapp"
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

			tb := table.New("Search Result").SetHeads("Target", "Version", "Collection")
			for _, item := range items {
				target := item.QualifiedName
				if target == "" {
					target = item.ID
				}
				tb.AddRow(target, item.Version, item.Collection)
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
	Scope      string
	Agent      string
	Yes        bool
	Collection bool
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
			c.BoolOpt(&opts.Yes, "yes", "y", false, "skip confirmation prompt")
			c.BoolOpt(&opts.Collection, "collection", "c", false, "treat targets as collection selectors")
			c.AddArg("skill-id", "skill id. if empty, restore from lock file")
		},
		Func: func(c *gcli.Command, _ []string) error {
			config, cwd, err := loadConfig()
			if err != nil {
				slog.Error(err)
				return err
			}

			targetArg := c.Arg("skill-id").String()
			if targetArg == "" {
				result, err := installapp.NewService(config.LockFile).Run(config, installapp.InstallReq{
					Agent:   opts.Agent,
					Scope:   opts.Scope,
					WorkDir: cwd,
				}, nil)
				if err != nil {
					slog.Error(err)
					return err
				}
				for _, record := range result.Restored {
					if err := WriteLine(os.Stdout, fmt.Sprintf("%s %s %s", record.SkillID, record.Agent, record.Scope)); err != nil {
						return err
					}
				}
				return nil
			}

			targets := splitInstallTargets(targetArg)
			if len(targets) == 0 {
				ccolor.Warnln("invalid skill targets")
				return nil
			}

			searchResult, err := newSearchService().ResolveInstallTargets(targets, opts.Collection)
			if err != nil {
				slog.Error(err)
				return err
			}

			skillIDs := make([]string, 0, len(searchResult.Resolved))
			for _, item := range searchResult.Resolved {
				skillIDs = append(skillIDs, item.ID)
			}
			if _, err := fmt.Fprintf(os.Stdout, "Will install skills: %s\n", strings.Join(skillIDs, ", ")); err != nil {
				return err
			}
			if !opts.Yes {
				confirmed, err := confirmInstall(os.Stdin, os.Stdout)
				if err != nil {
					return err
				}
				if !confirmed {
					return WriteLine(os.Stdout, "install cancelled")
				}
			}

			result, err := installapp.NewService(config.LockFile).RunResolved(config, installapp.InstallReq{
				Agent:   opts.Agent,
				Scope:   opts.Scope,
				WorkDir: cwd,
			}, searchResult.Resolved, searchResult.Failed)
			if err != nil {
				slog.Error(err)
				return err
			}
			for _, record := range result.Installed {
				if err := WriteLine(os.Stdout, fmt.Sprintf("installed %s %s", record.SkillID, record.InstalledPath)); err != nil {
					return err
				}
			}
			for _, failed := range result.ResolveFailed {
				if err := WriteLine(os.Stdout, fmt.Sprintf("- resolve failed %s %s", failed.Target, failed.Reason)); err != nil {
					return err
				}
			}
			for _, failed := range result.InstallFailed {
				if err := WriteLine(os.Stdout, fmt.Sprintf("install failed %s %s", failed.SkillID, failed.Reason)); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func splitInstallTargets(value string) []string {
	parts := strings.Split(value, ",")
	targets := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		targets = append(targets, part)
	}
	return targets
}

func confirmInstall(in *os.File, out *os.File) (bool, error) {
	if _, err := fmt.Fprint(out, "Continue? [y/N] "); err != nil {
		return false, err
	}
	answer, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && err.Error() != "EOF" {
		return false, err
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}

func buildUpdateCommand() *gcli.Command {
	var opts ManageOptions
	return &gcli.Command{
		Name: "update",
		Desc: "Update installed skills",
		Config: func(c *gcli.Command) {
			opts.bindCommand(c)
			c.BoolOpt(&opts.Yes, "yes", "y", false, "skip confirmation prompt")
		},
		Func: func(c *gcli.Command, _ []string) error {
			_, cwd, err := loadConfig()
			if err != nil {
				slog.Error(err)
				return err
			}

			result, err := updateapp.NewService(defaultConfigFile(cwd), cwd).Run(updateapp.Req{
				Agent:   opts.Agent,
				Scope:   opts.Scope,
				WorkDir: cwd,
			})
			if err != nil {
				slog.Error(err)
				return err
			}
			for _, record := range result.Updated {
				if err := WriteLine(os.Stdout, fmt.Sprintf("updated %s %s", record.SkillID, record.InstalledPath)); err != nil {
					return err
				}
			}
			for _, skipped := range result.Skipped {
				if err := WriteLine(os.Stdout, fmt.Sprintf("skipped %s %s", skipped.SkillID, skipped.Reason)); err != nil {
					return err
				}
			}
			for _, failed := range result.Failed {
				if err := WriteLine(os.Stdout, fmt.Sprintf("update failed %s %s", failed.SkillID, failed.Reason)); err != nil {
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
