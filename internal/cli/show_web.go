package cli

import (
"github.com/inhere/skillc/internal/app/webapp"
"github.com/inhere/skillc/internal/domain/skill"
)

// startSkillWebServer starts a local HTTP server to view skill files in a browser.
func startSkillWebServer(item skill.Skill, port int) error {
return webapp.NewServer().Serve(item, port)
}
