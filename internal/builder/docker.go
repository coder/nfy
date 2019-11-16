package builder

import (
	"fmt"
	"go.coder.com/nfy/internal/graph"
	"go.coder.com/nfy/internal/runner"
	"strings"
)

// Dockerfile assembles a Dockerfile from a recipe graph.
func Dockerfile(base string, grp graph.RecipeIndex) (string, error) {
	var file strings.Builder
	fmt.Fprintf(&file, "FROM %s\n", base)
	err := grp.Traverse(graph.TraverseOnce(func(r runner.Recipe) error {
		if r.Recipe.Comment != "" {
			fmt.Fprintf(&file, "# %s: %s\n", r.FullName(), r.Recipe.Comment)
		}
		if r.CheckOnly() {
			fmt.Fprintf(&file, "# Ensure the %q dependency exists:\n", r.FullName())
			fmt.Fprintf(&file, "RUN %s\n", r.Recipe.Check)
			return nil
		} else if r.Recipe.Install != "" {
			fmt.Fprintf(&file, "RUN %s\n", r.Recipe.Install)
		}
        return nil
	}))
	if err != nil {
        return "", fmt.Errorf("traverse failed: %w", err)
	}
	return file.String(), nil
}
