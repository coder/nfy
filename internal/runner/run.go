package runner

import (
	"fmt"
	"go.coder.com/nfy/internal/parse"
	"io"
	"os/exec"
)

func defaultShell() string {
	return "sh"
}

type Output struct {
	Stderr io.Writer
	Stdout io.Writer
}

type Recipe struct {
	parse.Recipe
	Repo string
}

func (r Recipe) FullName() string {
	if r.Repo == "" {
		return r.Name
	}
	return r.Repo + ":" + r.Name
}

// FromParseRecipes converts parse.Recipes into Recipes.
func FromParseRecipes(rs []parse.Recipe, repo string) []Recipe {
	var rr []Recipe
	for _, r := range rs {
		rr = append(rr, Recipe{r, repo})
	}
	return rr
}

func (r Recipe) Check(out Output) error {
	cmd := exec.Command(defaultShell(), "-c", r.Recipe.Check)
	cmd.Stderr = out.Stderr
	cmd.Stdout = out.Stdout
	return cmd.Run()
}

func (r Recipe) CheckOnly() bool {
	return r.Recipe.Check != "" && r.Recipe.Install == ""
}

// DependencyOnly returns whether this recipe only proxies dependencies.
func (r Recipe) DependencyOnly() bool {
	return r.Recipe.Check == "" && r.Recipe.Install == ""
}

func (r Recipe) Install(out Output) error {
	if r.Recipe.Install == "" {
		return fmt.Errorf("no installer provided")
	}

	cmd := exec.Command(defaultShell(), "-c", r.Recipe.Install)
	cmd.Stderr = out.Stderr
	cmd.Stdout = out.Stdout
	return cmd.Run()
}
