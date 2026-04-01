package cli

import (
	"fmt"
	"os"

	"github.com/gookit/gcli/v3"
	"github.com/gookit/gcli/v3/show/table"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/gookit/slog"
)

func buildCollectionCommand() *gcli.Command {
	cmd := &gcli.Command{
		Name: "collection",
		Desc: "Browse indexed collections",
		Aliases: []string{"coll"},
	}

	cmd.Add(&gcli.Command{
		Name: "list",
		Desc: "List indexed collections",
		Aliases: []string{"ls"},
		Func: func(c *gcli.Command, _ []string) error {
			service := newSearchService()
			items, err := service.ListCollections()
			if err != nil {
				slog.Error(err)
				return err
			}
			if len(items) == 0 {
				ccolor.Warnln("no collections found")
				return nil
			}

			tb := table.New("Collection List").SetHeads("Collection", "Skills", "Sources")
			for _, item := range items {
				tb.AddRow(item.Name, item.SkillCount, item.SourceCount)
			}
			_, err = fmt.Fprint(os.Stdout, tb.Render())
			return err
		},
	})

	cmd.Add(&gcli.Command{
		Name: "skills",
		Desc: "List skills in a collection",
		Config: func(c *gcli.Command) {
			c.AddArg("collection", "collection name", true)
		},
		Func: func(c *gcli.Command, _ []string) error {
			collection := c.Arg("collection").String()
			if collection == "" {
				return fmt.Errorf("collection name is required")
			}
			service := newSearchService()
			items, err := service.ListCollectionSkills(collection)
			if err != nil {
				slog.Error(err)
				return err
			}

			tb := table.New("Collection Skills").SetHeads("Name", "Description")
			for _, item := range items {
				tb.AddRow(item.Name, item.Description)
			}
			_, err = fmt.Fprint(os.Stdout, tb.Render())
			return err
		},
	})

	return cmd
}
