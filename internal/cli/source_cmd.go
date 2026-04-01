package cli

import (
	"fmt"
	"os"

	"github.com/gookit/gcli/v3"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/gookit/slog"
)


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

	cmd.Add(buildSourceSyncCommand())

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

func buildSourceSyncCommand() *gcli.Command {
	var syncAll bool

	return &gcli.Command{
		Name: "sync",
		Desc: "Sync source by id",
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
				err := service.SyncAll()
				if err != nil {
					slog.Error(err)
					return err
				}
				ccolor.Successln("ok")
				return nil
			}

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
					ccolor.Infof("synced %s %s\n", src.ID, src.Status)
					return nil
				}
			}
			ccolor.Infof("synced %s\n", sourceID)
			return nil
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
			ccolor.Infof("%s added.\n - path=%s\n", src.ID, src.Path)

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

			ccolor.Infof("%s added.\n - url=%s, ref=%s\n", src.ID, src.URL, src.Ref)

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
