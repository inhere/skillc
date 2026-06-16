package cli

import (
	"fmt"
	"os"

	"github.com/gookit/cliui/show/table"
	"github.com/gookit/gcli/v3"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/skillc/internal/app/projectapp"
)

func buildProjectCommand() *gcli.Command {
	cmd := &gcli.Command{
		Name: "project",
		Desc: "Manage registered projects",
	}
	cmd.Add(buildProjectListCommand())
	cmd.Add(buildProjectAddCommand())
	cmd.Add(buildProjectRemoveCommand())
	cmd.Add(buildProjectImportLockCommand())
	return cmd
}

func buildProjectListCommand() *gcli.Command {
	return &gcli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Desc:    "List registered projects",
		Func: func(c *gcli.Command, args []string) error {
			items, err := newProjectService().List()
			if err != nil {
				return err
			}
			tb := table.New("Project List").SetHeads("ID", "Name", "Path")
			for _, item := range items {
				tb.AddRow(item.ID, item.Name, item.Path)
			}
			_, err = fmt.Fprint(os.Stdout, tb.Render())
			return err
		},
	}
}

func buildProjectAddCommand() *gcli.Command {
	var id string
	var name string
	var description string
	return &gcli.Command{
		Name: "add",
		Desc: "Register a project path",
		Config: func(c *gcli.Command) {
			c.AddArg("path", "project path", true)
			c.StrOpt(&id, "id", "", "", "project id")
			c.StrOpt(&name, "name", "", "", "project name")
			c.StrOpt(&description, "description", "", "", "project description")
		},
		Func: func(c *gcli.Command, args []string) error {
			item, err := newProjectService().Add(projectapp.AddReq{
				ID:          id,
				Name:        name,
				Path:        c.Arg("path").String(),
				Description: description,
			})
			if err != nil {
				return err
			}
			ccolor.Infof("project added: %s %s\n", item.ID, item.Path)
			return nil
		},
	}
}

func buildProjectRemoveCommand() *gcli.Command {
	return &gcli.Command{
		Name:    "remove",
		Aliases: []string{"rm"},
		Desc:    "Remove a registered project",
		Config: func(c *gcli.Command) {
			c.AddArg("id", "project id", true)
		},
		Func: func(c *gcli.Command, args []string) error {
			id := c.Arg("id").String()
			if err := newProjectService().Remove(id); err != nil {
				return err
			}
			ccolor.Infof("project removed: %s\n", id)
			return nil
		},
	}
}

func buildProjectImportLockCommand() *gcli.Command {
	return &gcli.Command{
		Name: "import-lock",
		Desc: "Register projects found in the lock file",
		Func: func(c *gcli.Command, args []string) error {
			result, err := newProjectService().ImportFromLock()
			if err != nil {
				return err
			}
			for _, item := range result.Added {
				ccolor.Infof("imported %s %s\n", item.ID, item.Path)
			}
			for _, item := range result.Skipped {
				ccolor.Warnf("skipped %s %s\n", item.Path, item.Reason)
			}
			return nil
		},
	}
}

func newProjectService() *projectapp.Service {
	cwd := getWorkdir()
	return projectapp.NewService(defaultConfigFile(cwd), cwd)
}
