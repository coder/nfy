package parse

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"os"
	"path/filepath"
)

type Recipe struct {
	Name         string
	Install      string
	Check        string
	BuildOnly    bool
	Comment      string
	Dependencies []string
}

type Result struct {
	Imports []string
	Recipes []Recipe
}

func expectError(field, typ string) error {
	return fmt.Errorf("expected %q to be string", field, typ)
}

func parseRecipe(key string, val yaml.MapSlice) (Recipe, error) {
	var r Recipe
	var ok bool
	for _, it := range val {
		switch fmt.Sprint(it.Key) {
		case "install":
			r.Install, ok = it.Value.(string)
			if !ok {
				return r, expectError("install", "string")
			}
		case "check":
			r.Check, ok = it.Value.(string)
			if !ok {
				return r, expectError("check", "string")
			}
		case "build_only":
			r.BuildOnly, ok = it.Value.(bool)
			if !ok {
				return r, expectError("build_only", "bool")
			}
		case "comment":
			r.Comment, ok = it.Value.(string)
			if !ok {
				return r, expectError("comment", "string")
			}
		case "deps":
			deps, ok := it.Value.([]interface{})
			if !ok {
				return r, expectError("deps", "array")
			}
			for _, dep := range deps {
				r.Dependencies = append(r.Dependencies, fmt.Sprint(dep))
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
