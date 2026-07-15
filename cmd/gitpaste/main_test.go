package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestArgumentHandling(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		code int
		want string
	}{
		{"no command", nil, 2, "usage:"},
		{"unknown command", []string{"wat"}, 2, "unknown command"},
		{"clone missing URL", []string{"clone"}, 2, "requires exactly one"},
		{"clone extra command", []string{"clone", "git@github.com:owner/repo.git", "whoami"}, 2, "requires exactly one"},
		{"setup extra argument", []string{"setup", "zsh"}, 2, "does not accept arguments"},
		{"validate invalid", []string{"validate", "https://evil.example/owner/repo"}, 1, "invalid repository URL"},
		{"validate valid", []string{"validate", "--", "https://github.com/owner/repo"}, 0, "valid repository URL"},
		{"version", []string{"version"}, 0, "gitpaste dev"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			code := run(context.Background(), test.args, strings.NewReader(""), &stdout, &stderr)
			if code != test.code {
				t.Fatalf("run() code = %d, want %d; stdout=%q stderr=%q", code, test.code, stdout.String(), stderr.String())
			}
			if combined := stdout.String() + stderr.String(); !strings.Contains(combined, test.want) {
				t.Fatalf("run() output %q does not contain %q", combined, test.want)
			}
		})
	}
}

func TestParseCloneArgs(t *testing.T) {
	t.Parallel()
	url := "git@github.com:owner/repo.git"
	tests := []struct {
		args    []string
		wantURL string
		wantYes bool
		wantErr bool
	}{
		{[]string{url}, url, false, false},
		{[]string{"--yes", url}, url, true, false},
		{[]string{"--yes", "--", url}, url, true, false},
		{[]string{"--", url}, url, false, false},
		{[]string{"--yes", "--yes", url}, "", false, true},
		{[]string{url, "extra"}, "", false, true},
	}
	for _, test := range tests {
		gotURL, gotYes, err := parseCloneArgs(test.args)
		if (err != nil) != test.wantErr || gotURL != test.wantURL || gotYes != test.wantYes {
			t.Errorf("parseCloneArgs(%q) = (%q, %v, %v)", test.args, gotURL, gotYes, err)
		}
	}
}
