package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const version = "0.1.2"

type ContextInfo struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	Subscription string `json:"subscription,omitempty"`
	Tenant       string `json:"tenant,omitempty"`
	User         string `json:"user,omitempty"`
}

type AzAccount struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	TenantID string `json:"tenantId"`
	User     struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"user"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "azenv:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		printHelp()
		return nil
	}

	switch args[0] {
	case "version":
		fmt.Println(version)
	case "init":
		return cmdInit(args[1:])
	case "create":
		return cmdCreate(args[1:])
	case "use":
		return cmdUse(args[1:])
	case "ls", "list":
		return cmdList()
	case "current":
		return cmdCurrent()
	case "rm", "remove", "delete":
		return cmdRemove(args[1:])
	case "exec":
		return cmdExec(args[1:])
	case "path":
		return cmdPath(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
	return nil
}

func printHelp() {
	fmt.Print(`azenv - per-shell Azure CLI context helper

Usage:
  azenv init <zsh|bash|fish>          Print shell integration
  azenv create <name> [--login]       Create isolated Azure CLI context
  azenv use <name>                    Emit shell code to use a context
  azenv ls                            List contexts
  azenv current                       Show active context and Azure account
  azenv rm <name>                     Remove context
  azenv exec <name> -- <command...>   Run one command in a context
  azenv path <name>                   Print AZURE_CONFIG_DIR path
  azenv version                       Print version

Recommended setup:
  eval "$(azenv init zsh)"

Then use:
  azenv create prod --login
  azenv use prod
  az account set -s <subscription>    # only once per context
`)
}

func cmdInit(args []string) error {
	shell := "zsh"
	if len(args) > 0 {
		shell = args[0]
	}
	if len(args) > 1 {
		return errors.New("usage: azenv init <zsh|bash|fish>")
	}

	switch shell {
	case "zsh", "bash":
		fmt.Print(`azenv() {
  case "$1" in
    use)
      shift
      if [ "$#" -eq 0 ]; then
        command azenv use
        return $?
      fi
      local __azenv_script
      __azenv_script="$(AZENV_WRAPPER=1 command azenv use "$@")" || return $?
      eval "$__azenv_script"
      ;;
    *)
      command azenv "$@"
      ;;
  esac
}
`)
	case "fish":
		fmt.Print(`function azenv
  if test "$argv[1]" = "use"
    if test (count $argv) -lt 2
      command azenv use
      return $status
    end
    env AZENV_WRAPPER=1 AZENV_SHELL=fish azenv use $argv[2..-1] | source
  else
    command azenv $argv
  end
end
`)
	default:
		return fmt.Errorf("unsupported shell %q; supported shells: zsh, bash, fish", shell)
	}
	return nil
}

func cmdCreate(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: azenv create <name> [--login]")
	}
	name := args[0]
	if err := validateName(name); err != nil {
		return err
	}
	login := false
	for _, arg := range args[1:] {
		switch arg {
		case "--login":
			login = true
		default:
			return fmt.Errorf("unknown option %q; usage: azenv create <name> [--login]", arg)
		}
	}
	p := contextPath(name)
	if err := os.MkdirAll(p, 0700); err != nil {
		return err
	}
	fmt.Printf("Created context %q at %s\n", name, p)
	if login {
		return runAzWithConfig(p, "login")
	}
	return nil
}

func cmdUse(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: azenv use <name>")
	}
	if len(args) > 1 {
		return errors.New("usage: azenv use <name>")
	}
	name := args[0]
	if err := validateName(name); err != nil {
		return err
	}
	p := contextPath(name)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return fmt.Errorf("context %q does not exist; run: azenv create %s --login", name, name)
	} else if err != nil {
		return fmt.Errorf("cannot read context %q: %w", name, err)
	}

	// Important: stdout is shell code because the wrapper evals this output.
	if os.Getenv("AZENV_SHELL") == "fish" {
		fmt.Printf("set -gx AZURE_CONFIG_DIR %s\n", shellQuote(p))
		fmt.Printf("set -gx AZENV_CONTEXT %s\n", shellQuote(name))
	} else {
		fmt.Printf("export AZURE_CONFIG_DIR=%s\n", shellQuote(p))
		fmt.Printf("export AZENV_CONTEXT=%s\n", shellQuote(name))
	}
	if os.Getenv("AZENV_WRAPPER") == "1" {
		fmt.Fprintf(os.Stderr, "Using Azure context %q\n", name)
	} else {
		fmt.Fprintf(os.Stderr, "azenv use emits shell code. Run: eval \"$(azenv init zsh)\"\n")
	}
	return nil
}

