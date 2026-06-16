package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/gookit/cliui/show/table"
	"github.com/gookit/gcli/v3"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/gookit/slog"
	"github.com/inhere/skillc/internal/app/sourceapp"
	"github.com/inhere/skillc/internal/domain/source"
)

func buildSourceCommand() *gcli.Command {
	cmd := &gcli.Command{
		Name:    "source",
		Desc:    "Manage Skillc sources",
		Aliases: []string{"src"},
	}

	cmd.Add(buildSourceAddCommand())

	cmd.Add(&gcli.Command{
		Name:    "list",
		Desc:    "List configured sources",
		Aliases: []string{"ls"},
		Func: func(c *gcli.Command, args []string) error {
			service := newSourceService()
			list, err := service.List()
			if err != nil {
				return err
			}

			tb := table.New("Source List").SetHeads("ID", "Type", "Status", "Path")
			for _, src := range list {
				tb.AddRow(src.ID, src.Type, src.Status, src.Path)
			}
			_, err = fmt.Fprint(os.Stdout, tb.Render())
			return err
		},
	})

	cmd.Add(buildSourceSyncCommand())
	cmd.Add(buildSourceInfoCommand())
	cmd.Add(buildSourceCollectionsCommand())
	cmd.Add(buildSourceSkillsCommand())

	cmd.Add(&gcli.Command{
		Name:    "status",
		Desc:    "Show source status",
		Aliases: []string{"st"},
		Func: func(c *gcli.Command, args []string) error {
			service := newSourceService()
			list, err := service.List()
			if err != nil {
				return err
			}

			tb := table.New("Source Status").SetHeads("ID", "Status", "Error")
			for _, src := range list {
				tb.AddRow(src.ID, src.Status, src.ErrorMessage)
			}
			_, err = fmt.Fprint(os.Stdout, tb.Render())
			return err
		},
	})

	cmd.Add(&gcli.Command{
		Name:    "remove",
		Desc:    "Remove source by id",
		Aliases: []string{"rm"},
		Config: func(c *gcli.Command) {
			c.AddArg("id", "source id", true)
		},
		Func: func(c *gcli.Command, _ []string) error {
			id := c.Arg("id").String()

			service := newSourceService()
			if err := service.Remove(id); err != nil {
				return err
			}
			ccolor.Infof("removed %s\n", id)
			return nil
		},
	})

	return cmd
}

func truncateDescription(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}

func buildSourceCollectionsCommand() *gcli.Command {
	return &gcli.Command{
		Name: "collections",
		Desc: "List collections grouped under sources",
		Config: func(c *gcli.Command) {
			c.AddArg("source", "source id or name")
		},
		Func: func(c *gcli.Command, _ []string) error {
			sourceID := c.Arg("source").String()
			items, err := newSearchService().ListSourceCollections(sourceID)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				ccolor.Warnln("no collections found")
				return nil
			}
			tb := table.New("Source Collections").SetHeads("Source", "Collection", "Skills")
			for _, item := range items {
				sourceName := item.SourceID
				if item.SourceName != "" {
					sourceName = item.SourceName
				}
				tb.AddRow(sourceName, item.Name, item.SkillCount)
			}
			_, err = fmt.Fprint(os.Stdout, tb.Render())
			return err
		},
	}
}

func buildSourceSkillsCommand() *gcli.Command {
	var collection string
	return &gcli.Command{
		Name: "skills",
		Desc: "List skills under a source",
		Config: func(c *gcli.Command) {
			c.AddArg("source", "source id or name", true)
			c.StrOpt(&collection, "collection", "c", "", "filter by source collection")
		},
		Func: func(c *gcli.Command, args []string) error {
			defer func() {
				collection = ""
			}()
			sourceID := c.Arg("source").String()
			collectionFilter := collection
			if collectionFilter == "" {
				var err error
				collectionFilter, err = collectionOptionFromArgs(rawArgsAfterFirst(c.RawArgs()))
				if err != nil {
					return err
				}
			}
			items, err := newSearchService().ListSourceSkills(sourceID, collectionFilter)
			if err != nil {
				return err
			}
			tb := table.New("Source Skills").SetHeads("Collection", "Skill", "Description")
			for _, item := range items {
				tb.AddRow(item.Collection, item.ID, truncateDescription(item.Description, 60))
			}
			_, err = fmt.Fprint(os.Stdout, tb.Render())
			return err
		},
	}
}

