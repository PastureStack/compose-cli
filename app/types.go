package app

import (
	cli "github.com/PastureStack/compose-cli/internal/legacycli"
	"github.com/PastureStack/compose-cli/project"
)

type ProjectFactory interface {
	Create(c *cli.Context) (*project.Project, error)
}
