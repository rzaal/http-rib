package collection

import (
	"path/filepath"
	"testing"
)

func TestScaffoldAndLoad(t *testing.T) {
	dir := t.TempDir()

	if err := Scaffold(dir); err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}

	// scaffolding twice should error
	if err := Scaffold(dir); err == nil {
		t.Errorf("expected error scaffolding existing collection, got nil")
	}

	coll, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if coll.Manifest.Name != filepath.Base(dir) {
		t.Errorf("Manifest.Name = %q, want %q", coll.Manifest.Name, filepath.Base(dir))
	}

	flat := Flatten(coll.Tree)
	if len(flat) == 0 {
		t.Fatal("expected at least one item in tree")
	}

	found := false
	for _, fi := range flat {
		if fi.Item.Request != nil && fi.Item.Request.Method == "GET" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected sample GET request in tree")
	}

	if len(coll.Envs.Names()) != 3 {
		t.Errorf("expected 3 envs, got %d: %v", len(coll.Envs.Names()), coll.Envs.Names())
	}

	name, env := coll.Envs.Default()
	if name != "dev" {
		t.Errorf("Default() name = %q, want dev", name)
	}
	if env["baseUrl"] == "" {
		t.Errorf("expected baseUrl in dev env")
	}
}

func TestLoadNoCollection(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for missing collection")
	}
	var noColl *ErrNoCollection
	if _, ok := err.(*ErrNoCollection); !ok {
		_ = noColl
		t.Errorf("expected *ErrNoCollection, got %T", err)
	}
}