func rawArgsAfterFirst(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	return args[1:]
}

func collectionOptionFromArgs(args []string) (string, error) {
	for i, arg := range args {
		if arg == "--collection" || arg == "-c" {
			if i+1 < len(args) {
				return args[i+1], nil
			}
			return "", fmt.Errorf("%s option requires a value", arg)
		}
		if value, ok := strings.CutPrefix(arg, "--collection="); ok {
			return value, nil
		}
	}
	return "", nil
}

func buildSourceSyncCommand() *gcli.Command {
	var syncAll bool

	return &gcli.Command{
		Name:    "sync",
		Desc:    "Sync update sources",
		Aliases: []string{"up"},
		Config: func(c *gcli.Command) {
			c.BoolOpt(&syncAll, "all", "a", false, "sync all sources")
			c.AddArg("id", "source id")
		},
		Func: func(c *gcli.Command, _ []string) error {
			sourceID := c.Arg("id").String()
			if !syncAll && sourceID == "" {
				return fmt.Errorf("source id is required")
			}

			service := newSourceService()
			if syncAll {
				ccolor.Infof("sync ALL sources ...\n")
				err := service.SyncAll()
				if err != nil {
					return err
				}
				ccolor.Infof("synced ALL sources\n")
				return nil
			}

			// 解析部分匹配
			matched, err := service.MatchSources(sourceID)
			if err != nil {
				return err
			}
			switch len(matched) {
			case 0:
				return fmt.Errorf("source not found: %s", sourceID)
			case 1:
				exactID := matched[0].ID
				if exactID != sourceID {
					ccolor.Infof("matched: %s\n", exactID)
				}
				if err := service.Sync(exactID); err != nil {
					return err
				}
				// re-query status after sync
				list, err := service.List()
				if err != nil {
					return err
				}
				for _, src := range list {
					if src.ID == exactID {
						ccolor.Successf("Synced %s  status=%s\n", src.ID, src.Status)
						return nil
					}
				}
				ccolor.Successf("Synced %s\n", exactID)
				return nil
			default:
				ccolor.Warnf("multiple sources matched %q:\n", sourceID)
				for _, src := range matched {
					ccolor.Infof("  - %s (%s)\n", src.ID, src.Type)
				}
				confirmed, err := confirmPrompt(os.Stdin, os.Stdout, fmt.Sprintf("Sync all %d matched sources?", len(matched)))
				if err != nil {
					return err
				}
				if !confirmed {
					ccolor.Warnln("cancelled")
					return nil
				}
				for _, src := range matched {
					if err := service.Sync(src.ID); err != nil {
						ccolor.Errorf("failed to sync %s: %v\n", src.ID, err)
						continue
					}
					ccolor.Successf("Synced %s\n", src.ID)
				}
				return nil
			}
		},
	}
}

func buildSourceInfoCommand() *gcli.Command {
	return &gcli.Command{
		Name: "info",
		Desc: "Show source details",
		Config: func(c *gcli.Command) {
			c.AddArg("id", "source id or partial id", true)
		},
		Func: func(c *gcli.Command, _ []string) error {
			src, err := newSourceService().Info(c.Arg("id").String())
			if err != nil {
				return err
			}
			return printSourceInfo(src)
		},
	}
}

func buildSourceAddCommand() *gcli.Command {
	var id string
	var name string
	var ref string
	var syncNow bool

	cmd := &gcli.Command{
		Name: "add",
		Desc: "Add a git or local source",
		Config: func(c *gcli.Command) {
			c.StrOpt(&id, "id", "", "", "custom source id")
			c.StrOpt(&name, "name", "", "", "custom source name")
			c.StrOpt(&ref, "ref", "r", "", "git ref(branch/tag/hash)")
			c.BoolOpt(&syncNow, "sync", "", false, "sync source after adding")
			c.AddArg("value", "local source path or git source url", true)
		},
		Func: func(c *gcli.Command, _ []string) error {
			value := c.Arg("value").String()
			service := newLocalSourceService()
			src, err := service.Add(sourceapp.AddReq{
				Value: value,
				ID:    id,
				Name:  name,
				Ref:   ref,
			})
			if err != nil {
				slog.Error(err)
				return err
			}
			if err := printSourceAdded(src); err != nil {
				return err
			}

			if syncNow {
				if err := service.Sync(src.ID); err != nil {
					slog.Error(err)
					return err
				}
				return nil
			}
			ccolor.Infof("Next, please run: skillc source sync %s\n", src.ID)
			return nil
		},
	}
	cmd.Add(buildSourceAddLocalCommand())
	cmd.Add(buildSourceAddGitCommand())
	return cmd
}

