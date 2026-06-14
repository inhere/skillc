package cli

import (
	"fmt"
	"os"

	"github.com/gookit/cliui/show"
	"github.com/gookit/cliui/show/table"
	"github.com/gookit/gcli/v3"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/skillc/internal/app/profileapp"
	"github.com/inhere/skillc/internal/domain/profile"
)

func newProfileService() *profileapp.Service {
	cwd := getWorkdir()
	return profileapp.NewService(defaultConfigFile(cwd), cwd)
}

func buildProfileCommand() *gcli.Command {
	cmd := &gcli.Command{Name: "profile", Desc: "Manage Skillc profiles"}
	cmd.Add(buildProfileListCommand())
	cmd.Add(buildProfileShowCommand())
	cmd.Add(buildProfileCreateCommand())
	cmd.Add(buildProfileDiffCommand())
	cmd.Add(buildProfileApplyCommand())
	return cmd
}

func buildProfileListCommand() *gcli.Command {
	return &gcli.Command{
		Name:    "list",
		Desc:    "List profiles",
		Aliases: []string{"ls"},
		Func: func(c *gcli.Command, _ []string) error {
			items, err := newProfileService().List()
			if err != nil {
				return err
			}
			tb := table.New("Profiles").SetHeads("Name", "Targets", "Description")
			for _, item := range items {
				tb.AddRow(item.Name, len(item.Targets), item.Description)
			}
			_, err = fmt.Fprint(os.Stdout, tb.Render())
			return err
		},
	}
}

func buildProfileShowCommand() *gcli.Command {
	return &gcli.Command{
		Name: "show",
		Desc: "Show profile details",
		Config: func(c *gcli.Command) {
			c.AddArg("name", "profile name", true)
		},
		Func: func(c *gcli.Command, _ []string) error {
			item, err := newProfileService().Show(c.Arg("name").String())
			if err != nil {
				return err
			}
			show.AList("Profile", item)
			return nil
		},
	}
}

func buildProfileCreateCommand() *gcli.Command {
	var fromInstalled bool
	var fromCollection string
	var agentName string
	var scope string
	return &gcli.Command{
		Name: "create",
		Desc: "Create a profile",
		Config: func(c *gcli.Command) {
			c.AddArg("name", "profile name", true)
			c.BoolOpt(&fromInstalled, "from-installed", "", false, "create from installed skills")
			c.StrOpt(&fromCollection, "from-collection", "", "", "create from <source>/<collection>")
			c.StrOpt(&agentName, "agent", "a", "", "agent name")
			c.StrOpt(&scope, "scope", "s", "project", "scope")
		},
		Func: func(c *gcli.Command, _ []string) error {
			defer func() {
				fromInstalled = false
				fromCollection = ""
				agentName = ""
				scope = "project"
			}()
			name := c.Arg("name").String()
			fillProfileCreateOptions(c.RawArgs(), &fromInstalled, &fromCollection, &agentName, &scope)
			svc := newProfileService()
			switch {
			case fromCollection != "":
				_, err := svc.CreateFromCollection(name, fromCollection)
				if err != nil {
					return err
				}
			case fromInstalled:
				_, err := svc.CreateFromInstalled(name, profileapp.CreateFromInstalledReq{
					Agent:   agentName,
					Scope:   scope,
					WorkDir: getWorkdir(),
				})
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("use --from-installed or --from-collection")
			}
			ccolor.Successf("profile created: %s\n", name)
			return nil
		},
	}
}

func buildProfileDiffCommand() *gcli.Command {
	var agentName string
	var scope string
	return &gcli.Command{
		Name: "diff",
		Desc: "Preview profile apply plan",
		Config: func(c *gcli.Command) {
			c.AddArg("name", "profile name", true)
			c.StrOpt(&agentName, "agent", "a", "", "agent name")
			c.StrOpt(&scope, "scope", "s", "project", "scope")
		},
		Func: func(c *gcli.Command, _ []string) error {
			defer func() {
				agentName = ""
				scope = "project"
			}()
			fillProfileAgentScopeOptions(c.RawArgs(), &agentName, &scope)
			plan, err := newProfileService().PlanApply(c.Arg("name").String(), profileapp.ApplyReq{
				Agent:   agentName,
				Scope:   scope,
				WorkDir: getWorkdir(),
			})
			if err != nil {
				return err
			}
			return printProfilePlan(plan)
		},
	}
}

