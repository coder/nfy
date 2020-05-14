package graph

import (
	"cdr.dev/nfy/internal/runner"
	"context"
	"fmt"
	"sort"
	"strings"
)

type TraverseFn func(r runner.Installer) error

// TraverseOnce returns a TraverseFn that only calls fn on each target once.
func TraverseOnce(fn TraverseFn) TraverseFn {
	skip := make(map[string]bool)
	return func(r runner.Installer) error {
		if skip[r.FullName()] {
			return nil
		}

		err := fn(r)
		skip[r.FullName()] = true
		return err
	}
}

// sortedSlice returns the map as a slice of keys in a deterministic order.
func (ri RecipeIndex) sortedSlice() []Recipe {
	var rs []Recipe
	for _, v := range ri {
		rs = append(rs, v)
	}
	sort.Slice(rs, func(i, j int) bool {
		return len(rs[i].Installers) < len(rs[j].Installers)
	})
	return rs
}

// Traverse traverses all recipes in the graph. It will only present recipes that it has presented all dependencies for.
func (ri RecipeIndex) Traverse(ctx context.Context, fn TraverseFn) error {
	for name, r := range ri {
		err := r.Traverse(ctx, fn)
		if err != nil {
			return fmt.Errorf("load %s: %w", name, err)
		}
	}
	return nil
}

func (r Recipe) tryInstaller(ctx context.Context, ins Installer, fn TraverseFn) error {
	for _, dep := range ins.Dependencies {
		r, err := dep.Load(ctx)
		if err != nil {
			return fmt.Errorf("%s: %w", ins.Name, err)
		}
		err = r.Traverse(ctx, fn)
		if err != nil {
			return fmt.Errorf("%s: %w", ins.Name, err)
		}
	}
	return nil
}

// Traverse calls fn for each recipe in it's graph until fn returns false or there are no more entries.
// Traverse is depth-first.
// It is eventually called against the Recipe itself.
func (r Recipe) Traverse(ctx context.Context, fn TraverseFn) error {
	var (
		installer Installer
		errs      []error
		lastErr   error
	)

	// Try each installer and use the one that works.
	for _, installer = range r.Installers {
		err := r.tryInstaller(ctx, installer, fn)
		lastErr = err
		errs = append(errs, err)
		if err == nil {
			break
		}
	}

	if lastErr != nil {
		var errStr strings.Builder
		for _, err := range errs {
			fmt.Fprintf(&errStr, "\n\t%+v", err.Error())
		}
		return fmt.Errorf("%s", errStr.String())
	}

	return fn(installer.Runner)
}
