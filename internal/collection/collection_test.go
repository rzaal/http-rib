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
	envDir := filepath.Join(dir, requestsDirName, envSubdir)
	gitignoreData, err := os.ReadFile(filepath.Join(envDir, ".gitignore"))
	if err != nil {
		t.Fatalf("expected .gitignore to be created: %v", err)
	}
	if !strings.Contains(string(gitignoreData), "*"+secretsSuffix) {
		t.Errorf(".gitignore missing secrets pattern, got: %s", gitignoreData)
	}
}

func TestScaffoldAppendsToExistingGitignore(t *testing.T) {
	dir := t.TempDir()
	envDir := filepath.Join(dir, requestsDirName, envSubdir)
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatalf("setup: mkdir env dir: %v", err)
	}

	preexisting := "node_modules\n"
	if err := os.WriteFile(filepath.Join(envDir, ".gitignore"), []byte(preexisting), 0o644); err != nil {
		t.Fatalf("setup: write .gitignore: %v", err)
	}

	if err := Scaffold(dir); err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(envDir, ".gitignore"))
	if err != nil {
		t.Fatalf("expected .gitignore to exist: %v", err)
	}
	got := string(data)

	if !strings.Contains(got, "node_modules") {
		t.Errorf("expected pre-existing content preserved, got: %s", got)
	}
	pattern := "*" + secretsSuffix
	if !strings.Contains(got, pattern) {
		t.Errorf("expected secrets pattern appended, got: %s", got)
	}

	// re-running ensureGitignore should not duplicate the line
	if err := ensureGitignore(envDir); err != nil {
		t.Fatalf("ensureGitignore() error = %v", err)
	}
	data2, err := os.ReadFile(filepath.Join(envDir, ".gitignore"))
	if err != nil {
		t.Fatalf("re-read .gitignore: %v", err)
	}
	if strings.Count(string(data2), pattern) != 1 {
		t.Errorf("expected pattern to appear exactly once, got: %s", data2)
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
