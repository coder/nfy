package main

import (
	"bytes"
	"context"
	"github.com/fatih/color"
	"github.com/spf13/pflag"
	"go.coder.com/cli"
	"cdr.dev/nfy/internal/clog"
	"cdr.dev/nfy/internal/graph"
	"cdr.dev/nfy/internal/parse"
	"cdr.dev/nfy/internal/runner"
	"os"
	"path/filepath"
	"time"
)

type installCmd struct {
	ctx context.Context

	showOutput bool
	targets    []string
}

func (a installCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "install",
		Usage: "",
		Desc:  "installs the nfy configuration to the local system",
	}
}

func (a *installCmd) RegisterFlags(fl *pflag.FlagSet) {
	fl.BoolVarP(&a.showOutput, "output", "o", false, "always show script output")
	fl.StringSliceVarP(&a.targets, "targets", "t", nil, "only install specific targets")
}

func localGraph(targets []string) graph.RecipeIndex {
	var err error
	path := os.Getenv("NFY_PATH")
	if path == "" {
		path, err = os.Getwd()
		if err != nil {
			// WTF.
			panic(err)
		}
	}
	clog.Debug("using path: %v", path)

	var parsedRecipes []parse.Recipe
	root := filepath.Join(path, "nfy.yml")
	err = parse.Traverse(&parsedRecipes, root)
	if err != nil {
		clog.Fatal("%v", err)
	}

	graphIndex, err := graph.Generate(runner.FromParseRecipes(parsedRecipes, ""), graph.RemoteConfig{Path: path})
	if err != nil {
		clog.Fatal("%+v", err)
	}

	// If no specify targets are specified, evaluate all.
	if targets == nil {
		return graphIndex
	}

	// Replace the graphIndex with a filtered version if targets are specified.
	return func() graph.RecipeIndex {
		newIndex := make(graph.RecipeIndex)
		for _, v := range targets {
			recipe, ok := graphIndex[v]
			if !ok {
				graphIndex.Dump()
				clog.Fatal("no recipe %q not found", v)
			}
			newIndex[v] = recipe
		}
		return newIndex
	}()
}

func (a installCmd) Run(fl *pflag.FlagSet) {
	var (
		totalCounter   int
		installCounter int
	)

	graphIndex := localGraph(a.targets)
	err := graphIndex.Traverse(
		a.ctx,
		graph.TraverseOnce(
			func(installer runner.Installer) error {
				totalCounter++
				if installer.DependencyOnly() || installer.Recipe.BuildOnly {
					return nil
				}

				// TODO: sync.
				var outBuf bytes.Buffer
				out := runner.Output{
					Stderr: &outBuf,
					Stdout: &outBuf,
				}

				prefix := color.New(color.Bold).Sprint(installer.Name)

				start := time.Now()
				if installer.Recipe.Check != "" {
					err := installer.Check(out)
					if a.showOutput {
						clog.Info("%s\t --- begin check output", prefix)
						outBuf.WriteTo(os.Stdout)
						clog.Info("%s\t --- end check output", prefix)
					}
					if err == nil {
						clog.Info("%s\tcheck succeeded (%v)", prefix, time.Since(start))
						return nil
					}
				}

				err := installer.Install(out)
				if err != nil {
					outBuf.WriteTo(os.Stdout)
					clog.Fatal("%s\tinstall failed: %v (%v)", prefix, err, time.Since(start))
				}
				var noCheckMessage string
				if installer.Recipe.Check == "" {
					noCheckMessage = "no check, "
				}
				clog.Success("%s\t%sinstalled (%v)", prefix, noCheckMessage, time.Since(start))
				if a.showOutput {
					clog.Info("%s\t --- begin install output")
					outBuf.WriteTo(os.Stdout)
					clog.Info("%s\t --- end install output")
				}

				installCounter++
				return nil
			},
		),
	)
	if err != nil {
		clog.Fatal("%+v", err)
	}
	clog.Success("total: %v, installed: %v", totalCounter, installCounter)
}
