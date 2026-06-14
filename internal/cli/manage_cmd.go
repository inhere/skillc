package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gookit/cliui/show"
	"github.com/gookit/cliui/show/table"
	"github.com/gookit/gcli/v3"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/gookit/slog"
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/listapp"
	"github.com/inhere/skillc/internal/app/statusapp"
	"github.com/inhere/skillc/internal/app/updateapp"
	"github.com/inhere/skillc/internal/app/webapp"
	"github.com/inhere/skillc/internal/domain/agent"
	cfg "github.com/inhere/skillc/internal/domain/config"
	"github.com/inhere/skillc/internal/domain/skill"
	sourcepkg "github.com/inhere/skillc/internal/domain/source"
	"github.com/inhere/skillc/internal/infra/agentfs"
	"github.com/inhere/skillc/internal/infra/termselect"
)

func buildSearchCommand() *gcli.Command {
	var agentFilter string
	var sourceTypeFilter string
	return &gcli.Command{
		Name:    "search",
		Desc:    "Search indexed skills",
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
	var webMode bool
	var port int
	return &gcli.Command{
		Name: "show",
		Desc: "Show indexed skill details",
		Config: func(c *gcli.Command) {
			c.AddArg("skill", "skill id or name", true)
			c.BoolOpt(&webMode, "web", "w", false, "start a local web server to browse skill files")
			c.IntOpt(&port, "port", "p", 8080, "web server port (used with --web)")
		},
		Func: func(c *gcli.Command, _ []string) error {
			skillID := c.Arg("skill").String()
			service := newSearchService()
			item, err := service.Show(skillID)
			if err != nil {
				return err
			}

			if webMode {
				return webapp.NewServer().Serve(item, port)
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
	Scope       string
	Agent       string
	Yes         bool
	UseCopy     bool
	InstallMode string
}

func (mo *ManageOptions) bindCommand(c *gcli.Command) {
	c.StrOpt(&mo.Scope, "scope", "s", string(agent.ScopeProject), "scope name")
	c.StrOpt(&mo.Agent, "agent", "a", agent.DefaultAgentName, "agent name or directory")
}

// bindInstallModeFlags 注册安装模式选项，未设置时使用 config.install_mode 或平台默认模式。
func (mo *ManageOptions) bindInstallModeFlags(c *gcli.Command) {
	c.BoolOpt(&mo.UseCopy, "copy", "", false, "install skills by copying files instead of creating a symlink")
	c.StrOpt(&mo.InstallMode, "install-mode", "", "", "install mode: symlink, junction, copy")
}

// resolveInstallMode 根据 CLI 标志和 config 决定本次安装使用的 Mode
func (mo *ManageOptions) resolveInstallMode(config cfg.Config) agentfs.Mode {
	if mo.UseCopy {
		return agentfs.ModeCopy
	}
	if mo.InstallMode != "" {
		return agentfs.NormalizeMode(mo.InstallMode)
	}
	return agentfs.NormalizeMode(config.InstallMode)
}

func buildInstallCommand() *gcli.Command {
	var opts ManageOptions
	var sourceArg string
	var interactive bool
	return &gcli.Command{
		Name:    "install",
		Desc:    "Install skills",
		Aliases: []string{"ins"},
		Config: func(c *gcli.Command) {
			opts.bindCommand(c)
			opts.bindInstallModeFlags(c)
			c.BoolOpt(&opts.Yes, "yes", "y", false, "skip confirmation prompt")
			c.BoolOpt(&interactive, "interactive", "i", false, "interactively select skills to install")
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

			if opts.UseCopy && opts.InstallMode != "" {
				return fmt.Errorf("--copy and --install-mode are mutually exclusive")
			}
			if opts.InstallMode != "" && !agentfs.IsValidMode(opts.InstallMode) {
				return fmt.Errorf("invalid --install-mode %q, allowed: symlink, junction, copy", opts.InstallMode)
			}
			installMode := opts.resolveInstallMode(config)
			fallbackNotifier := func(_ string, target string, err error) {
				ccolor.Warnf("symlink not supported, fallback to copy for %s: %v\n", target, err)
			}

			targetArg := c.Arg("skill").String()
			if interactive {
				items, err := newSearchService().Search(targetArg, opts.Agent, "")
				if err != nil {
					return err
				}
				if len(items) == 0 {
					ccolor.Warnln("no skills found")
					return nil
				}
				selected, err := newMultiSelector().SelectMulti(context.Background(), termselect.Options{
					Title:        "Install skills",
					Items:        skillSelectItems(items),
					FilterPrompt: "filter skills",
				})
				if err != nil {
					return err
				}
				resolved := selectedSkills(items, selected)
				if len(resolved) == 0 {
					ccolor.Warnln("no skills selected")
					return nil
				}
				ok, err := printInstallPlanAndConfirm(resolved, opts)
				if err != nil {
					return err
				}
				if !ok {
					return nil
				}

				result, err := installapp.NewService(config.LockFile).
					WithInstallMode(installMode).
					WithSymlinkFallbackNotifier(fallbackNotifier).
					RunResolved(config, installapp.InstallReq{
						Agent:   opts.Agent,
						Scope:   opts.Scope,
						WorkDir: cwd,
					}, resolved, nil)
				if err != nil {
					return err
				}
				return printInstallResult(result)
			}

			if targetArg == "" {
				svc := installapp.NewService(config.LockFile).
					WithInstallMode(installMode).
					WithSymlinkFallbackNotifier(fallbackNotifier)
				result, err := svc.Run(config, installapp.InstallReq{
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

			searchResult, err := newSearchService().ResolveInstallTargets(targets, false)
			if err != nil {
				return err
			}

			ok, err := printInstallPlanAndConfirm(searchResult.Resolved, opts)
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}

			result, err := installapp.NewService(config.LockFile).
				WithInstallMode(installMode).
				WithSymlinkFallbackNotifier(fallbackNotifier).
				RunResolved(config, installapp.InstallReq{
					Agent:   opts.Agent,
					Scope:   opts.Scope,
					WorkDir: cwd,
				}, searchResult.Resolved, searchResult.Failed)
			if err != nil {
				return err
			}
			return printInstallResult(result)
		},
	}
}

func printInstallPlan(items []skill.Skill, opts ManageOptions) {
	skillIDs := make([]string, 0, len(items))
	for _, item := range items {
		skillIDs = append(skillIDs, item.ID)
	}
	ccolor.Infof("Will install skills: %s\n", strings.Join(skillIDs, ", "))
	ccolor.Infof(" - install to scope: %s, agent: %s\n", opts.Scope, opts.Agent)
}

func printInstallPlanAndConfirm(items []skill.Skill, opts ManageOptions) (bool, error) {
	printInstallPlan(items, opts)
	if opts.Yes {
		return true, nil
	}
	confirmed, err := confirmInstall(os.Stdin, os.Stdout)
	if err != nil {
		return false, err
	}
	if !confirmed {
		ccolor.Warnln("install cancelled")
		return false, nil
	}
	return true, nil
}

func printInstallResult(result installapp.CommandResult) error {
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
	var checkOnly bool
	var interactive bool
	return &gcli.Command{
		Name:    "update",
		Desc:    "Update installed skills",
		Aliases: []string{"up"},
		Config: func(c *gcli.Command) {
			opts.bindCommand(c)
			c.StrOpt(&target, "target", "t", "", "skill id to update (default: update all)")
			c.BoolOpt(&checkOnly, "check", "", false, "check update candidates without installing")
			c.BoolOpt(&interactive, "interactive", "i", false, "interactively select update candidates")
			c.AddArg("skill", "skill id to update (same as --target)")
		},
		Func: func(c *gcli.Command, _ []string) error {
			_, cwd, err := loadConfig()
			if err != nil {
				return err
			}
			if target == "" {
				target = c.Arg("skill").String()
			}
			if err := validateUpdateMode(checkOnly, interactive); err != nil {
				return err
			}
			if checkOnly {
				result, err := statusapp.NewService(defaultConfigFile(cwd), cwd).Run(statusapp.Req{
					Agent:   opts.Agent,
					Scope:   opts.Scope,
					WorkDir: cwd,
					Sync:    true,
				})
				if err != nil {
					return err
				}
				return printUpdateCheckResult(result, target)
			}
			if interactive {
				result, err := statusapp.NewService(defaultConfigFile(cwd), cwd).Run(statusapp.Req{
					Agent:   opts.Agent,
					Scope:   opts.Scope,
					WorkDir: cwd,
					Sync:    true,
				})
				if err != nil {
					return err
				}
				candidates := filterUpdateCandidates(result.Items, target)
				selectItems := updateSelectItems(candidates)
				if len(selectItems) == 0 {
					ccolor.Successln("no update candidates")
					return nil
				}
				selected, err := newMultiSelector().SelectMulti(context.Background(), termselect.Options{
					Title:        "Update skills",
					Items:        selectItems,
					FilterPrompt: "filter update candidates",
				})
				if err != nil {
					return err
				}
				targets := selectedUpdateTargets(candidates, selected)
				if len(targets) == 0 {
					ccolor.Warnln("no update targets selected")
					return nil
				}
				for _, selectedTarget := range targets {
					result, err := newUpdateService(defaultConfigFile(cwd), cwd).Run(updateapp.Req{
						Target:  selectedTarget,
						Agent:   opts.Agent,
						Scope:   opts.Scope,
						WorkDir: cwd,
					})
					if err != nil {
						slog.Error(err)
						return err
					}
					printUpdateResult(result)
				}
				return nil
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
			printUpdateResult(result)
			return nil
		},
	}
}

func validateUpdateMode(checkOnly bool, interactive bool) error {
	if checkOnly && interactive {
		return fmt.Errorf("--check and --interactive are mutually exclusive")
	}
	return nil
}

func printUpdateCheckResult(result statusapp.Result, target string) error {
	items := filterUpdateCheckItems(result.Items, target)
	if len(items) == 0 {
		ccolor.Successln("no update candidates")
		return nil
	}
	tb := table.New("Update Check").SetHeads("Status", "Skill", "Source", "Agent", "Current", "Latest", "Reason")
	for _, item := range items {
		tb.AddRow(item.Status, item.SkillID, item.SourceID, item.Agent, item.CurrentVersion, item.LatestVersion, item.Reason)
	}
	_, err := fmt.Fprint(os.Stdout, tb.Render())
	return err
}

func filterUpdateCandidates(items []statusapp.Item, target string) []statusapp.Item {
	out := make([]statusapp.Item, 0, len(items))
	for _, item := range items {
		if item.Status != statusapp.StatusOutdated && item.Status != statusapp.StatusMissing {
			continue
		}
		if target != "" && !matchesStatusTarget(item, target) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func filterUpdateCheckItems(items []statusapp.Item, target string) []statusapp.Item {
	out := make([]statusapp.Item, 0, len(items))
	for _, item := range items {
		if target != "" && !matchesStatusTarget(item, target) {
			continue
		}
		if item.Status == statusapp.StatusInstalled {
			continue
		}
		out = append(out, item)
	}
	return out
}

func matchesStatusTarget(item statusapp.Item, target string) bool {
	return item.SkillID == target || item.QualifiedName == target || item.SourceQualifiedName == target
}

func printUpdateResult(result updateapp.Result) {
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

func buildStatusCommand() *gcli.Command {
	var opts ManageOptions
	var profileName string
	return &gcli.Command{
		Name: "status",
		Desc: "Show skill status",
		Config: func(c *gcli.Command) {
			opts.bindCommand(c)
			c.StrOpt(&profileName, "profile", "p", "", "profile name")
		},
		Func: func(c *gcli.Command, _ []string) error {
			config, cwd, err := loadConfig()
			if err != nil {
				return err
			}
			scope, err := parseScope(opts.Scope)
			if err != nil {
				return err
			}
			result, err := statusapp.NewService(defaultConfigFile(cwd), cwd).Run(statusapp.Req{
				Agent:   opts.Agent,
				Scope:   string(scope),
				Profile: profileName,
				WorkDir: cwd,
			})
			if err != nil {
				return err
			}
			return printStatusResult(result, config)
		},
	}
}

func printStatusResult(result statusapp.Result, _ cfg.Config) error {
	if len(result.Items) == 0 {
		ccolor.Warnln("no skills found")
		return nil
	}
	tb := table.New("Skill Status").SetHeads("Status", "Skill", "Source", "Agent", "Scope", "Profile", "Current", "Latest", "Reason")
	for _, item := range result.Items {
		tb.AddRow(item.Status, item.SkillID, item.SourceID, item.Agent, item.Scope, item.Profile, item.CurrentVersion, item.LatestVersion, item.Reason)
	}
	if _, err := fmt.Fprint(os.Stdout, tb.Render()); err != nil {
		return err
	}
	ccolor.Infof("summary installed=%d missing=%d outdated=%d orphan=%d unmanaged=%d source_error=%d\n",
		result.Summary.Installed,
		result.Summary.Missing,
		result.Summary.Outdated,
		result.Summary.Orphan,
		result.Summary.Unmanaged,
		result.Summary.SourceError,
	)
	return nil
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
