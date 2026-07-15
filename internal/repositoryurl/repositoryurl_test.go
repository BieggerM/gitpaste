package repositoryurl

import "testing"

func TestParseValid(t *testing.T) {
	t.Parallel()
	cases := []string{
		"git@github.com:owner/repo.git",
		"git@github.com:owner/repo",
		"https://github.com/owner/repo",
		"https://github.com/owner/repo.git",
		"git@gitlab.com:owner/repository.git",
		"https://gitlab.com/group/subgroup/repo.git",
		"https://gitlab.com/group_1/sub-group/repo.name",
	}
	for _, input := range cases {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			got, err := Parse(input)
			if err != nil {
				t.Fatalf("Parse(%q): %v", input, err)
			}
			if got.String() != input {
				t.Fatalf("Parse(%q) = %q", input, got)
			}
		})
	}
}

func TestParseInvalid(t *testing.T) {
	t.Parallel()
	cases := []string{
		"",
		"https://github.com/owner/repo;rm -rf /",
		"https://user:password@github.com/owner/repo.git",
		"https://github.com/owner/repo.git?x=1",
		"https://github.com/owner/repo.git#fragment",
		"git@evil.example:owner/repo.git",
		"git@github.com:owner/repo.git extra-command",
		"http://github.com/owner/repo.git",
		"ssh://git@github.com/owner/repo.git",
		"https://github.com/owner",
		"https://github.com/owner/repo/extra",
		"https://gitlab.com/group",
		"https://gitlab.com/group//repo",
		"https://github.com/owner/..",
		"https://github.com/owner/.git",
		"https://github.com/_/repo",
		"https://github.com/-owner/repo",
		"https://github.com/owner-/repo",
		"https://github.com/own--er/repo",
		"https://gitlab.com/group_/repo",
		"https://gitlab.com/group/_repo",
		"https://gitlab.com/group/repo..name",
		"https://gitlab.com/group/repo.atom",
		"https://github.com:443/owner/repo",
		"https://GITHUB.com/owner/repo",
		"https://github.com/owner/repo%20name",
		"https://github.com/owner/repo&whoami",
		"https://github.com/owner/repo|whoami",
		"https://github.com/owner/repo$(whoami)",
		"--upload-pack=evil",
		" github.com/owner/repo",
		"https://github.com/owner/repo\n",
	}
	for _, input := range cases {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			if got, err := Parse(input); err == nil {
				t.Fatalf("Parse(%q) unexpectedly succeeded: %q", input, got)
			}
		})
	}
}
