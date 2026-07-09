package main

import (
	"path/filepath"
	"testing"
)

func TestPathsUseAzenvHomeAsSourceOfTruth(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AZENV_HOME", root)

	if got := configPath(); got != filepath.Join(root, "config.yaml") {
		t.Fatalf("configPath() = %q", got)
	}
	if got := contextPath("prod"); got != filepath.Join(root, "contexts", "prod", "azure") {
		t.Fatalf("contextPath() = %q", got)
	}
	if got := metaPath("prod"); got != filepath.Join(root, "contexts", "prod", "metadata.yaml") {
		t.Fatalf("metaPath() = %q", got)
	}
}

func TestInferContextNameFromAzureConfigDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AZENV_HOME", root)

	cfg := filepath.Join(root, "contexts", "prod", "azure")
	if got := inferContextName(cfg); got != "prod" {
		t.Fatalf("inferContextName() = %q, want prod", got)
	}
	if got := inferContextName(filepath.Join(root, "contexts", "prod")); got != "" {
		t.Fatalf("inferContextName() = %q, want empty", got)
	}
}
