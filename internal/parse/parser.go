package parse

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type Installer struct {
	// Name identifies the installer when multiple are provided.
	Name         string
	Script       string
	Dependencies []string
}

func (i Installer) FQDN(r Recipe) string {
	if i.Name == "" {
		return r.Name
	}
	return r.Name + " [" + i.Name + "]"
}

type Recipe struct {
	Name       string
	Check      string
	BuildOnly  bool
	Comment    string
	Installers []Installer
}

type Result struct {
	Imports []string
	Recipes []Recipe
}

func expectError(field, typ string) error {
	return fmt.Errorf("expected %q to be %q", field, typ)
}

func parseDependencies(v interface{}) ([]string, error) {
	deps, ok := v.([]interface{})
	if !ok {
		return nil, expectError("deps", "array")
	}
	var ds []string
	for _, dep := range deps {
		ds = append(ds, fmt.Sprint(dep))
	}
	return ds, nil
}

func parseRecipe(key string, val yaml.MapSlice) (Recipe, error) {
	// overloaded is true
	var overloaded bool

	var r Recipe
	var ok bool
	for _, it := range val {
		key := fmt.Sprint(it.Key)
		switch {
		case strings.HasPrefix(key, "install"):
			var installer Installer
			const splitChar = "_"
			if tok := strings.Split(key, splitChar); len(tok) == 2 {
				overloaded = true
				installer.Name = tok[1]
			}

			// Simple path
			if !overloaded {
				installer.Script, ok = it.Value.(string)
				if !ok {
					return r, expectError("install", "string")
				}
				r.Installers = append(r.Installers, installer)
				continue
			}

			// Parse out overloaded dependencies.
			args, ok := it.Value.(yaml.MapSlice)
			if !ok {
				return r, expectError(key+".deps", "MapSlice")
			}
			for _, it := range args {
				switch it.Key {
				case "deps":
					var err error
					installer.Dependencies, err = parseDependencies(it.Value)
					if err != nil {
						return r, err
					}
				case "script":
					installer.Script, ok = it.Value.(string)
					if !ok {
						return r, expectError("script", "string")
					}
				default:
					return r, fmt.Errorf("overloaded target has unexpected key %q", it.Key)
				}
			}
			r.Installers = append(r.Installers, installer)

		case key == "check":
			r.Check, ok = it.Value.(string)
			if !ok {
				return r, expectError("check", "string")
			}
		case key == "build_only":
			r.BuildOnly, ok = it.Value.(bool)
			if !ok {
				return r, expectError("build_only", "bool")
			}
		case key == "comment":
			r.Comment, ok = it.Value.(string)
			if !ok {
				return r, expectError("comment", "string")
			}
		case key == "deps":
			switch {
			case len(r.Installers) > 1:
				return r, fmt.Errorf("if the target is overloaded, deps must be provided per installer")
			default:
				var err error
				// Add an inert install. This is a depedency proxy.
				if len(r.Installers) == 0 {
					r.Installers = append(r.Installers, Installer{})
				}
				r.Installers[0].Dependencies, err = parseDependencies(it.Value)
				if err != nil {
					return r, err
				}
			}
		default:
			return r, fmt.Errorf("unexpected directive %q", it.Key)
		}
	}
	r.Name = key
	return r, nil
}

// Parse parses a recipe from a source file.
func Parse(r io.Reader) (*Result, error) {
	var items yaml.MapSlice
	err := yaml.NewDecoder(r).Decode(&items)
	if err != nil {
		if err == io.EOF {
			return &Result{}, nil
		}
		return nil, err
	}

	var rs Result
	for _, item := range items {
		if item.Key == nil {
			return nil, fmt.Errorf("nil key? value: %+v", item.Value)
		}
		key, ok := item.Key.(string)
		if !ok {
			return nil, fmt.Errorf("%v is of type %T; we expect a string", item.Key, item.Key)
		}

		switch key {
		case "import":
			arr, ok := item.Value.([]interface{})
			if !ok {
				return nil, fmt.Errorf("%v is of type %T, we expected a []interface{}", item.Value, item.Value)
			}
			for _, imp := range arr {
				rs.Imports = append(rs.Imports, fmt.Sprint(imp))
			}
		default:
			// Recipe
			val, ok := item.Value.(yaml.MapSlice)
			if !ok {
				return nil, fmt.Errorf("%v is of type %T, we expected a yaml.MapSlice", item.Value, item.Value)
			}

			recipe, err := parseRecipe(key, val)
			if err != nil {
				return nil, fmt.Errorf("parsing %v failed: %w", key, err)
			}
			rs.Recipes = append(rs.Recipes, recipe)
		}
	}

	return &rs, nil
}

// Traverse parses the import tree in a directory.
func Traverse(recipes *[]Recipe, path string) error {
	fi, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path=%s, no %s found", path, filepath.Base(path))
		}
		return fmt.Errorf("path=%s, %w", path, err)
	}
	defer fi.Close()

	res, err := Parse(fi)
	if err != nil {
		return err
	}
	*recipes = append(*recipes, res.Recipes...)

	for _, im := range res.Imports {
		err = Traverse(recipes,
			filepath.Join(
				filepath.Dir(path),
				im,
			),
		)
		if err != nil {
			return err
		}
	}

	return nil
}
