# gitpaste

`gitpaste` turns a GitHub or GitLab repository URL pasted directly into Bash or Zsh into a safe, confirmed `git clone`. It uses the shell's command-not-found hook, so ordinary commands and ordinary command-not-found behavior remain untouched.

```console
$ git@github.com:owner/repository.git
gitpaste: Clone git@github.com:owner/repository.git? [Y/n]
Cloning into 'repository'...
```

## Installation

Download the archive for your OS and architecture from [GitHub Releases](https://github.com/bieggerm/gitpaste/releases), extract it, and put the binary on `PATH`:

```bash
tar -xzf gitpaste_0.1.0_linux_amd64.tar.gz
install -m 0755 gitpaste ~/.local/bin/gitpaste
gitpaste setup
```

Archive names use `linux` or `darwin` and `amd64` or `arm64`. Ensure `~/.local/bin` is on `PATH`. To build from source with Go 1.22 or newer:

```bash
go install github.com/bieggerm/gitpaste/cmd/gitpaste@latest
```

Then enable integration for each detected Bash/Zsh rc file:

```bash
gitpaste setup
```

`setup` installs scripts below `${XDG_CONFIG_HOME:-$HOME/.config}/gitpaste` and adds a uniquely marked source block to existing `~/.bashrc` and/or `~/.zshrc`. If neither exists, it creates the rc file for `$SHELL`. Restart the shell or source the updated rc file. `install-shell-hook` remains an explicit alias for scripts and older instructions.

Zsh safely copies and chains an existing `command_not_found_handler`. Bash cannot safely copy a function handler without dynamic evaluation; if `command_not_found_handle` already exists, gitpaste warns and leaves it unchanged. Users with a distribution-provided Bash handler must choose how to combine the two functions manually.

## Usage

```text
gitpaste clone [--yes] [--] <repository-url>
gitpaste validate [--] <repository-url>
gitpaste setup
gitpaste install-shell-hook
gitpaste uninstall-shell-hook
gitpaste version
```

Confirmation is required by default. `--yes` skips it for direct CLI use. The shell hook never supplies `--yes`.

Supported forms are:

```text
https://github.com/owner/repository
https://github.com/owner/repository.git
git@github.com:owner/repository.git
https://gitlab.com/group/subgroup/repository.git
git@gitlab.com:group/repository.git
```

GitHub paths must have exactly an owner and repository; owners follow GitHub's alphanumeric/single-hyphen account-name rules. GitLab paths may contain subgroups and follow GitLab's slug boundaries, including no consecutive special characters. Path segments accept ASCII letters, digits, `.`, `_`, and `-`; an optional final `.git` suffix is accepted.

## Security model

Every pasted value is untrusted. The hook only forwards a single command word that resembles a supported URL. The Go validator then applies a strict allowlist and rejects whitespace, control characters, shell metacharacters, extra commands, credentials, ports, queries, fragments, unsupported hosts/protocols, and malformed paths. Git is executed directly as separate process arguments using the option terminator: `git clone -- <validated-url>`. No shell evaluation or command-string interpolation is used.

Shell hooks run after the shell parses the input. They cannot stop separate shell syntax that was never part of the URL argument—for example, an unquoted `; other-command` pasted after a URL. Only paste a repository URL as a single command; gitpaste is not a sandbox for arbitrary pasted shell text.

Cloning a repository does not itself execute repository code. A repository can still contain malicious scripts, build files, hooks intended for later installation, or misleading instructions, so inspect it before running scripts or builds.

See [ARCHITECTURE.md](ARCHITECTURE.md) for the detailed trust boundaries.

## Uninstall

```bash
gitpaste uninstall-shell-hook
```

Only the marker-bounded gitpaste blocks and installed user-local hook scripts are removed. Remove the binary separately from wherever it was installed.

## Development and release

```bash
gofmt -w .
go vet ./...
go test -race ./...
go build ./cmd/gitpaste
goreleaser release --snapshot --clean
```

Run the release workflow with a semantic version to have GitHub's workers verify, tag, publish, checksum, and smoke-test the release:

```bash
gh workflow run release.yml --ref main -f version=v0.1.0-rc.1
gh run watch --exit-status
```

Pushing an existing semantic version tag such as `v0.1.0` also starts the same pipeline. Releases contain Linux/macOS `amd64`/`arm64` `.tar.gz` archives and `checksums.txt`. Homebrew, an optional Oh My Zsh wrapper, and `.deb`/`.rpm` assets are intentionally deferred until this release flow is established.

Licensed under the MIT License.
