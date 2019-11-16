package graph

import (
	"fmt"
	"go.coder.com/nfy/internal/runner"
	"sort"
)

type TraverseFn func(r runner.Recipe) error

// TraverseOnce returns a TraverseFn that only calls fn on each target once.
func TraverseOnce(fn TraverseFn) TraverseFn {
	skip := make(map[string]bool)
	return func(r runner.Recipe) error {
		if skip[r.FullName()] {
			return nil
		}

		err := fn(r)
		skip[r.FullName()] = true
		return err
	}
}

// sortedSlice returns the map as a slice of keys in a deterministic order.
func (r RecipeIndex) sortedSlice() []Recipe {
	var rs []Recipe
	for _, v := range r {
		rs = append(rs, v)
	}
	sort.Slice(rs, func(i, j int) bool {
		return rs[i].Name < rs[j].Name
	})
	return rs
}

// Traverse traverses all recipes in the graph. It will only present recipes that it has presented all dependencies for.
func (r RecipeIndex) Traverse(fn TraverseFn) error {
	for _, r := range r {
		err := r.Traverse(fn)
		if err != nil {
			return err
		}
	}
	return nil
}

// Traverse calls fn for each recipe in it's graph until fn returns false or there are no more entries.
// Traverse is depth-first.
// It is eventually called against the Recipe itself.
func (r Recipe) Traverse(fn TraverseFn) error {
	for _, dep := range r.Dependencies {
		r, err := dep.Load()
		if err != nil {
			return fmt.Errorf("%s: %w", dep.Name(), err)
		}
		err = r.Traverse(fn)
		if err != nil {
			return fmt.Errorf("%s: %w", dep.Name(), err)
		}
	}
	return fn(r.Recipe)
}
