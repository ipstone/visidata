package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathPrecedence(t *testing.T) {
	cwd := t.TempDir()
	home := t.TempDir()
	cwdPath := filepath.Join(cwd, DefaultConfigName)
	homePath := filepath.Join(home, DefaultConfigName)
	if err := os.WriteFile(cwdPath, []byte("preview_rows = 2\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(homePath, []byte("preview_rows = 3\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	path, err := ResolvePath("", "", cwd, home)
	if err != nil {
		t.Fatalf("ResolvePath returned error: %v", err)
	}
	if path != cwdPath {
		t.Fatalf("ResolvePath cwd precedence = %q, want %q", path, cwdPath)
	}

	path, err = ResolvePath("", "/tmp/from-env", cwd, home)
	if err != nil {
		t.Fatalf("ResolvePath returned error: %v", err)
	}
	if path != "/tmp/from-env" {
		t.Fatalf("ResolvePath env precedence = %q, want /tmp/from-env", path)
	}

	path, err = ResolvePath("/tmp/explicit", "/tmp/from-env", cwd, home)
	if err != nil {
		t.Fatalf("ResolvePath returned error: %v", err)
	}
	if path != "/tmp/explicit" {
		t.Fatalf("ResolvePath explicit precedence = %q, want /tmp/explicit", path)
	}
}

func TestLoadParsesTOMLConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), DefaultConfigName)
	content := []byte(`
filetype = "fixed"
header = 0
preview_rows = 5
table = "people"
theme = "amber"
show_hidden = true
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Filetype == nil || *cfg.Filetype != "fixed" {
		t.Fatalf("filetype = %#v, want fixed", cfg.Filetype)
	}
	if cfg.Header == nil || *cfg.Header != 0 {
		t.Fatalf("header = %#v, want 0", cfg.Header)
	}
	if cfg.PreviewRows == nil || *cfg.PreviewRows != 5 {
		t.Fatalf("preview_rows = %#v, want 5", cfg.PreviewRows)
	}
	if cfg.Table == nil || *cfg.Table != "people" {
		t.Fatalf("table = %#v, want people", cfg.Table)
	}
	if cfg.Theme == nil || *cfg.Theme != "amber" {
		t.Fatalf("theme = %#v, want amber", cfg.Theme)
	}
	if cfg.ShowHidden == nil || !*cfg.ShowHidden {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}
