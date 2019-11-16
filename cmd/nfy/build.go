package main

import (
	"fmt"
	"github.com/spf13/pflag"
	"go.coder.com/cli"
	"go.coder.com/nfy/internal/builder"
	"go.coder.com/nfy/internal/clog"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type buildCmd struct {
	targets    []string
	base       string
	dockerFile bool
}

func (a buildCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "build",
		Usage: "[flags] -b <base> <image_name>",
		Desc:  "builds a Docker image",
	}
}

func (a *buildCmd) RegisterFlags(fl *pflag.FlagSet) {
	fl.StringSliceVarP(&a.targets, "targets", "t", nil, "only install specific targets")
	fl.StringVarP(&a.base, "base", "b", "", "base image for FROM clause")
	fl.BoolVarP(&a.dockerFile, "dockerfile", "f", false, "just print the Dockerfile")
}

func (a *buildCmd) Run(fl *pflag.FlagSet) {
	if a.base == "" {
		clog.Fatal("-b (base) required")
	}
	imageName := fl.Arg(0)
	if imageName == "" {
		clog.Error("image name must be provided")
		fl.Usage()
		os.Exit(1)
	}

	graphIndex := localGraph(a.targets)
	dfile, err := builder.Dockerfile(a.base, graphIndex)
	if err != nil {
		clog.Fatal("dockerfile build failed: %+v", err)
	}
	if a.dockerFile {
		fmt.Printf("%v\n",
			strings.TrimSpace(dfile),
		)
		return
	}

	// Prepare build context.
	dir, err := ioutil.TempDir("", "nfy")
	if err != nil {
		clog.Fatal("create tempdir failed: %v", err)
	}
	defer os.RemoveAll(dir)

	err = ioutil.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(dfile), 0640)
	if err != nil {
		clog.Fatal("write Dockerfile failed: %v", err)
	}

	// Execute Docker build.
	cmd := exec.Command("docker", "build", "-t", imageName, ".")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err != nil {
		clog.Fatal("docker build: %v", err)
	}
}
