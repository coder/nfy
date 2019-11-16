package graph

import (
	"fmt"
	"go.coder.com/nfy/internal/runner"
	"strings"
)

type RecipeIndex map[string]Recipe

// evalDepList evaluates a string dependency list and produces a set of virtual recipe loaders.
func evalDepList(parent string, remoteConfig RemoteConfig, deps []string, ind RecipeIndex) ([]RecipeLoader, error) {
	var ls []RecipeLoader
	for _, dep := range deps {
		if strings.Index(dep, ":") >= 0 {
			t, err := parseRemoteTarget(dep)
			if err != nil {
				return nil, fmt.Errorf("%q is misformatted: %w", dep, err)
			}
			ls = append(ls, &remoteLoader{
				raw:    dep,
				target: *t,
				parent: parent,
				config: remoteConfig,
			})
			continue
		}
		// TODO: support remote dependencies.
		ls = append(ls, &localLoader{
			parent: parent,
			name:   dep,
			ind:    ind,
		})
	}
	return ls, nil
}

// RemoteConfig configures how we pull dependencies.
type RemoteConfig struct {
	Path string
}

// Generate produces a graph for each recipe.
func Generate(recipes []runner.Recipe, rconfig RemoteConfig) (RecipeIndex, error) {
	localIndex := make(RecipeIndex, len(recipes))

	for _, recipe := range recipes {
		deps, err := evalDepList(recipe.Recipe.Name, rconfig, recipe.Recipe.Dependencies, localIndex)
		if err != nil {
			return nil, err
		}
		_, ok := localIndex[recipe.Name]
		if ok {
			return nil, fmt.Errorf("%s is declared multiple times", recipe.Name)
		}
		// Store recipe.
		localIndex[recipe.Name] = Recipe{
			Recipe:       recipe,
			Dependencies: deps,
		}
	}

	return localIndex, nil
}
