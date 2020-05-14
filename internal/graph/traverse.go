package graph

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"cdr.dev/nfy/internal/runner"
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
		err := r.Traverse(ctx, name, fn)
		if err != nil {
			return err
		}
	}
	return nil
}

type depError struct {
	ins    Installer
	parent string
	err    error
}

func (l *depError) Error() string {
	return l.err.Error()
}

func (r Recipe) tryInstaller(ctx context.Context, parent string, ins Installer, fn TraverseFn) *depError {
	for _, dep := range ins.Dependencies {
		r, err := dep.Load(ctx)
		if err != nil {
			return &depError{
				ins:    ins,
				parent: parent,
				err:    err,
			}
		}
		err = r.Traverse(ctx, parent, fn)
		if err != nil {
			return &depError{
				ins:    ins,
				parent: parent,
				err:    err,
			}
		}
	}
	return nil
}

type depErrors []*depError

func (d depErrors) Error() string {
	var s strings.Builder
	for _, err := range d {
		if ds, ok := err.err.(depErrors); ok {
			fmt.Fprintf(&s, ds.Error())
			continue
		}
		// Show dependency.
		fmt.Fprintf(&s, "\n\t%s -> %v", err.parent, strings.TrimSpace(err.Error()))
	}
	return s.String()
}

// Traverse calls fn for each recipe in it's graph until fn returns false or there are no more entries.
// Traverse is depth-first.
// It is eventually called against the Recipe itself.
func (r Recipe) Traverse(ctx context.Context, parent string, fn TraverseFn) error {
	var (
		installer Installer
		errs      depErrors
		lastErr   *depError
	)

	// Try each installer and use the one that works.
	for _, installer = range r.Installers {
		err := r.tryInstaller(ctx, parent, installer, fn)
		lastErr = err
		errs = append(errs, err)
		if err == nil {
			break
		}
	}

	if lastErr != nil {
		return errs
	}

	return fn(installer.Runner)
}
