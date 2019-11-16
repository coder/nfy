package main

import (
	"bytes"
	"github.com/fatih/color"
	"github.com/spf13/pflag"
	"go.coder.com/cli"
	"go.coder.com/nfy/internal/clog"
	"go.coder.com/nfy/internal/graph"
	"go.coder.com/nfy/internal/parse"
	"go.coder.com/nfy/internal/runner"
	"os"
	"path/filepath"
	"time"
)

type installCmd struct {
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
	// Replace the graphIndex with a filtered version if targets are specified.
	if targets != nil {
		graphIndex = func() graph.RecipeIndex {
			newIndex := make(graph.RecipeIndex)
			for _, v := range targets {
				recipe, ok := graphIndex[v]
				if !ok {
					clog.Fatal("%v not found", v)
				}
				newIndex[v] = recipe
			}
			return newIndex
		}()
	}

	return graphIndex
}

func (a installCmd) Run(fl *pflag.FlagSet) {
	var (
		totalCounter   int
		installCounter int
	)

	graphIndex := localGraph(a.targets)
	err := graphIndex.Traverse(
		graph.TraverseOnce(
			func(r runner.Recipe) error {
				totalCounter++
				if r.DependencyOnly() || r.BuildOnly {
					return nil
				}

				// TODO: sync.
				var outBuf bytes.Buffer
				out := runner.Output{
					Stderr: &outBuf,
					Stdout: &outBuf,
				}

				prefix := color.New(color.Bold).Sprint(r.Name)

				start := time.Now()
				if r.Recipe.Check != "" {
					err := r.Check(out)
					if a.showOutput {
						clog.Info("%s\t --- begin check output", prefix)
						outBuf.WriteTo(os.Stdout)
						clog.Info("%s\t --- end check output", prefix)
					}
					if err == nil {
						clog.Info("%s\tcheck succeeded (%v)", prefix, time.Since(start))
						return nil
					}
				} else {
					clog.Warn("%s\tno check, always reinstalling", prefix)
				}

				err := r.Install(out)
				if err != nil {
					outBuf.WriteTo(os.Stdout)
					clog.Fatal("%s\tinstall failed: %v (%v)", prefix, err, time.Since(start))
				}
				clog.Success("%s\tinstalled (%v)", prefix, time.Since(start))
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
		clog.Fatal("traverse failed: %w", err)
	}
	clog.Success("total: %v, installed: %v", totalCounter, installCounter)
}