func buildProfileApplyCommand() *gcli.Command {
	var agentName string
	var scope string
	var dryRun bool
	var yes bool
	return &gcli.Command{
		Name: "apply",
		Desc: "Apply a profile",
		Config: func(c *gcli.Command) {
			c.AddArg("name", "profile name", true)
			c.StrOpt(&agentName, "agent", "a", "", "agent name")
			c.StrOpt(&scope, "scope", "s", "project", "scope")
			c.BoolOpt(&dryRun, "dry-run", "", false, "preview without installing")
			c.BoolOpt(&yes, "yes", "y", false, "skip confirmation")
		},
		Func: func(c *gcli.Command, _ []string) error {
			defer func() {
				agentName = ""
				scope = "project"
				dryRun = false
				yes = false
			}()
			name := c.Arg("name").String()
			fillProfileApplyOptions(c.RawArgs(), &agentName, &scope, &dryRun, &yes)
			svc := newProfileService()
			req := profileapp.ApplyReq{Agent: agentName, Scope: scope, WorkDir: getWorkdir()}
			plan, err := svc.PlanApply(name, req)
			if err != nil {
				return err
			}
			if err := printProfilePlan(plan); err != nil {
				return err
			}
			if dryRun {
				return nil
			}
			if !yes {
				confirmed, err := confirmPrompt(os.Stdin, os.Stdout, "Apply profile?")
				if err != nil {
					return err
				}
				if !confirmed {
					ccolor.Warnln("profile apply cancelled")
					return nil
				}
			}
			result, err := svc.Apply(name, req)
			if err != nil {
				return err
			}
			ccolor.Successf("profile applied: %s installed=%d\n", name, len(result.Installed))
			return nil
		},
	}
}

func printProfilePlan(plan profile.ApplyPlan) error {
	tb := table.New("Profile Plan").SetHeads("Action", "Source", "Skill", "Reason")
	for _, item := range plan.Items {
		tb.AddRow(item.Action, item.Target.Source, item.Target.Skill, item.Reason)
	}
	_, err := fmt.Fprint(os.Stdout, tb.Render())
	return err
}

func fillProfileCreateOptions(args []string, fromInstalled *bool, fromCollection *string, agentName *string, scope *string) {
	for i, arg := range rawArgsAfterFirst(args) {
		switch arg {
		case "--from-installed":
			*fromInstalled = true
		case "--from-collection":
			if i+2 <= len(rawArgsAfterFirst(args)) {
				*fromCollection = rawArgsAfterFirst(args)[i+1]
			}
		case "--agent", "-a":
			if i+2 <= len(rawArgsAfterFirst(args)) {
				*agentName = rawArgsAfterFirst(args)[i+1]
			}
		case "--scope", "-s":
			if i+2 <= len(rawArgsAfterFirst(args)) {
				*scope = rawArgsAfterFirst(args)[i+1]
			}
		}
	}
}

func fillProfileAgentScopeOptions(args []string, agentName *string, scope *string) {
	for i, arg := range rawArgsAfterFirst(args) {
		switch arg {
		case "--agent", "-a":
			if i+2 <= len(rawArgsAfterFirst(args)) {
				*agentName = rawArgsAfterFirst(args)[i+1]
			}
		case "--scope", "-s":
			if i+2 <= len(rawArgsAfterFirst(args)) {
				*scope = rawArgsAfterFirst(args)[i+1]
			}
		}
	}
}

func fillProfileApplyOptions(args []string, agentName *string, scope *string, dryRun *bool, yes *bool) {
	for _, arg := range rawArgsAfterFirst(args) {
		switch arg {
		case "--dry-run":
			*dryRun = true
		case "--yes", "-y":
			*yes = true
		}
	}
	fillProfileAgentScopeOptions(args, agentName, scope)
}
