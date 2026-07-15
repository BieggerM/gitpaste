# gitpaste implementation

Goal: build a production-quality Go CLI that safely turns a single pasted, supported Git repository URL into a confirmed `git clone` operation through Bash or Zsh command-not-found hooks. The initial distribution is a shell-plugin-style GitHub Release, not a system package.

## Architecture and security decisions

- Keep URL validation in `internal/repositoryurl` as a strict allowlist for GitHub/GitLab HTTPS and SCP-style SSH URLs.
- Reject whitespace, control characters, shell metacharacters, credentials, ports, queries, fragments, malformed path segments, and option-like values before execution.
- Invoke Git only with `exec.Command("git", "clone", "--", validatedURL)`; never invoke a shell or reconstruct a command string.
- Keep confirmation and process execution in `internal/clone`, with dependency injection for deterministic tests and exact child exit-code propagation.
- Keep shell installation in `internal/shell`; install scripts under the user config directory and edit rc files only inside unique marker lines.
- Hooks gate on exactly one command word and a cheap URL pattern, then call `gitpaste clone -- "$value"`. They preserve normal not-found status and chain a pre-existing handler captured at source time where the shell permits it.
- Package hook scripts and documentation with release binaries through GoReleaser archives.
- Make `gitpaste setup` the primary shell-plugin entry point; retain the explicit install/uninstall commands for compatibility and scripting.

## Work plan

- [x] Initialize Go module and implement strict repository URL validation with table-driven tests.
- [x] Implement clone confirmation, cancellation, process execution, and exit propagation with tests.
- [x] Implement CLI parsing and integration tests for argument/error behavior.
- [x] Implement safe Bash/Zsh hooks and shell-hook detection tests.
- [x] Implement idempotent shell-hook install/uninstall with marker-scoped rc edits and tests.
- [x] Add `gitpaste setup` and revise documentation around the shell-plugin experience.
- [x] Simplify GoReleaser/GitHub Actions to GitHub archives and checksums only; defer Homebrew, OMZ, deb, and rpm.
- [x] Run formatting, static analysis, unit/integration tests, shell syntax checks, cross-builds, and release-config checks.

## Open details to verify

- [x] Local Git accepts `git clone -- <url>`.
- [x] Bash and Zsh syntax checks are available in the development environment.
- [x] GoReleaser is not installed locally; the configuration was checked against the current official GoReleaser v2 archive/action documentation and parsed as YAML. The tag workflow will perform the first actual GoReleaser build in a real Git repository.

## Verification result

- `go test ./...`, `go test -race ./...`, and `go vet ./...` pass.
- Linux/macOS amd64/arm64 cross-builds pass with CGO disabled.
- Bash and Zsh scripts pass syntax and behavioral integration tests.
- A temporary-home smoke test verified `gitpaste setup` and `uninstall-shell-hook` end to end.
- Git accepts the safe `git clone -- <url>` argument form.

## Terra review remediation

- [x] Make the Zsh hook idempotent and test repeated sourcing plus prior-handler status propagation.
- [x] Replace direct installer writes with checked, permission-controlled atomic writes; refuse unsafe hook-file symlinks while deliberately supporting symlinked rc files.
- [x] Reject relative `XDG_CONFIG_HOME` and report embedded-script upgrades as setup changes.
- [x] Tighten unambiguous GitHub/GitLab namespace constraints and add invalid-path tests.
- [x] Pin the release action to a verified commit SHA and correct archive filename documentation.
- [x] Re-run formatting, tests/race tests, vet, cross-builds, shell checks, setup smoke tests, and focused review.

Remediation verification: all Go tests and race tests pass, vet is clean, Bash/Zsh syntax checks pass, all four release targets cross-build, symlink and repeated-source regressions pass, and a temporary-home setup/second-setup/uninstall smoke test confirmed private modes and idempotent status reporting. CodeRabbit remained unavailable, so the final review was local.

## Worker-driven release pipeline

- [x] Remove CI annotations by updating pinned GitHub-owned actions and disabling the empty-module cache.
- [x] Add a manual semantic-version release input while preserving version-tag releases.
- [x] Verify on workers before publishing: tests, vet, formatting, shell syntax, and a GoReleaser snapshot.
- [x] Publish through a separately permissioned job, creating an idempotent annotated tag only for manual releases.
- [x] Add post-release checksum and Linux binary/setup smoke tests.
- [x] Validate locally, commit, push, and watch the worker run.

GitHub accepted both workflows, and the annotation-free test run for commit `9f19d93` passed. The release workflow is active but intentionally has not been dispatched, so no tag or release exists yet.
