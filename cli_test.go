package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func executeForTest(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := newRootCommand(strings.NewReader(""), &stdout, &stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func TestUseEmitsShellForContext(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AZENV_HOME", root)
	if _, err := createContext("prod"); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := executeForTest(t, "use", "prod")
	if err != nil {
		t.Fatalf("use error = %v", err)
	}
	if !strings.Contains(stdout, "export AZURE_CONFIG_DIR='"+filepath.Join(root, "contexts", "prod", "azure")+"'") {
		t.Fatalf("stdout = %q", stdout)
	}
	if !strings.Contains(stderr, "azenv use emits shell code") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestListMarksActiveContext(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AZENV_HOME", root)
	if _, err := createContext("prod"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AZURE_CONFIG_DIR", filepath.Join(root, "contexts", "prod", "azure"))

	stdout, _, err := executeForTest(t, "ls")
	if err != nil {
		t.Fatalf("ls error = %v", err)
	}
	if !strings.Contains(stdout, "* prod") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestCurrentReportsContextFromAzureConfigDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AZENV_HOME", root)
	if _, err := createContext("prod"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AZURE_CONFIG_DIR", filepath.Join(root, "contexts", "prod", "azure"))

	stdout, _, err := executeForTest(t, "current")
	if err != nil {
		t.Fatalf("current error = %v", err)
	}
	if !strings.Contains(stdout, "Context:          prod") {
		t.Fatalf("stdout = %q", stdout)
	}
	if !strings.Contains(stdout, "Azure account:    not logged in or az unavailable") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestExecSupportsDashDashDelimiter(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell command differs on Windows")
	}
	root := t.TempDir()
	t.Setenv("AZENV_HOME", root)
	if _, err := createContext("prod"); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(root, "exec.out")
	_, _, err := executeForTest(t, "exec", "prod", "--", "sh", "-c", "printf %s \"$AZURE_CONFIG_DIR\" > \"$1\"", "sh", outFile)
	if err != nil {
		t.Fatalf("exec error = %v", err)
	}
	b, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read exec output: %v", err)
	}
	want := filepath.Join(root, "contexts", "prod", "azure")
	if string(b) != want {
		t.Fatalf("exec output = %q, want %q", string(b), want)
	}
}

func TestCloneAndRename(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AZENV_HOME", root)
	if _, err := createContext("prod"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "contexts", "prod", "azure", "marker"), []byte("ok"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := executeForTest(t, "clone", "prod", "sandbox"); err != nil {
		t.Fatalf("clone error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "contexts", "sandbox", "azure", "marker")); err != nil {
		t.Fatalf("cloned marker missing: %v", err)
	}
	if _, _, err := executeForTest(t, "rename", "sandbox", "dev"); err != nil {
		t.Fatalf("rename error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "contexts", "dev", "azure", "marker")); err != nil {
		t.Fatalf("renamed marker missing: %v", err)
	}
}
