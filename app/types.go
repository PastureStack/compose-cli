package app

import (
	"github.com/PastureStack/compose-cli/project"
	"github.com/urfave/cli"
)

type ProjectFactory interface {
	Create(c *cli.Context) (*project.Project, error)
}