func cmdList() error {
	entries, err := os.ReadDir(contextsBase())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	names := []string{}
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	current := currentContextName()
	for _, n := range names {
		marker := " "
		if n == current {
			marker = "*"
		}
		fmt.Printf("%s %s\n", marker, n)
	}
	return nil
}

func cmdCurrent() error {
	cfg := os.Getenv("AZURE_CONFIG_DIR")
	if cfg == "" {
		cfg = filepath.Join(homeDir(), ".azure")
	}
	name := inferContextName(cfg)

	fmt.Printf("Context:          %s\n", valueOr(name, "default"))
	fmt.Printf("AZURE_CONFIG_DIR: %s\n", cfg)

	acc, err := azAccount(cfg)
	if err != nil {
		fmt.Println("Azure account:    not logged in or az unavailable")
		return nil
	}
	fmt.Printf("Subscription:     %s\n", acc.Name)
	fmt.Printf("Subscription ID:  %s\n", acc.ID)
	fmt.Printf("Tenant ID:        %s\n", acc.TenantID)
	fmt.Printf("User:             %s\n", acc.User.Name)
	return nil
}

func cmdRemove(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: azenv rm <name>")
	}
	name := args[0]
	if err := validateName(name); err != nil {
		return err
	}
	p := contextPath(name)
	if samePath(os.Getenv("AZURE_CONFIG_DIR"), p) {
		return errors.New("cannot remove active context in this shell; run: unset AZURE_CONFIG_DIR AZENV_CONTEXT")
	}
	if err := os.RemoveAll(p); err != nil {
		return err
	}
	fmt.Printf("Removed context %q\n", name)
	return nil
}

func cmdExec(args []string) error {
	if len(args) < 3 || args[1] != "--" {
		return errors.New("usage: azenv exec <name> -- <command...>")
	}
	name := args[0]
	if err := validateName(name); err != nil {
		return err
	}
	p := contextPath(name)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return fmt.Errorf("context %q does not exist", name)
	} else if err != nil {
		return fmt.Errorf("cannot read context %q: %w", name, err)
	}

	c := exec.Command(args[2], args[3:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = append(os.Environ(), "AZURE_CONFIG_DIR="+p, "AZENV_CONTEXT="+name)
	return c.Run()
}

func cmdPath(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: azenv path <name>")
	}
	name := args[0]
	if err := validateName(name); err != nil {
		return err
	}
	fmt.Println(contextPath(name))
	return nil
}

func azAccount(cfg string) (*AzAccount, error) {
	c := exec.Command("az", "account", "show", "-o", "json")
	c.Env = append(os.Environ(), "AZURE_CONFIG_DIR="+cfg)
	out, err := c.Output()
	if err != nil {
		return nil, err
	}
	var acc AzAccount
	if err := json.Unmarshal(out, &acc); err != nil {
		return nil, err
	}
	return &acc, nil
}

func runAzWithConfig(cfg string, args ...string) error {
	c := exec.Command("az", args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = append(os.Environ(), "AZURE_CONFIG_DIR="+cfg)
	return c.Run()
}

func contextsBase() string {
	if v := os.Getenv("AZENV_HOME"); v != "" {
		return filepath.Join(v, "contexts")
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "azenv", "contexts")
	}
	return filepath.Join(homeDir(), ".config", "azenv", "contexts")
}

func contextPath(name string) string { return filepath.Join(contextsBase(), name, ".azure") }
func homeDir() string                { h, _ := os.UserHomeDir(); return h }
func valueOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

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
	if len(parts) == 2 && parts[0] != "." && parts[1] == ".azure" {
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
