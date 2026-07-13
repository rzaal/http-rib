package collection

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

// EnvsDir returns the folder envs live in, relative to the collection root.
const EnvsDir = "requests/env"

// secretsSuffix marks the sibling file holding real secret values for an
// env; it is gitignored so tokens never get committed. A ".example" suffix
// on top of that is the committed template showing the expected shape.
const secretsSuffix = ".secrets.yaml"
const secretsExampleSuffix = secretsSuffix + ".example"

func LoadEnvs(collectionDir string) (*Envs, error) {
	dir := filepath.Join(collectionDir, EnvsDir)
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
		name, ok := baseEnvName(ent.Name())
		if !ok {
			continue
		}

		env, err := loadYAMLEnv(filepath.Join(dir, ent.Name()))
		if err != nil {
			return nil, fmt.Errorf("parse env %s: %w", ent.Name(), err)
		}

		secretsPath := filepath.Join(dir, name+secretsSuffix)
		if secrets, err := loadYAMLEnv(secretsPath); err == nil {
			for k, v := range secrets {
				env[k] = v
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("parse secrets for %s: %w", name, err)
		}

		e.byName[name] = env
		e.names = append(e.names, name)
	}
	sort.Strings(e.names)
	return e, nil
}

// baseEnvName returns the env name for a plain env file (e.g. "dev.yaml" ->
// "dev", true). Secrets files and their .example templates are not base env
// files and return ok=false.
func baseEnvName(filename string) (string, bool) {
	if strings.HasSuffix(filename, secretsExampleSuffix) || strings.HasSuffix(filename, secretsSuffix) {
		return "", false
	}
	ext := filepath.Ext(filename)
	if ext != ".yaml" && ext != ".yml" {
		return "", false
	}
	return filename[:len(filename)-len(ext)], true
}

func loadYAMLEnv(path string) (Env, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var env Env
	if err := yaml.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	if env == nil {
		env = Env{}
	}
	return env, nil
}

func (e *Envs) Names() []string {
	return e.names
}

func (e *Envs) Get(name string) Env {
	return e.byName[name]
}

// WriteEnvVar persists key=value into <envName>.secrets.yaml under
// collectionDir's env dir, creating the file if needed and merging with any
// existing keys. Used by post-response captures to save values like a
// session token without touching the committed base env file.
func WriteEnvVar(collectionDir, envName, key, value string) error {
	dir := filepath.Join(collectionDir, EnvsDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create env dir: %w", err)
	}

	path := filepath.Join(dir, envName+secretsSuffix)
	env, err := loadYAMLEnv(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read secrets for %s: %w", envName, err)
		}
		env = Env{}
	}

	env[key] = value

	data, err := yaml.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal secrets for %s: %w", envName, err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write secrets for %s: %w", envName, err)
	}
	return nil
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
