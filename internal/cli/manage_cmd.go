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
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
)

func buildSearchCommand() *gcli.Command {
	var agentFilter string
	var sourceTypeFilter string
	return &gcli.Command{
		Name: "search",
		Desc: "Search indexed skills",
		Aliases: []string{"find"},
		Config: func(c *gcli.Command) {
			c.AddArg("keyword", "search keyword")
			c.StrOpt(&agentFilter, "agent", "a", "", "filter by agent name")
			c.StrOpt(&sourceTypeFilter, "source-type", "t", "", "filter by source type (e.g. git, local)")
		},
		Func: func(c *gcli.Command, _ []string) error {
			keyword := c.Arg("keyword").String()
			service := newSearchService()
			items, err := service.Search(keyword, agentFilter, sourcepkg.Type(sourceTypeFilter))
			if err != nil {
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

type updateRunner interface {
	Run(updateapp.Req) (updateapp.Result, error)
}

var newUpdateService = func(configFile string, baseDir string) updateRunner {
	return updateapp.NewService(configFile, baseDir)
}

type ManageOptions struct {
	Scope      string
	Agent      string
	Yes        bool
	Collection bool
}

func (mo *ManageOptions) bindCommand(c *gcli.Command) {
	c.StrOpt(&mo.Scope, "scope", "s", string(agent.ScopeProject), "scope name")
	c.StrOpt(&mo.Agent, "agent", "a", agent.DefaultAgentName, "agent name or directory")
}

func buildInstallCommand() *gcli.Command {
	var opts ManageOptions
	var sourceArg string
	return &gcli.Command{
		Name:    "install",
		Desc:    "Install skills",
		Aliases: []string{"ins"},
		Config: func(c *gcli.Command) {
			opts.bindCommand(c)
			c.BoolOpt(&opts.Yes, "yes", "y", false, "skip confirmation prompt")
			c.BoolOpt(&opts.Collection, "collection", "c", false, "treat targets as collection selectors")
			c.StrOpt(&sourceArg, "source", "S", "", "git url or local path: add & sync source before installing")
			c.AddArg("skill", "skill id. if empty, restore from lock file")
		},
		Func: func(c *gcli.Command, _ []string) error {
			config, cwd, err := loadConfig()
			if err != nil {
				return err
			}

			// --source: 新增/复用 source 并同步
			if sourceArg != "" {
				srcSvc := newSourceService()
				src, isNew, err := srcSvc.EnsureSource(sourceArg, "")
				if err != nil {
					return fmt.Errorf("failed to ensure source: %w", err)
				}
				if isNew {
					ccolor.Infof("source added: %s (%s)\n", src.ID, src.Type)
				} else {
					ccolor.Infof("source exists: %s (%s)\n", src.ID, src.Type)
				}
				ccolor.Infof("syncing source %s ...\n", src.ID)
				if err := srcSvc.Sync(src.ID); err != nil {
					return fmt.Errorf("failed to sync source: %w", err)
				}
				ccolor.Infof("source synced: %s\n", src.ID)
				// reload config so index is fresh
				config, cwd, err = loadConfig()
				if err != nil {
					return err
				}
			}

			targetArg := c.Arg("skill").String()
			if targetArg == "" {
				result, err := installapp.NewService(config.LockFile).Run(config, installapp.InstallReq{
					Agent:   opts.Agent,
					Scope:   opts.Scope,
					WorkDir: cwd,
				}, nil)
				if err != nil {
					return err
				}
				for _, record := range result.Restored {
					ccolor.Infof("- restored %s  agent=%s scope=%s path=%s\n", record.SkillID, record.Agent, record.Scope, record.InstalledPath)
				}
				ccolor.Successf("restore complete: %d skill(s) restored\n", len(result.Restored))
				return nil
			}

			targets := splitInstallTargets(targetArg)
			if len(targets) == 0 {
				ccolor.Warnln("invalid skill targets")
				return nil
			}

			searchResult, err := newSearchService().ResolveInstallTargets(targets, opts.Collection)
			if err != nil {
				return err
			}

			skillIDs := make([]string, 0, len(searchResult.Resolved))
			for _, item := range searchResult.Resolved {
				skillIDs = append(skillIDs, item.ID)
			}
			ccolor.Infof("Will install skills: %s\n", strings.Join(skillIDs, ", "))
			ccolor.Infof(" - install to scope: %s, agent: %s\n", opts.Scope, opts.Agent)

			if !opts.Yes {
				confirmed, err := confirmInstall(os.Stdin, os.Stdout)
				if err != nil {
					return err
				}
				if !confirmed {
					ccolor.Warnln("install cancelled")
					return nil
				}
			}

			result, err := installapp.NewService(config.LockFile).RunResolved(config, installapp.InstallReq{
				Agent:   opts.Agent,
				Scope:   opts.Scope,
				WorkDir: cwd,
			}, searchResult.Resolved, searchResult.Failed)
			if err != nil {
				return err
			}
			for _, record := range result.Installed {
				ccolor.Infof("- installed %s %s\n", record.SkillID, record.InstalledPath)
			}
			for _, failed := range result.ResolveFailed {
				ccolor.Warnf("- resolve failed %s %s\n", failed.Target, failed.Reason)
			}
			for _, failed := range result.InstallFailed {
				ccolor.Errorf("- install failed %s %s\n", failed.SkillID, failed.Reason)
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
	return confirmPrompt(in, out, "Continue?")
}

func confirmPrompt(in *os.File, out *os.File, prompt string) (bool, error) {
	if _, err := fmt.Fprintf(out, "%s [y/N] ", prompt); err != nil {
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
	var target string
	return &gcli.Command{
		Name: "update",
		Desc: "Update installed skills",
		Aliases: []string{"up"},
		Config: func(c *gcli.Command) {
			opts.bindCommand(c)
			c.StrOpt(&target, "target", "t", "", "skill id to update (default: update all)")
		},
		Func: func(c *gcli.Command, _ []string) error {
			_, cwd, err := loadConfig()
			if err != nil {
				return err
			}

			result, err := newUpdateService(defaultConfigFile(cwd), cwd).Run(updateapp.Req{
				Target:  target,
				Agent:   opts.Agent,
				Scope:   opts.Scope,
				WorkDir: cwd,
			})
			if err != nil {
				slog.Error(err)
				return err
			}
			for _, record := range result.Updated {
				ccolor.Infof("updated %s %s\n", record.SkillID, record.InstalledPath)
			}
			for _, skipped := range result.Skipped {
				ccolor.Infof("skipped %s %s\n", skipped.SkillID, skipped.Reason)
			}
			for _, failed := range result.CleanupFailed {
				ccolor.Errorf("cleanup failed %s %s\n", failed.SkillID, failed.Reason)
			}
			for _, failed := range result.Failed {
				ccolor.Errorf("update failed %s %s\n", failed.SkillID, failed.Reason)
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
		Aliases: []string{"uni", "remove", "rm"},
		Config: func(c *gcli.Command) {
			opts.bindCommand(c)
			c.AddArg("skill", "skill id, allow multiple", true, true)
		},
		Func: func(c *gcli.Command, _ []string) error {
			skillIDs := c.Arg("skill").Strings()
			config, cwd, err := loadConfig()
			if err != nil {
				return err
			}
			scope, err := parseScope(opts.Scope)
			if err != nil {
				return err
			}

			svc := installapp.NewService(config.LockFile).WithRuntime(config, cwd)
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
			config, cwd, err := loadConfig()
			if err != nil {
				slog.Error(err)
				return err
			}

			scope, err := parseScope(opts.Scope)
			if err != nil {
				return err
			}

			svc := listapp.NewService(config.LockFile).WithRuntime(config, cwd)
			items, err := svc.List(opts.Agent, string(scope))
			if err != nil {
				slog.Error(err)
				return err
			}
			if len(items) == 0 {
				ccolor.Warnln("no skills found")
			} else {
				tb := table.New("List Skills").SetHeads("Skill ID", "Agent", "Scope", "Status")
				for _, item := range items {
					tb.AddRow(item.SkillID, item.Agent, item.Scope, item.Status)
				}
				_, err = fmt.Fprint(os.Stdout, tb.Render())
				if err != nil {
					return err
				}
			}

			unrecorded, err := svc.ScanUnrecorded(opts.Agent, scope)
			if err != nil {
				slog.Error(err)
				return err
			}
			if len(unrecorded) > 0 {
				ccolor.Infof("\nUnrecorded Skills:\n")
				for _, g := range unrecorded {
					ccolor.Printf("  <cyan>%-15s</> %s\n", g.AgentName, strings.Join(g.Skills, ", "))
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

			boolStr := func(v bool) string {
				if v {
					return "<green>yes</>"
				}
				return "<red>no</>"
			}

			tb := table.New("Doctor Check").SetHeads("Check", "Value")
			tb.AddRow("git_available", boolStr(result.GitAvailable))
			tb.AddRow("config_ok", boolStr(result.ConfigOK))
			tb.AddRow("lock_file", result.LockFile)
			tb.AddRow("repo_cache_dir", result.RepoCacheDir)
			_, err = fmt.Fprint(os.Stdout, tb.Render())
			return err
		},
	}
}