func buildSourceAddLocalCommand() *gcli.Command {
	var syncNow bool
	var id string
	var name string
	return &gcli.Command{
		Name: "local",
		Desc: "Add a local source",
		Config: func(c *gcli.Command) {
			c.StrOpt(&id, "id", "", "", "custom source id")
			c.StrOpt(&name, "name", "", "", "custom source name")
			c.BoolOpt(&syncNow, "sync", "", false, "sync source after adding")
			c.AddArg("path", "local source path", true)
		},
		Func: func(c *gcli.Command, _ []string) error {
			pathArg := c.Arg("path").String()
			if pathArg == "" {
				return fmt.Errorf("local source path is required")
			}

			service := newLocalSourceService()
			src, err := service.AddLocalWithOptions(pathArg, source.SourceOptions{ID: id, Name: name})
			if err != nil {
				slog.Error(err)
				return err
			}
			if err := printSourceAdded(src); err != nil {
				return err
			}

			if syncNow {
				if err := service.Sync(src.ID); err != nil {
					slog.Error(err)
					return err
				}
				return nil
			}
			ccolor.Infof("Next, please run: skillc source sync %s\n", src.ID)
			return nil
		},
	}
}

func buildSourceAddGitCommand() *gcli.Command {
	var syncNow bool
	var id string
	var name string
	return &gcli.Command{
		Name: "git",
		Desc: "Add a git source",
		Config: func(c *gcli.Command) {
			c.StrOpt(&id, "id", "", "", "custom source id")
			c.StrOpt(&name, "name", "", "", "custom source name")
			c.AddArg("url", "git source url", true)
			c.AddArg("ref", "git ref(branch/tag/hash)", false)
			c.BoolOpt(&syncNow, "sync", "", false, "sync source after adding")
		},
		Func: func(c *gcli.Command, _ []string) error {
			urlArg := c.Arg("url").String()
			ref := c.Arg("ref").String()

			if urlArg == "" {
				return fmt.Errorf("git source url is required")
			}
			service := newLocalSourceService()
			src, err := service.AddGitWithOptions(urlArg, ref, source.SourceOptions{ID: id, Name: name})
			if err != nil {
				slog.Error(err)
				return err
			}

			if err := printSourceAdded(src); err != nil {
				return err
			}

			if syncNow {
				if err := service.Sync(src.ID); err != nil {
					slog.Error(err)
					return err
				}
				return nil
			}
			ccolor.Infof("Next, please run: skillc source sync %s\n", src.ID)
			return nil
		},
	}
}

func printSourceAdded(src source.Source) error {
	if _, err := fmt.Fprintf(os.Stdout, "%s added.\n - name=%s\n - type=%s\n", src.ID, src.Name, src.Type); err != nil {
		return err
	}
	if src.Type == source.TypeGit {
		_, err := fmt.Fprintf(os.Stdout, " - url=%s, ref=%s\n", src.URL, src.Ref)
		return err
	}
	_, err := fmt.Fprintf(os.Stdout, " - path=%s\n", src.Path)
	return err
}

func printSourceInfo(src source.Source) error {
	tb := table.New("Source Info").SetHeads("Field", "Value")
	tb.AddRow("ID", src.ID)
	tb.AddRow("Name", src.Name)
	tb.AddRow("Type", src.Type)
	tb.AddRow("Status", src.Status)
	if src.Path != "" {
		tb.AddRow("Path", src.Path)
	}
	if src.URL != "" {
		tb.AddRow("URL", src.URL)
	}
	if src.Ref != "" {
		tb.AddRow("Ref", src.Ref)
	}
	if src.ResolvedRef != "" {
		tb.AddRow("Resolved Ref", src.ResolvedRef)
	}
	_, err := fmt.Fprint(os.Stdout, tb.Render())
	return err
}
