package collection

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manifest is the parsed ribnip.yaml at the collection root.
type Manifest struct {
	Name    string `yaml:"name"`
	Version int    `yaml:"version"`
}

// Item is a node in the request tree: either a directory (Children set) or a
// leaf request (Request set).
type Item struct {
	Label    string // display name (dir name or request Name)
	Path     string // relative path from requests/ root, used as a stable key
	Request  *Request
	Children []*Item
}

// Collection is a loaded ribnip collection: manifest + request tree + envs.
type Collection struct {
	Dir      string
	Manifest Manifest
	Tree     []*Item
	Envs     *Envs
}

const manifestName = "ribnip.yaml"
const requestsDirName = "requests"

// envSubdir is the requests/ subfolder reserved for environments; it's
// loaded via LoadEnvs and excluded from the browsable request tree.
const envSubdir = "env"

// ErrNoCollection is returned by Load when dir has no ribnip.yaml.
type ErrNoCollection struct{ Dir string }

func (e *ErrNoCollection) Error() string {
	return fmt.Sprintf("no collection found in %s (missing %s)", e.Dir, manifestName)
}

// Load reads a collection rooted at dir. Returns *ErrNoCollection if dir has
// no ribnip.yaml.
func Load(dir string) (*Collection, error) {
	manifestPath := filepath.Join(dir, manifestName)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &ErrNoCollection{Dir: dir}
		}
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	envs, err := LoadEnvs(dir)
	if err != nil {
		return nil, fmt.Errorf("load envs: %w", err)
	}

	tree, err := loadTree(filepath.Join(dir, requestsDirName), "")
	if err != nil {
		return nil, fmt.Errorf("load requests: %w", err)
	}

	return &Collection{Dir: dir, Manifest: m, Tree: tree, Envs: envs}, nil
}

func loadTree(dir, relPrefix string) ([]*Item, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	var items []*Item
	for _, ent := range entries {
		full := filepath.Join(dir, ent.Name())
		rel := ent.Name()
		if relPrefix != "" {
			rel = filepath.Join(relPrefix, ent.Name())
		}
		if ent.IsDir() {
			if relPrefix == "" && ent.Name() == envSubdir {
				continue // reserved for environments, not a browsable request folder
			}
			children, err := loadTree(full, rel)
			if err != nil {
				return nil, err
			}
			if len(children) == 0 {
				continue
			}
			items = append(items, &Item{Label: ent.Name(), Path: rel, Children: children})
			continue
		}
		ext := filepath.Ext(ent.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		req, err := LoadRequest(full)
		if err != nil {
			return nil, err
		}
		label := req.Name
		if label == "" {
			label = strings.TrimSuffix(ent.Name(), ext)
		}
		items = append(items, &Item{Label: label, Path: rel, Request: req})
	}
	return items, nil
}

// Flatten walks the tree depth-first into a flat, display-ordered list of
// items (both dirs and leaves), useful for a flat list widget with indent
// depth tracked alongside.
type FlatItem struct {
	Item  *Item
	Depth int
}

func Flatten(tree []*Item) []FlatItem {
	var out []FlatItem
	var walk func(items []*Item, depth int)
	walk = func(items []*Item, depth int) {
		for _, it := range items {
			out = append(out, FlatItem{Item: it, Depth: depth})
			if it.Children != nil {
				walk(it.Children, depth+1)
			}
		}
	}
	walk(tree, 0)
	return out
}

// Scaffold creates a starter collection at dir: manifest, a dev env
// (committed non-secret file plus a gitignored .secrets.yaml sibling for
// real keys, and a committed .secrets.yaml.example template), and a
// .gitignore. It errors if a manifest already exists. No sample request is
// created — the requests/ folder starts empty.
func Scaffold(dir string) error {
	manifestPath := filepath.Join(dir, manifestName)
	if _, err := os.Stat(manifestPath); err == nil {
		return fmt.Errorf("collection already exists at %s", manifestPath)
	}

	name := filepath.Base(dir)
	manifest := fmt.Sprintf("name: %s\nversion: 1\n", name)
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	envDir := filepath.Join(dir, requestsDirName, envSubdir)
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		return fmt.Errorf("create env dir: %w", err)
	}

	base := "baseUrl: \"https://httpbin.org\"\n"
	if err := os.WriteFile(filepath.Join(envDir, "dev.yaml"), []byte(base), 0o644); err != nil {
		return fmt.Errorf("write dev env: %w", err)
	}

	secrets := "token: \"dev-secret\"\n"
	if err := os.WriteFile(filepath.Join(envDir, "dev"+secretsSuffix), []byte(secrets), 0o600); err != nil {
		return fmt.Errorf("write dev secrets: %w", err)
	}

	example := "token: \"changeme\"\n"
	if err := os.WriteFile(filepath.Join(envDir, "dev"+secretsExampleSuffix), []byte(example), 0o644); err != nil {
		return fmt.Errorf("write dev secrets example: %w", err)
	}

	gitignorePath := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		ignore := fmt.Sprintf("%s/%s/*%s\n", requestsDirName, envSubdir, secretsSuffix)
		if err := os.WriteFile(gitignorePath, []byte(ignore), 0o644); err != nil {
			return fmt.Errorf("write .gitignore: %w", err)
		}
	}

	return nil
}
