package main

import (
	"github.com/spf13/pflag"
	"go.coder.com/cli"
)

type rootCmd struct {
}

func (c *rootCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "nfy",
		Usage: "<subcommand> [flags] <args>",
		Desc: `nfy is a local configuration management tool.
Read up at nfy.dev`,
	}
}

func (c *rootCmd) Subcommands() []cli.Command {
	return []cli.Command{
		&installCmd{},
		&buildCmd{},
	}
}

func (c *rootCmd) Run(f *pflag.FlagSet) {
	f.Usage()
}

func main() {
	cli.RunRoot(&rootCmd{})
}
