package clone

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestConfirmationAndSafeArguments(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a POSIX shell script")
	}
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	writeFakeGit(t, dir, "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$GITPASTE_ARGS_FILE\"\n")
	t.Setenv("PATH", dir)
	t.Setenv("GITPASTE_ARGS_FILE", argsFile)

	var stdout, stderr bytes.Buffer
	code, err := Run(context.Background(), "https://github.com/owner/repo.git", Options{
		Input: strings.NewReader("yes\n"), Output: &stdout, Error: &stderr,
	})
	if err != nil || code != 0 {
		t.Fatalf("Run() = (%d, %v), stderr=%q", code, err, stderr.String())
	}
	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "clone\n--\nhttps://github.com/owner/repo.git\n" {
		t.Fatalf("git arguments = %q", got)
	}
	if !strings.Contains(stderr.String(), "Clone https://github.com/owner/repo.git?") {
		t.Fatalf("missing prompt: %q", stderr.String())
	}
}

func TestCancellationDoesNotRunGit(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	var stdout, stderr bytes.Buffer
	code, err := Run(context.Background(), "git@github.com:owner/repo.git", Options{
		Input: strings.NewReader("n\n"), Output: &stdout, Error: &stderr,
	})
	if err != nil || code != 0 {
		t.Fatalf("Run() = (%d, %v)", code, err)
	}
	if !strings.Contains(stderr.String(), "clone cancelled") {
		t.Fatalf("missing cancellation message: %q", stderr.String())
	}
}

func TestEmptyInputDoesNotConfirm(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	code, err := Run(context.Background(), "git@github.com:owner/repo.git", Options{
		Input: strings.NewReader(""), Output: &bytes.Buffer{}, Error: &bytes.Buffer{},
	})
	if code != 1 || err == nil || !strings.Contains(err.Error(), "before an answer") {
		t.Fatalf("Run() = (%d, %v)", code, err)
	}
}

func TestMissingGit(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	code, err := Run(context.Background(), "https://github.com/owner/repo", Options{
		Yes: true, Input: strings.NewReader(""), Output: &bytes.Buffer{}, Error: &bytes.Buffer{},
	})
	if code != 127 || err == nil || !strings.Contains(err.Error(), "could not start git") {
		t.Fatalf("Run() = (%d, %v)", code, err)
	}
}

func TestGitExitCodePropagation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a POSIX shell script")
	}
	dir := t.TempDir()
	writeFakeGit(t, dir, "#!/bin/sh\nexit 42\n")
	t.Setenv("PATH", dir)
	code, err := Run(context.Background(), "https://gitlab.com/group/repo", Options{
		Yes: true, Input: strings.NewReader(""), Output: &bytes.Buffer{}, Error: &bytes.Buffer{},
	})
	if err != nil || code != 42 {
		t.Fatalf("Run() = (%d, %v)", code, err)
	}
}

func writeFakeGit(t *testing.T, dir, contents string) {
	t.Helper()
	path := filepath.Join(dir, "git")
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}
}
