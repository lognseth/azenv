package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func contextsBase() string {
	return filepath.Join(azenvHome(), "contexts")
}

func azenvHome() string {
	if v := os.Getenv("AZENV_HOME"); v != "" {
		return v
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "azenv")
	}
	return filepath.Join(homeDir(), ".config", "azenv")
}

func contextDir(name string) string  { return filepath.Join(contextsBase(), name) }
func contextPath(name string) string { return filepath.Join(contextDir(name), "azure") }
func metaPath(name string) string    { return filepath.Join(contextDir(name), "metadata.yaml") }
func configPath() string             { return filepath.Join(azenvHome(), "config.yaml") }
func homeDir() string                { h, _ := os.UserHomeDir(); return h }

func validateName(name string) error {
	if name == "" {
		return errors.New("context name cannot be empty")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("invalid context name %q; use letters, numbers, dots, underscores, and hyphens", name)
	}
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-' {
			continue
		}
		return fmt.Errorf("invalid context name %q; use letters, numbers, dots, underscores, and hyphens", name)
	}
	return nil
}

func currentContextName() string {
	if cfg := os.Getenv("AZURE_CONFIG_DIR"); cfg != "" {
		return inferContextName(cfg)
	}
	return ""
}

func inferContextName(cfg string) string {
	base := contextsBase()
	rel, err := filepath.Rel(filepath.Clean(base), filepath.Clean(cfg))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return ""
	}
	parts := strings.Split(rel, string(os.PathSeparator))
	if len(parts) == 2 && parts[0] != "." && parts[1] == "azure" {
		return parts[0]
	}
	return ""
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
