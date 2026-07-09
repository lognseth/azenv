package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateContextWritesConfigAndMetadata(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AZENV_HOME", root)

	meta, err := createContext("prod")
	if err != nil {
		t.Fatalf("createContext() error = %v", err)
	}
	if meta.AzureConfigDir != filepath.Join(root, "contexts", "prod", "azure") {
		t.Fatalf("AzureConfigDir = %q", meta.AzureConfigDir)
	}
	if _, err := os.Stat(filepath.Join(root, "config.yaml")); err != nil {
		t.Fatalf("config.yaml not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "contexts", "prod", "metadata.yaml")); err != nil {
		t.Fatalf("metadata.yaml not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "contexts", "prod", "azure")); err != nil {
		t.Fatalf("azure dir not written: %v", err)
	}
}

func TestListContextsLoadsLegacyMissingMetadata(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AZENV_HOME", root)
	if err := os.MkdirAll(filepath.Join(root, "contexts", "dev", "azure"), 0700); err != nil {
		t.Fatal(err)
	}
	metas, err := listContexts()
	if err != nil {
		t.Fatalf("listContexts() error = %v", err)
	}
	if len(metas) != 1 || metas[0].Name != "dev" {
		t.Fatalf("listContexts() = %#v", metas)
	}
}
