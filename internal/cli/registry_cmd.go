package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/gookit/cliui/show/table"
	"github.com/gookit/gcli/v3"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/skillc/internal/app/installapp"
	"github.com/inhere/skillc/internal/app/registryapp"
	"github.com/inhere/skillc/internal/domain/registry"
	"github.com/inhere/skillc/internal/domain/skill"
	"github.com/inhere/skillc/internal/infra/agentfs"
)

func buildRegistryCommand() *gcli.Command {
	cmd := &gcli.Command{
		Name: "registry",
		Desc: "Manage Skillc registries",
	}
	cmd.Add(buildRegistryListCommand())
	cmd.Add(buildRegistryAddCommand())
	cmd.Add(buildRegistryRemoveCommand())
	cmd.Add(buildRegistrySyncCommand())
	cmd.Add(buildRegistrySearchCommand())
	cmd.Add(buildRegistryInfoCommand())
	cmd.Add(buildRegistryInstallCommand())
	cmd.Add(buildRegistryAddSourceCommand())
	return cmd
}

func buildRegistryListCommand() *gcli.Command {
	return &gcli.Command{
		Name:    "list",
		Desc:    "List configured registries",
		Aliases: []string{"ls"},
		Func: func(c *gcli.Command, args []string) error {
			items, err := newRegistryService().List()
			if err != nil {
				return err
			}
			tb := table.New("Registry List").SetHeads("ID", "Name", "Type", "Status", "Location")
			for _, item := range items {
				tb.AddRow(item.ID, item.Name, item.Type, item.Status, registryLocation(item))
			}
			_, err = fmt.Fprint(os.Stdout, tb.Render())
			return err
		},
	}
}

func buildRegistryAddCommand() *gcli.Command {
	var id string
	var name string
	return &gcli.Command{
		Name: "add",
		Desc: "Add a registry catalog",
		Config: func(c *gcli.Command) {
			c.StrOpt(&id, "id", "", "", "custom registry id")
			c.StrOpt(&name, "name", "", "", "custom registry name")
			c.AddArg("value", "registry catalog path or URL", true)
		},
		Func: func(c *gcli.Command, _ []string) error {
			item, err := newRegistryService().Add(registryapp.AddReq{
				ID:    id,
				Name:  name,
				Value: c.Arg("value").String(),
			})
			if err != nil {
				return err
			}
			ccolor.Infof("registry added: %s\n", item.ID)
			return nil
		},
	}
}

func buildRegistryRemoveCommand() *gcli.Command {
	return &gcli.Command{
		Name:    "remove",
		Desc:    "Remove registry by id",
		Aliases: []string{"rm"},
		Config: func(c *gcli.Command) {
			c.AddArg("id", "registry id", true)
		},
		Func: func(c *gcli.Command, _ []string) error {
			id := c.Arg("id").String()
			if err := newRegistryService().Remove(id); err != nil {
				return err
			}
			ccolor.Infof("registry removed: %s\n", id)
			return nil
		},
	}
}

func buildRegistrySyncCommand() *gcli.Command {
	var syncAll bool
	return &gcli.Command{
		Name: "sync",
		Desc: "Sync registry catalogs",
		Config: func(c *gcli.Command) {
			c.BoolOpt(&syncAll, "all", "a", false, "sync all registries")
			c.AddArg("id", "registry id")
		},
		Func: func(c *gcli.Command, _ []string) error {
			service := newRegistryService()
			if syncAll {
				if err := service.SyncAll(); err != nil {
					return err
				}
				ccolor.Infof("registry synced: all\n")
				return nil
			}
			id := c.Arg("id").String()
			if id == "" {
				return fmt.Errorf("registry id is required")
			}
			if err := service.Sync(id); err != nil {
				return err
			}
			ccolor.Infof("registry synced: %s\n", id)
			return nil
		},
	}
}

func buildRegistrySearchCommand() *gcli.Command {
	return &gcli.Command{
		Name: "search",
		Desc: "Search registry catalog entries",
		Config: func(c *gcli.Command) {
			c.AddArg("keyword", "search keyword")
		},
		Func: func(c *gcli.Command, _ []string) error {
			items, err := newRegistryService().Search(c.Arg("keyword").String())
			if err != nil {
				return err
			}
			tb := table.New("Registry Search").SetHeads("Registry", "ID", "Name", "Type", "Tags", "Location")
			for _, item := range items {
				tb.AddRow(item.RegistryID, item.ID, item.Name, item.Type, strings.Join(item.Tags, ","), entryLocation(item))
			}
			_, err = fmt.Fprint(os.Stdout, tb.Render())
			return err
		},
	}
}

func buildRegistryInfoCommand() *gcli.Command {
	return &gcli.Command{
		Name: "info",
		Desc: "Show registry entry details",
		Config: func(c *gcli.Command) {
			c.AddArg("entry", "entry id or registry/entry id", true)
		},
		Func: func(c *gcli.Command, _ []string) error {
			item, err := newRegistryService().Info(c.Arg("entry").String())
			if err != nil {
				return err
			}
			return printRegistryEntryInfo(item)
		},
	}
}

