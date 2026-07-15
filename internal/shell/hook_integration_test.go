package shell

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBashHookDetection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell integration requires POSIX")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not installed")
	}
	tests := []struct {
		name     string
		call     string
		wantCall bool
	}{
		{"HTTPS", `command_not_found_handle "https://github.com/owner/repo.git"`, true},
		{"SSH", `command_not_found_handle "git@gitlab.com:group/repo.git"`, true},
		{"multiple arguments", `command_not_found_handle "https://github.com/owner/repo" extra`, false},
		{"unsupported host", `command_not_found_handle "https://evil.example/owner/repo"`, false},
		{"metacharacters", `command_not_found_handle "https://github.com/owner/repo;whoami"`, false},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			called := runHookDriver(t, bash, "../../shell/gitpaste.bash", test.call)
			if called != test.wantCall {
				t.Fatalf("hook called gitpaste = %v, want %v", called, test.wantCall)
			}
		})
	}
}

func TestBashHookPreservesExistingHandler(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell integration requires POSIX")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not installed")
	}
	called := runHookDriver(t, bash, "../../shell/gitpaste.bash", `
command_not_found_handle() { printf 'original' > "$OUT"; }
. "$HOOK"
command_not_found_handle "https://github.com/owner/repo"
`)
	if !called {
		t.Fatal("existing Bash handler was not preserved")
	}
}

func TestZshHookDetectionAndChaining(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell integration requires POSIX")
	}
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh is not installed")
	}
	if !runHookDriver(t, zsh, "../../shell/gitpaste.zsh", `command_not_found_handler "https://gitlab.com/group/subgroup/repo.git"`) {
		t.Fatal("Zsh hook did not call gitpaste for a candidate URL")
	}
	if !runHookDriver(t, zsh, "../../shell/gitpaste.zsh", `
command_not_found_handler() { printf 'original' > "$OUT"; }
. "$HOOK"
command_not_found_handler not-a-url
`) {
		t.Fatal("Zsh hook did not chain to the existing handler")
	}
	if !runHookDriver(t, zsh, "../../shell/gitpaste.zsh", `
command_not_found_handler() { printf 'original' > "$OUT"; }
. "$HOOK"
. "$HOOK"
command_not_found_handler not-a-url
`) {
		t.Fatal("Zsh hook did not preserve chaining after repeated sourcing")
	}
	t.Run("prior handler status", func(t *testing.T) {
		dir := t.TempDir()
		hook, err := filepath.Abs("../../shell/gitpaste.zsh")
		if err != nil {
			t.Fatal(err)
		}
		driver := filepath.Join(dir, "driver")
		contents := "command_not_found_handler() { return 23; }\n. \"$HOOK\"\ncommand_not_found_handler not-a-url\nexit $?\n"
		if err := os.WriteFile(driver, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(zsh, driver)
		cmd.Env = append(os.Environ(), "HOOK="+hook)
		err = cmd.Run()
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) || exitError.ExitCode() != 23 {
			t.Fatalf("prior handler exit = %v, want 23", err)
		}
	})
}

func runHookDriver(t *testing.T, shellPath, hookPath, call string) bool {
	t.Helper()
	dir := t.TempDir()
	output := filepath.Join(dir, "called")
	mock := filepath.Join(dir, "gitpaste")
	if err := os.WriteFile(mock, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$OUT\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	absoluteHook, err := filepath.Abs(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	driver := filepath.Join(dir, "driver")
	contents := ". \"$HOOK\"\n" + call + "\n"
	if strings.HasPrefix(strings.TrimSpace(call), "command_not_found_handler()") || strings.HasPrefix(strings.TrimSpace(call), "command_not_found_handle()") {
		contents = call + "\n"
	}
	if err := os.WriteFile(driver, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(shellPath, driver)
	cmd.Env = append(os.Environ(), "PATH="+dir, "OUT="+output, "HOOK="+absoluteHook)
	combined, runErr := cmd.CombinedOutput()
	if runErr != nil {
		var exitErr *exec.ExitError
		if !errors.As(runErr, &exitErr) || exitErr.ExitCode() != 127 {
			t.Fatalf("shell driver: %v: %s", runErr, combined)
		}
	}
	data, err := os.ReadFile(output)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "original" {
		return true
	}
	if string(data) != "clone\n--\nhttps://github.com/owner/repo.git\n" &&
		string(data) != "clone\n--\ngit@gitlab.com:group/repo.git\n" &&
		string(data) != "clone\n--\nhttps://gitlab.com/group/subgroup/repo.git\n" {
		t.Fatalf("unexpected gitpaste arguments: %q", data)
	}
	return true
}
