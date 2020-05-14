package graph

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"cdr.dev/nfy/internal/clog"
	"cdr.dev/nfy/internal/lockfile"
	"cdr.dev/nfy/internal/parse"
	"cdr.dev/nfy/internal/runner"
)

// RecipeLoader implements a recipe graph where some
// recipes may not be loaded yet.
type RecipeLoader interface {
	Load(ctx context.Context) (*Recipe, error)
	Name() string
}

type Installer struct {
	Name         string
	Runner       runner.Installer
	Dependencies []RecipeLoader
}

// Recipe represents a loaded recipe.
type Recipe struct {
	Installers []Installer
}

type localLoader struct {
	name string
	ind    RecipeIndex
}

func (l *localLoader) Name() string {
	return l.name
}

func (l *localLoader) Load(_ context.Context) (*Recipe, error) {
	r, ok := l.ind[l.name]
	if !ok {
		return nil, fmt.Errorf("%s: recipe not found locally", l.name)
	}
	return &r, nil
}

type remoteLoader struct {
	raw string

	target remoteTarget

	// parent is provided for error reporting.
	parent string

	config RemoteConfig
}

func (l *remoteLoader) Name() string {
	return l.raw
}

func (l *remoteLoader) lock() (func(), error) {
	lockPath := filepath.Join(l.config.Path, ".nfy.lockfile")
	err := lockfile.Lock(lockPath)
	var printWaitOnce sync.Once
	for err == lockfile.ErrLocked {
		printWaitOnce.Do(func() {
			clog.Info("waiting on %v...", lockPath)
		})
		err = lockfile.Lock(lockPath)
		time.Sleep(time.Millisecond * 10)
	}
	if err != nil {
		return nil, err
	}

	return func() {
		lockfile.Unlock(lockPath)
	}, nil
}

func (l *remoteLoader) Load(ctx context.Context) (*Recipe, error) {
	unlock, err := l.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()

	dir, err := ioutil.TempDir("", filepath.Join("nfy"))
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	if l.target.Tag == "" {
		l.target.Tag = "HEAD"
	}

	clog.Info("cloning %v", l.raw)
	cmd := exec.CommandContext(ctx, "git", "clone",
		"--depth", "1",
		"-b", l.target.Tag,
		"https://"+l.target.Repo, ".",
	)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to clone: %w\n%s", err, out)
	}
	clog.Success("cloned %v", l.raw)

	var recipes []parse.Recipe
	err = parse.Traverse(&recipes, filepath.Join(dir, "nfy.yml"))
	if err != nil {
		return nil, err
	}

	grp, err := Generate(runner.FromParseRecipes(recipes, l.raw), l.config)
	if err != nil {
		return nil, err
	}

	targetGraph, ok := grp[l.target.Target]
	if !ok {
		return nil, fmt.Errorf("repo does not have target %v", l.target.Target)
	}

	return &Recipe{
		Installers: targetGraph.Installers,
	}, nil
}
