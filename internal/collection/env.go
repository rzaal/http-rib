package collection

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Env is a flat map of variable name -> value for one environment.
type Env map[string]string

// Envs holds all loaded environments keyed by name (filename without extension),
// e.g. "dev", "acceptance", "production".
type Envs struct {
	Dir    string
	byName map[string]Env
	names  []string // sorted, stable order
}

func LoadEnvs(collectionDir string) (*Envs, error) {
	dir := filepath.Join(collectionDir, "env")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return &Envs{Dir: dir, byName: map[string]Env{}}, nil
		}
		return nil, fmt.Errorf("read env dir: %w", err)
	}

	e := &Envs{Dir: dir, byName: map[string]Env{}}
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		ext := filepath.Ext(ent.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		name := ent.Name()[:len(ent.Name())-len(ext)]
		data, err := os.ReadFile(filepath.Join(dir, ent.Name()))
		if err != nil {
			return nil, fmt.Errorf("read env %s: %w", ent.Name(), err)
		}
		var env Env
		if err := yaml.Unmarshal(data, &env); err != nil {
			return nil, fmt.Errorf("parse env %s: %w", ent.Name(), err)
		}
		if env == nil {
			env = Env{}
		}
		e.byName[name] = env
		e.names = append(e.names, name)
	}
	sort.Strings(e.names)
	return e, nil
}

func (e *Envs) Names() []string {
	return e.names
}

func (e *Envs) Get(name string) Env {
	return e.byName[name]
}

// Default returns "dev" if present, otherwise the first env alphabetically,
// otherwise an empty Env with name "".
func (e *Envs) Default() (string, Env) {
	if env, ok := e.byName["dev"]; ok {
		return "dev", env
	}
	if len(e.names) > 0 {
		return e.names[0], e.byName[e.names[0]]
	}
	return "", Env{}
}