func buildRegistryAddSourceCommand() *gcli.Command {
	var id string
	var name string
	var syncNow bool
	return &gcli.Command{
		Name: "add-source",
		Desc: "Add a source from a registry entry",
		Config: func(c *gcli.Command) {
			c.StrOpt(&id, "id", "", "", "custom source id")
			c.StrOpt(&name, "name", "", "", "custom source name")
			c.BoolOpt(&syncNow, "sync", "", false, "sync source after adding")
			c.AddArg("entry", "entry id or registry/entry id", true)
		},
		Func: func(c *gcli.Command, _ []string) error {
			src, err := newRegistryService().AddSource(registryapp.AddSourceReq{
				EntryID: c.Arg("entry").String(),
				ID:      id,
				Name:    name,
				Sync:    syncNow,
			})
			if err != nil {
				return err
			}
			ccolor.Infof("source added: %s\n", src.ID)
			return nil
		},
	}
}

func buildRegistryInstallCommand() *gcli.Command {
	var opts ManageOptions
	return &gcli.Command{
		Name: "install",
		Desc: "Install a skill from a registry result",
		Config: func(c *gcli.Command) {
			opts.bindCommand(c)
			opts.bindInstallModeFlags(c)
			c.BoolOpt(&opts.Yes, "yes", "y", false, "skip confirmation prompt")
			c.AddArg("skill", "registry skill target, e.g. team/go-pro", true)
		},
		Func: func(c *gcli.Command, args []string) error {
			if err := applyRegistryInstallTrailingOptions(&opts, args); err != nil {
				return err
			}
			config, cwd, err := loadConfig()
			if err != nil {
				return err
			}
			if opts.UseCopy && opts.InstallMode != "" {
				return fmt.Errorf("--copy and --install-mode are mutually exclusive")
			}
			if opts.InstallMode != "" && !agentfs.IsValidMode(opts.InstallMode) {
				return fmt.Errorf("invalid --install-mode %q, allowed: symlink, junction, copy", opts.InstallMode)
			}

			item, err := newRegistryService().MaterializeSkill(c.Arg("skill").String())
			if err != nil {
				return err
			}
			ok, err := printInstallPlanAndConfirm([]skill.Skill{item}, opts)
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}

			installMode := opts.resolveInstallMode(config)
			fallbackNotifier := func(_ string, target string, err error) {
				ccolor.Warnf("symlink not supported, fallback to copy for %s: %v\n", target, err)
			}
			result, err := installapp.NewService(config.LockFile).
				WithInstallMode(installMode).
				WithSymlinkFallbackNotifier(fallbackNotifier).
				RunResolved(config, installapp.InstallReq{
					SkillID: item.SourceQualifiedName,
					Agent:   opts.Agent,
					Scope:   opts.Scope,
					WorkDir: cwd,
				}, []skill.Skill{item}, nil)
			if err != nil {
				return err
			}
			return printInstallResult(result)
		},
	}
}

func applyRegistryInstallTrailingOptions(opts *ManageOptions, args []string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		name, value, hasValue := strings.Cut(arg, "=")
		switch name {
		case "--yes", "-y":
			opts.Yes = true
		case "--copy":
			opts.UseCopy = true
		case "--agent", "-a":
			if !hasValue {
				i++
				if i >= len(args) {
					return fmt.Errorf("%s requires a value", name)
				}
				value = args[i]
			}
			opts.Agent = value
		case "--scope", "-s":
			if !hasValue {
				i++
				if i >= len(args) {
					return fmt.Errorf("%s requires a value", name)
				}
				value = args[i]
			}
			opts.Scope = value
		case "--install-mode":
			if !hasValue {
				i++
				if i >= len(args) {
					return fmt.Errorf("%s requires a value", name)
				}
				value = args[i]
			}
			opts.InstallMode = value
		default:
			return fmt.Errorf("unknown registry install argument: %s", arg)
		}
	}
	return nil
}

func registryLocation(item registry.Registry) string {
	if item.Type == registry.TypeHTTP {
		return item.URL
	}
	return item.Path
}

func entryLocation(item registry.Entry) string {
	if item.URL != "" {
		return item.URL
	}
	return item.Path
}

func printRegistryEntryInfo(item registry.Entry) error {
	tb := table.New("Registry Entry").SetHeads("Field", "Value")
	tb.AddRow("Registry", item.RegistryID)
	tb.AddRow("ID", item.ID)
	tb.AddRow("Name", item.Name)
	tb.AddRow("Description", item.Description)
	tb.AddRow("Type", item.Type)
	if item.URL != "" {
		tb.AddRow("URL", item.URL)
	}
	if item.Path != "" {
		tb.AddRow("Path", item.Path)
	}
	if item.Ref != "" {
		tb.AddRow("Ref", item.Ref)
	}
	if len(item.Tags) > 0 {
		tb.AddRow("Tags", strings.Join(item.Tags, ","))
	}
	_, err := fmt.Fprint(os.Stdout, tb.Render())
	return err
}
