package cli

import (
	"github.com/gookit/gcli/v3"
	"github.com/inhere/skillc/internal/app/webapp"
)

type webManagerServer interface {
	Serve(host string, port int) error
}

var newWebManagerServer = func(configFile string, baseDir string) webManagerServer {
	return webapp.NewManagerServer(configFile, baseDir)
}

func buildWebCommand() *gcli.Command {
	var host string
	var port int
	return &gcli.Command{
		Name: "web",
		Desc: "Start local web manager",
		Config: func(c *gcli.Command) {
			c.StrOpt(&host, "host", "", "127.0.0.1", "web server host")
			c.IntOpt(&port, "port", "p", 8080, "web server port")
		},
		Func: func(c *gcli.Command, _ []string) error {
			cwd := getWorkdir()
			return newWebManagerServer(defaultConfigFile(cwd), cwd).Serve(host, port)
		},
	}
}
