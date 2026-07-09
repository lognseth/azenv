package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

func createContext(name string) (*ContextMeta, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	if _, err := ensureConfig(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(contextPath(name), 0700); err != nil {
		return nil, fmt.Errorf("create Azure config dir: %w", err)
	}
	meta, err := loadMeta(name)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	now := time.Now().UTC()
	if meta == nil {
		meta = &ContextMeta{Name: name, CreatedAt: now}
	}
	meta.Name = name
	meta.AzureConfigDir = contextPath(name)
	meta.UpdatedAt = now
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = now
	}
	return meta, saveMeta(meta)
}

func loadMeta(name string) (*ContextMeta, error) {
	b, err := os.ReadFile(metaPath(name))
	if err != nil {
		return nil, err
	}
	var meta ContextMeta
	if err := yaml.Unmarshal(b, &meta); err != nil {
		return nil, fmt.Errorf("read metadata for %q: %w", name, err)
	}
	if meta.Name == "" {
		meta.Name = name
	}
	if meta.AzureConfigDir == "" {
		meta.AzureConfigDir = contextPath(name)
	}
	return &meta, nil
}

func saveMeta(meta *ContextMeta) error {
	if err := validateName(meta.Name); err != nil {
		return err
	}
	meta.AzureConfigDir = contextPath(meta.Name)
	meta.UpdatedAt = time.Now().UTC()
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = meta.UpdatedAt
	}
	if err := os.MkdirAll(contextDir(meta.Name), 0700); err != nil {
		return err
	}
	b, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath(meta.Name), b, 0600)
}

func loadOrCreateMeta(name string) (*ContextMeta, error) {
	meta, err := loadMeta(name)
	if err == nil {
		return meta, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return createContext(name)
}

func listContexts() ([]ContextMeta, error) {
	entries, err := os.ReadDir(contextsBase())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var metas []ContextMeta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if _, err := os.Stat(contextPath(name)); err != nil {
			continue
		}
		meta, err := loadMeta(name)
		if errors.Is(err, os.ErrNotExist) {
			meta = &ContextMeta{Name: name, AzureConfigDir: contextPath(name)}
		} else if err != nil {
			return nil, err
		}
		metas = append(metas, *meta)
	}
	sort.Slice(metas, func(i, j int) bool { return metas[i].Name < metas[j].Name })
	return metas, nil
}

func contextExists(name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	if _, err := os.Stat(contextPath(name)); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("context %q does not exist; run: azenv create %s", name, name)
	} else if err != nil {
		return fmt.Errorf("cannot read context %q: %w", name, err)
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, info.Mode().Perm())
	})
}
