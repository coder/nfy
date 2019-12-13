package main

import (
	"context"
	"github.com/spf13/pflag"
	"go.coder.com/cli"
	"go.coder.com/nfy/internal/clog"
	"os"
	"os/signal"
)

type rootCmd struct {
	ctx context.Context
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
		&installCmd{ctx: c.ctx},
		&buildCmd{ctx: c.ctx},
	}
}

func (c *rootCmd) Run(f *pflag.FlagSet) {
	f.Usage()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sigs := make(chan os.Signal)
		signal.Notify(sigs, os.Interrupt)
		for s := range sigs {
			cancel()
			clog.Info("recieved %s, aborting", s)
		}
	}()
	cli.RunRoot(&rootCmd{ctx})
}
