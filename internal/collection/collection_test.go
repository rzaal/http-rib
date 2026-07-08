package collection

import (
	"os"
	"path/filepath"
	"strings"
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

	// requests/ starts empty aside from the reserved env/ folder
	flat := Flatten(coll.Tree)
	if len(flat) != 0 {
		t.Errorf("expected empty request tree on scaffold, got %d items: %v", len(flat), flat)
	}

	if len(coll.Envs.Names()) != 1 {
		t.Errorf("expected only dev env, got %d: %v", len(coll.Envs.Names()), coll.Envs.Names())
	}

	name, env := coll.Envs.Default()
	if name != "dev" {
		t.Errorf("Default() name = %q, want dev", name)
	}
	if env["baseUrl"] == "" {
		t.Errorf("expected baseUrl in dev env")
	}
	if env["token"] == "" {
		t.Errorf("expected secret token merged into dev env, got %v", env)
	}

	// secrets files must exist but stay out of git
	secretsPath := filepath.Join(dir, requestsDirName, envSubdir, "dev"+secretsSuffix)
	if _, err := os.Stat(secretsPath); err != nil {
		t.Errorf("expected secrets file at %s: %v", secretsPath, err)
	}
	examplePath := filepath.Join(dir, requestsDirName, envSubdir, "dev"+secretsExampleSuffix)
	if _, err := os.Stat(examplePath); err != nil {
		t.Errorf("expected secrets example file at %s: %v", examplePath, err)
	}
	gitignoreData, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("expected .gitignore to be created: %v", err)
	}
	if !strings.Contains(string(gitignoreData), "*"+secretsSuffix) {
		t.Errorf(".gitignore missing secrets pattern, got: %s", gitignoreData)
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
