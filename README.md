# azenv

`azenv` gives Azure CLI named, per-shell contexts by isolating `AZURE_CONFIG_DIR` per context.

It does not replace `az`. It only manages which Azure CLI config directory your current shell uses.

## Install from source

```bash
./install.sh
```

By default this installs to `~/.local/bin/azenv`.

For zsh, make sure `~/.local/bin` is on your `PATH`:

```zsh
mkdir -p ~/.local/bin
export PATH="$HOME/.local/bin:$PATH"
```

Add that `PATH` line to `~/.zshrc` if it is not already there.

To install to `/usr/local/bin` instead:

```bash
./install.sh --system
```

## Shell integration

For zsh:

```bash
eval "$(azenv init zsh)"
```

Put that in `~/.zshrc` after installing the binary. This defines a shell function so `azenv use <context>` can update `AZURE_CONFIG_DIR` in the current terminal.

For bash:

```bash
eval "$(azenv init bash)"
```

For fish:

```fish
azenv init fish | source
```

## Usage

Create a context:

```zsh
azenv create dev
```

Activate it in the current zsh terminal:

```zsh
azenv use dev
echo $AZURE_CONFIG_DIR
azenv current
```

If `azenv use dev` prints `export AZURE_CONFIG_DIR=...` instead of switching, load the shell integration first:

```zsh
eval "$(azenv init zsh)"
```

Create a context and log in immediately:

```bash
azenv create prod --login
```

Activate it in the current terminal:

```bash
azenv use prod
```

Set the subscription once:

```bash
az account set -s <subscription-id-or-name>
```

Next time, just run:

```bash
azenv use prod
```

List contexts:

```bash
azenv ls
```

The active context is marked with `*`.

Show current context and Azure account:

```bash
azenv current
```

Run a single command in another context without switching the current shell:

```bash
azenv exec prod -- az account show
azenv exec staging -- terraform plan
```

Remove a context:

```bash
azenv rm prod
```

## Storage

By default, contexts live here:

```text
~/.config/azenv/contexts/<name>/.azure
```

Override with:

```bash
export AZENV_HOME="$HOME/.azenv"
```

Then contexts live under:

```text
$AZENV_HOME/contexts/<name>/.azure
```

## Notes

A standalone program cannot directly change the parent shell environment. That is why `azenv use` needs the shell integration from `azenv init`. The wrapper evaluates the export statements emitted by the binary.

## Release Notes

See [CHANGELOG.md](CHANGELOG.md).

## License

MIT. See [LICENSE](LICENSE).
