package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newRootCommand(in io.Reader, out, errOut io.Writer) *cobra.Command {
	var root *cobra.Command
	root = &cobra.Command{
		Use:           "azenv",
		Short:         "Per-shell Azure CLI context helper",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetIn(in)
	root.SetOut(out)
	root.SetErr(errOut)

	root.AddCommand(
		newVersionCommand(out),
		newInitCommand(out),
		newCreateCommand(out),
		newUseCommand(out, errOut),
		newListCommand(out),
		newCurrentCommand(out),
		newRemoveCommand(out),
		newExecCommand(),
		newPathCommand(out),
		newLoginCommand(out),
		newLogoutCommand(out),
		newDoctorCommand(out),
		newCloneCommand(out),
		newRenameCommand(out),
		newStarshipCommand(out),
		newCompletionCommand(root, out),
	)
	return root
}

func newVersionCommand(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(out, version)
		},
	}
}

func newInitCommand(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "init [zsh|bash|fish]",
		Short: "Print shell integration",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := "zsh"
			if len(args) > 0 {
				shell = args[0]
			}
			switch shell {
			case "zsh", "bash":
				fmt.Fprint(out, `azenv() {
  case "$1" in
    use)
      shift
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
				fmt.Fprint(out, `function azenv
  if test "$argv[1]" = "use"
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
		},
	}
}

func newCreateCommand(out io.Writer) *cobra.Command {
	var login bool
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create an isolated Azure CLI context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			meta, err := createContext(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "Created context %q at %s\n", meta.Name, meta.AzureConfigDir)
			if login {
				if err := runAzWithConfig(meta.AzureConfigDir, "login"); err != nil {
					return err
				}
				refreshAzureMetadataBestEffort(meta)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&login, "login", false, "Run az login after creating the context")
	return cmd
}

func newUseCommand(out, errOut io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "use [name]",
		Short: "Switch the current shell to a context",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return completeContextNames(toComplete), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			} else {
				selected, err := pickContext()
				if err != nil {
					return err
				}
				name = selected
			}
			if err := contextExists(name); err != nil {
				return err
			}
			meta, err := loadOrCreateMeta(name)
			if err != nil {
				return err
			}
			meta.LastUsedAt = time.Now().UTC()
			refreshAzureMetadataBestEffort(meta)
			_ = saveMeta(meta)
			writeShellUse(out, meta)
			if os.Getenv("AZENV_WRAPPER") == "1" {
				fmt.Fprintf(errOut, "Using Azure context %q\n", name)
			} else {
				fmt.Fprintf(errOut, "azenv use emits shell code. Run: eval \"$(azenv init zsh)\"\n")
			}
			return nil
		},
	}
}

func newListCommand(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			metas, err := listContexts()
			if err != nil {
				return err
			}
			current := currentContextName()
			for _, meta := range metas {
				marker := " "
				if meta.Name == current {
					marker = "*"
				}
				details := strings.TrimSpace(strings.Join(nonEmpty(meta.SubscriptionName, meta.TenantName, meta.User), " | "))
				if details == "" {
					fmt.Fprintf(out, "%s %s\n", marker, meta.Name)
				} else {
					fmt.Fprintf(out, "%s %s  %s\n", marker, meta.Name, details)
				}
			}
			return nil
		},
	}
}

func newCurrentCommand(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show active context and Azure account",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := os.Getenv("AZURE_CONFIG_DIR")
			if cfg == "" {
				cfg = filepath.Join(homeDir(), ".azure")
			}
			name := inferContextName(cfg)
			fmt.Fprintf(out, "Context:          %s\n", valueOr(name, "default"))
			fmt.Fprintf(out, "AZURE_CONFIG_DIR: %s\n", cfg)
			if name != "" {
				if meta, err := loadMeta(name); err == nil {
					refreshAzureMetadataBestEffort(meta)
					if hasAzureMeta(meta) {
						printMeta(out, meta)
					} else {
						fmt.Fprintln(out, "Azure account:    not logged in or az unavailable")
					}
					return nil
				}
			}
			acc, err := azAccount(cfg)
			if err != nil {
				fmt.Fprintln(out, "Azure account:    not logged in or az unavailable")
				return nil
			}
			fmt.Fprintf(out, "Subscription:     %s\n", acc.Name)
			fmt.Fprintf(out, "Subscription ID:  %s\n", acc.ID)
			fmt.Fprintf(out, "Tenant ID:        %s\n", acc.TenantID)
			fmt.Fprintf(out, "User:             %s\n", acc.User.Name)
			return nil
		},
	}
}

