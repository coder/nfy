package builder

import (
	"context"
	"fmt"
	"cdr.dev/nfy/internal/graph"
	"cdr.dev/nfy/internal/runner"
	"strings"
)

// Dockerfile assembles a Dockerfile from a recipe graph.
func Dockerfile(ctx context.Context, base string, grp graph.RecipeIndex) (string, error) {
	var file strings.Builder
	fmt.Fprintf(&file, "FROM %s\n", base)
	err := grp.Traverse(ctx, graph.TraverseOnce(func(r runner.Installer) error {
		if r.Recipe.Comment != "" {
			fmt.Fprintf(&file, "# %s: %s\n", r.FullName(), r.Recipe.Comment)
		}
		if r.CheckOnly() {
			fmt.Fprintf(&file, "# Ensure the %q dependency exists:\n", r.FullName())
			fmt.Fprintf(&file, "RUN %s\n", r.Recipe.Check)
			return nil
		} else if ins := r.Recipe.Installers[0]; ins.Script != "" {
			fmt.Fprintf(&file, "RUN %s\n", ins.Script)
		}
		return nil
	}))
	if err != nil {
		return "", fmt.Errorf("traverse failed: %w", err)
	}
	return file.String(), nil
}
