// Package legacycli keeps the command declarations stable while the runtime
// uses the maintained urfave/cli v3 parser. It intentionally exposes only the
// small v1 surface used by this repository.
package legacycli

import (
	"context"
	"strings"

	urfave "github.com/urfave/cli/v3"
)

// These variables remain for source compatibility. The v3 renderer is used
// because the historical templates reference fields removed from v3.
var (
	AppHelpTemplate     string
	CommandHelpTemplate string
)

type Flag interface {
	toV3(local bool) urfave.Flag
}

type StringSlice []string

type StringFlag struct {
	Name, Usage, EnvVar string
	Value               string
	Hidden              bool
}

type StringSliceFlag struct {
	Name, Usage, EnvVar string
	Value               *StringSlice
	Hidden              bool
}

type BoolFlag struct {
	Name, Usage, EnvVar string
	Hidden              bool
}

type BoolTFlag struct {
	Name, Usage, EnvVar string
	Hidden              bool
}

type IntFlag struct {
	Name, Usage, EnvVar string
	Value               int
	Hidden              bool
}

type Int64Flag struct {
	Name, Usage, EnvVar string
	Value               int64
	Hidden              bool
}

func splitNames(value string) (string, []string) {
	parts := strings.Split(value, ",")
	name := strings.TrimSpace(parts[0])
	aliases := make([]string, 0, len(parts)-1)
	for _, part := range parts[1:] {
		if alias := strings.TrimSpace(part); alias != "" {
			aliases = append(aliases, alias)
		}
	}
	return name, aliases
}

func envSources(value string) urfave.ValueSourceChain {
	if strings.TrimSpace(value) == "" {
		return urfave.ValueSourceChain{}
	}
	parts := strings.Split(value, ",")
	keys := make([]string, 0, len(parts))
	for _, part := range parts {
		if key := strings.TrimSpace(part); key != "" {
			keys = append(keys, key)
		}
	}
	return urfave.EnvVars(keys...)
}

func (f StringFlag) toV3(local bool) urfave.Flag {
	name, aliases := splitNames(f.Name)
	return &urfave.StringFlag{Name: name, Aliases: aliases, Usage: f.Usage, Sources: envSources(f.EnvVar), Value: f.Value, Hidden: f.Hidden, Local: local}
}

func (f StringSliceFlag) toV3(local bool) urfave.Flag {
	name, aliases := splitNames(f.Name)
	var value []string
	if f.Value != nil {
		value = append(value, (*f.Value)...)
	}
	return &urfave.StringSliceFlag{Name: name, Aliases: aliases, Usage: f.Usage, Sources: envSources(f.EnvVar), Value: value, Hidden: f.Hidden, Local: local}
}

func (f BoolFlag) toV3(local bool) urfave.Flag {
	name, aliases := splitNames(f.Name)
	return &urfave.BoolFlag{Name: name, Aliases: aliases, Usage: f.Usage, Sources: envSources(f.EnvVar), Hidden: f.Hidden, Local: local}
}

func (f BoolTFlag) toV3(local bool) urfave.Flag {
	name, aliases := splitNames(f.Name)
	return &urfave.BoolFlag{Name: name, Aliases: aliases, Usage: f.Usage, Sources: envSources(f.EnvVar), Value: true, Hidden: f.Hidden, Local: local}
}

func (f IntFlag) toV3(local bool) urfave.Flag {
	name, aliases := splitNames(f.Name)
	return &urfave.IntFlag{Name: name, Aliases: aliases, Usage: f.Usage, Sources: envSources(f.EnvVar), Value: f.Value, Hidden: f.Hidden, Local: local}
}

func (f Int64Flag) toV3(local bool) urfave.Flag {
	name, aliases := splitNames(f.Name)
	return &urfave.Int64Flag{Name: name, Aliases: aliases, Usage: f.Usage, Sources: envSources(f.EnvVar), Value: f.Value, Hidden: f.Hidden, Local: local}
}

type Context struct {
	context context.Context
	command *urfave.Command
}

func (c *Context) Args() []string                   { return c.command.Args().Slice() }
func (c *Context) NArg() int                        { return c.command.NArg() }
func (c *Context) String(name string) string        { return c.command.String(name) }
func (c *Context) StringSlice(name string) []string { return c.command.StringSlice(name) }
func (c *Context) Bool(name string) bool            { return c.command.Bool(name) }
func (c *Context) Int(name string) int              { return c.command.Int(name) }
func (c *Context) Int64(name string) int64          { return c.command.Int64(name) }
func (c *Context) Generic(name string) any          { return c.command.Value(name) }
func (c *Context) FlagNames() []string              { return c.command.FlagNames() }
func (c *Context) IsSet(name string) bool           { return c.command.IsSet(name) }

func (c *Context) GlobalString(name string) string { return c.command.Root().String(name) }
func (c *Context) GlobalStringSlice(name string) []string {
	return c.command.Root().StringSlice(name)
}
func (c *Context) GlobalBool(name string) bool { return c.command.Root().Bool(name) }
func (c *Context) GlobalInt(name string) int   { return c.command.Root().Int(name) }
func (c *Context) GlobalSet(name, value string) error {
	return c.command.Root().Set(name, value)
}

type Command struct {
	Name, ShortName, Usage, Description, ArgsUsage string
	Action                                         func(*Context) error
	Flags                                          []Flag
	Subcommands                                    []Command
	SkipFlagParsing                                bool
	Hidden                                         bool
}

func convertFlags(flags []Flag, local bool) []urfave.Flag {
	result := make([]urfave.Flag, 0, len(flags))
	for _, flag := range flags {
		result = append(result, flag.toV3(local))
	}
	return result
}

func (c Command) toV3() *urfave.Command {
	command := &urfave.Command{
		Name:            c.Name,
		Usage:           c.Usage,
		Description:     c.Description,
		ArgsUsage:       c.ArgsUsage,
		Flags:           convertFlags(c.Flags, true),
		SkipFlagParsing: c.SkipFlagParsing,
		Hidden:          c.Hidden,
	}
	if c.ShortName != "" {
		command.Aliases = []string{c.ShortName}
	}
	for _, child := range c.Subcommands {
		command.Commands = append(command.Commands, child.toV3())
	}
	if c.Action != nil {
		command.Action = func(ctx context.Context, cmd *urfave.Command) error {
			return c.Action(&Context{context: ctx, command: cmd})
		}
	}
	return command
}

type App struct {
	Name, Usage, Version, Author, Email string
	Before                              func(*Context) error
	Flags                               []Flag
	Commands                            []Command
}

func NewApp() *App { return &App{} }

func (a *App) Run(args []string) error {
	root := &urfave.Command{
		Name:    a.Name,
		Usage:   a.Usage,
		Version: a.Version,
		Flags:   convertFlags(a.Flags, false),
	}
	if a.Author != "" {
		root.Authors = []any{a.Author}
	}
	for _, command := range a.Commands {
		root.Commands = append(root.Commands, command.toV3())
	}
	if a.Before != nil {
		root.Before = func(ctx context.Context, cmd *urfave.Command) (context.Context, error) {
			return ctx, a.Before(&Context{context: ctx, command: cmd})
		}
	}
	return root.Run(context.Background(), args)
}

func ShowAppHelp(ctx *Context) error {
	return urfave.ShowRootCommandHelp(ctx.command.Root())
}

func ShowCommandHelp(ctx *Context, _ string) error {
	return urfave.ShowSubcommandHelp(ctx.command)
}

func NewExitError(message any, code int) error { return urfave.Exit(message, code) }
