package graph

import (
	"fmt"
	"strings"

	"cdr.dev/nfy/internal/runner"
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
func Generate(installers []runner.Installer, rconfig RemoteConfig) (RecipeIndex, error) {
	localIndex := make(RecipeIndex, len(installers))

	for _, installer := range installers {
		// We always append to the exist recipe's installers.
		r, _ := localIndex[installer.Recipe.Name]

		loaders, err := evalDepList(installer.FullName(), rconfig, installer.Dependencies, localIndex)
		if err != nil {
			return nil, err
		}
		r.Installers = append(r.Installers, Installer{
			Runner:       installer,
			Name:         installer.Name,
			Dependencies: loaders,
		})
		localIndex[installer.Recipe.Name] = r
	}

	return localIndex, nil
}