func newRemoveCommand(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:     "rm <name>",
		Aliases: []string{"remove", "delete"},
		Short:   "Remove a context",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := validateName(name); err != nil {
				return err
			}
			if samePath(os.Getenv("AZURE_CONFIG_DIR"), contextPath(name)) {
				return fmt.Errorf("cannot remove active context in this shell; run: unset AZURE_CONFIG_DIR AZENV_CONTEXT")
			}
			if err := os.RemoveAll(contextDir(name)); err != nil {
				return err
			}
			fmt.Fprintf(out, "Removed context %q\n", name)
			return nil
		},
	}
}

func newExecCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "exec <name> -- <command...>",
		Short:              "Run one command in a context without switching shells",
		DisableFlagParsing: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completeContextNames(toComplete), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveDefault
		},
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("usage: azenv exec <name> -- <command...>")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			commandArgs := args[1:]
			if len(commandArgs) > 0 && commandArgs[0] == "--" {
				commandArgs = commandArgs[1:]
			}
			if len(commandArgs) == 0 {
				return fmt.Errorf("usage: azenv exec <name> -- <command...>")
			}
			if err := contextExists(name); err != nil {
				return err
			}
			meta, err := loadOrCreateMeta(name)
			if err != nil {
				return err
			}
			meta.LastUsedAt = time.Now().UTC()
			_ = saveMeta(meta)
			c := exec.Command(commandArgs[0], commandArgs[1:]...)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Env = append(os.Environ(), "AZURE_CONFIG_DIR="+meta.AzureConfigDir, "AZENV_CONTEXT="+name)
			return c.Run()
		},
	}
}

func newPathCommand(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "path <name>",
		Short: "Print a context AZURE_CONFIG_DIR path",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return completeContextNames(toComplete), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateName(args[0]); err != nil {
				return err
			}
			fmt.Fprintln(out, contextPath(args[0]))
			return nil
		},
	}
}

func newLoginCommand(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "login <name>",
		Short: "Run az login in a context",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return completeContextNames(toComplete), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			meta, err := loadOrCreateMeta(args[0])
			if err != nil {
				return err
			}
			if err := runAzWithConfig(meta.AzureConfigDir, "login"); err != nil {
				return err
			}
			refreshAzureMetadataBestEffort(meta)
			fmt.Fprintf(out, "Logged in context %q\n", meta.Name)
			return nil
		},
	}
}

func newLogoutCommand(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "logout [name]",
		Short: "Run az logout in a context",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return completeContextNames(toComplete), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := currentContextName()
			if len(args) == 1 {
				name = args[0]
			}
			if name == "" {
				return fmt.Errorf("no active azenv context; pass a context name")
			}
			if err := contextExists(name); err != nil {
				return err
			}
			if err := runAzWithConfig(contextPath(name), "logout"); err != nil {
				return err
			}
			meta, _ := loadOrCreateMeta(name)
			meta.SubscriptionID = ""
			meta.SubscriptionName = ""
			meta.TenantID = ""
			meta.TenantName = ""
			meta.User = ""
			_ = saveMeta(meta)
			fmt.Fprintf(out, "Logged out context %q\n", name)
			return nil
		},
	}
}

func newDoctorCommand(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check local azenv setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(out, "azenv: %s\n", version)
			fmt.Fprintf(out, "home: %s\n", azenvHome())
			fmt.Fprintf(out, "config: %s\n", configPath())
			fmt.Fprintf(out, "contexts: %s\n", contextsBase())
			checkTool(out, "az", "Azure CLI")
			checkTool(out, "fzf", "fzf picker")
			current := currentContextName()
			if current == "" {
				fmt.Fprintln(out, "current: default")
			} else {
				fmt.Fprintf(out, "current: %s\n", current)
			}
			return nil
		},
	}
}

func newCloneCommand(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "clone <source> <dest>",
		Short: "Clone a context",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completeContextNames(toComplete), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			src, dst := args[0], args[1]
			if err := contextExists(src); err != nil {
				return err
			}
			if err := validateName(dst); err != nil {
				return err
			}
			if _, err := os.Stat(contextDir(dst)); err == nil {
				return fmt.Errorf("context %q already exists", dst)
			}
			if err := copyDir(contextDir(src), contextDir(dst)); err != nil {
				return err
			}
			meta, err := loadOrCreateMeta(dst)
			if err != nil {
				return err
			}
			now := time.Now().UTC()
			meta.Name = dst
			meta.CreatedAt = now
			meta.LastUsedAt = time.Time{}
			if err := saveMeta(meta); err != nil {
				return err
			}
			fmt.Fprintf(out, "Cloned context %q to %q\n", src, dst)
			return nil
		},
	}
}

