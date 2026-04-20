package cli

import (
	"fmt"
	"os"

	"github.com/gookit/gcli/v3"
	"github.com/gookit/gcli/v3/show/table"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/gookit/slog"
)

func buildSourceCommand() *gcli.Command {
	cmd := &gcli.Command{
		Name:    "source",
		Desc:    "Manage Skillc sources",
		Aliases: []string{"src"},
	}

	add := &gcli.Command{
		Name: "add",
		Desc: "Add a git or local source",
	}
	add.Add(buildSourceAddLocalCommand())
	add.Add(buildSourceAddGitCommand())
	cmd.Add(add)

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
			if _, err := fmt.Fprintf(os.Stdout, "%s added.\n - path=%s\n", src.ID, src.Path); err != nil {
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
	return &gcli.Command{
		Name: "git",
		Desc: "Add a git source",
		Config: func(c *gcli.Command) {
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
			service := newSourceService()
			src, err := service.AddGit(urlArg, ref)
			if err != nil {
				slog.Error(err)
				return err
			}

			if _, err := fmt.Fprintf(os.Stdout, "%s added.\n - url=%s, ref=%s\n", src.ID, src.URL, src.Ref); err != nil {
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
