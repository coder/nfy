package runner

import (
	"fmt"
	"io"
	"os/exec"

	"cdr.dev/nfy/internal/parse"
)

func defaultShell() string {
	return "sh"
}

type Output struct {
	Stderr io.Writer
	Stdout io.Writer
}

type Installer struct {
	Recipe parse.Recipe
	Repo string
	parse.Installer
}

func (i Installer) FullName() string {
	if i.Repo == "" {
		return i.Recipe.Name
	}
	return i.Repo + ":" + i.Recipe.Name
}

// FromParseRecipes converts parse.Recipes into Recipes.
// It generates a new runner.Recipe for each Installer.
func FromParseRecipes(rs []parse.Recipe, repo string) []Installer {
	var is []Installer
	for _, recipe := range rs {
		for _, installer := range recipe.Installers {
			is = append(is, Installer{recipe, repo, installer})
		}
		if len(recipe.Installers) == 0 && recipe.Check != "" {
			// Add a check-only installer if none provided.
			is = append(is, Installer{
				Recipe:    recipe,
				Repo:      repo,
				Installer: parse.Installer{},
			})
		}
	}
	return is
}

func (i Installer) Check(out Output) error {
	cmd := exec.Command(defaultShell(), "-c", i.Recipe.Check)
	cmd.Stderr = out.Stderr
	cmd.Stdout = out.Stdout
	return cmd.Run()
}

func (i Installer) CheckOnly() bool {
	return i.Recipe.Check != "" && len(i.Recipe.Installers) == 0
}

// DependencyOnly returns whether this recipe only proxies dependencies.
func (i Installer) DependencyOnly() bool {
	return i.Recipe.Check == "" && len(i.Recipe.Installers) == 0
}

func (i Installer) Install(out Output) error {
	if i.Script == "" {
		return fmt.Errorf("no installer provided")
	}

	cmd := exec.Command(defaultShell(), "-c", i.Script)
	cmd.Stderr = out.Stderr
	cmd.Stdout = out.Stdout
	return cmd.Run()
}