func newRenameCommand(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old> <new>",
		Short: "Rename a context",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completeContextNames(toComplete), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			oldName, newName := args[0], args[1]
			if err := contextExists(oldName); err != nil {
				return err
			}
			if err := validateName(newName); err != nil {
				return err
			}
			if samePath(os.Getenv("AZURE_CONFIG_DIR"), contextPath(oldName)) {
				return fmt.Errorf("cannot rename active context in this shell")
			}
			if _, err := os.Stat(contextDir(newName)); err == nil {
				return fmt.Errorf("context %q already exists", newName)
			}
			if err := os.Rename(contextDir(oldName), contextDir(newName)); err != nil {
				return err
			}
			meta, err := loadOrCreateMeta(newName)
			if err != nil {
				return err
			}
			meta.Name = newName
			if err := saveMeta(meta); err != nil {
				return err
			}
			fmt.Fprintf(out, "Renamed context %q to %q\n", oldName, newName)
			return nil
		},
	}
}

func newStarshipCommand(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "starship",
		Short: "Print current context for a Starship custom module",
		Run: func(cmd *cobra.Command, args []string) {
			if name := currentContextName(); name != "" {
				fmt.Fprintf(out, "󰠅 %s\n", name)
			}
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Print Starship custom module configuration",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(out, `[custom.azenv]
command = "azenv starship"
when = "test -n \"$AZURE_CONFIG_DIR\""
format = "[$output]($style) "
style = "blue bold"
`)
		},
	})
	return cmd
}

func newCompletionCommand(root *cobra.Command, out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish]",
		Short: "Generate shell completion script",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(out)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, true)
			default:
				return fmt.Errorf("unsupported shell %q; supported shells: bash, zsh, fish", args[0])
			}
		},
	}
}

func writeShellUse(out io.Writer, meta *ContextMeta) {
	if os.Getenv("AZENV_SHELL") == "fish" {
		fmt.Fprintf(out, "set -gx AZURE_CONFIG_DIR %s\n", shellQuote(meta.AzureConfigDir))
		fmt.Fprintf(out, "set -gx AZENV_CONTEXT %s\n", shellQuote(meta.Name))
		return
	}
	fmt.Fprintf(out, "export AZURE_CONFIG_DIR=%s\n", shellQuote(meta.AzureConfigDir))
	fmt.Fprintf(out, "export AZENV_CONTEXT=%s\n", shellQuote(meta.Name))
}

func pickContext() (string, error) {
	metas, err := listContexts()
	if err != nil {
		return "", err
	}
	if len(metas) == 0 {
		return "", fmt.Errorf("no contexts found; run: azenv create <name>")
	}
	if _, err := exec.LookPath("fzf"); err != nil {
		return "", fmt.Errorf("no context specified and fzf is not installed; run: azenv use <name>")
	}
	names := make([]string, 0, len(metas))
	for _, meta := range metas {
		names = append(names, meta.Name)
	}
	c := exec.Command("fzf", "--prompt=azenv> ")
	c.Stdin = strings.NewReader(strings.Join(names, "\n"))
	c.Stderr = os.Stderr
	selected, err := c.Output()
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(string(selected))
	if name == "" {
		return "", fmt.Errorf("no context selected")
	}
	return name, nil
}

func printMeta(out io.Writer, meta *ContextMeta) {
	if meta.SubscriptionName != "" {
		fmt.Fprintf(out, "Subscription:     %s\n", meta.SubscriptionName)
	}
	if meta.SubscriptionID != "" {
		fmt.Fprintf(out, "Subscription ID:  %s\n", meta.SubscriptionID)
	}
	if meta.TenantName != "" {
		fmt.Fprintf(out, "Tenant:           %s\n", meta.TenantName)
	}
	if meta.TenantID != "" {
		fmt.Fprintf(out, "Tenant ID:        %s\n", meta.TenantID)
	}
	if meta.User != "" {
		fmt.Fprintf(out, "User:             %s\n", meta.User)
	}
	if !meta.LastUsedAt.IsZero() {
		fmt.Fprintf(out, "Last used:        %s\n", meta.LastUsedAt.Format(time.RFC3339))
	}
}

func hasAzureMeta(meta *ContextMeta) bool {
	return meta.SubscriptionName != "" || meta.SubscriptionID != "" || meta.TenantName != "" || meta.TenantID != "" || meta.User != ""
}

func checkTool(out io.Writer, bin, label string) {
	if path, err := exec.LookPath(bin); err == nil {
		fmt.Fprintf(out, "%s: ok (%s)\n", label, path)
	} else {
		fmt.Fprintf(out, "%s: missing\n", label)
	}
}

func nonEmpty(values ...string) []string {
	var out []string
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}

func completeContextNames(prefix string) []string {
	metas, err := listContexts()
	if err != nil {
		return nil
	}
	var names []string
	for _, meta := range metas {
		if strings.HasPrefix(meta.Name, prefix) {
			names = append(names, meta.Name)
		}
	}
	return names
}

func valueOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
