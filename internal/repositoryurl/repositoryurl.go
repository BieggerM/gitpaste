// Package repositoryurl validates repository URLs accepted by gitpaste.
package repositoryurl

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	httpsPattern = regexp.MustCompile(`^https://(github\.com|gitlab\.com)/([A-Za-z0-9._-]+(?:/[A-Za-z0-9._-]+)+)$`)
	sshPattern   = regexp.MustCompile(`^git@(github\.com|gitlab\.com):([A-Za-z0-9._-]+(?:/[A-Za-z0-9._-]+)+)$`)
)

// URL is a repository URL that passed the strict validator.
type URL string

func (u URL) String() string { return string(u) }

// Parse accepts only GitHub and GitLab HTTPS or SCP-style SSH repository URLs.
func Parse(raw string) (URL, error) {
	if raw == "" {
		return "", errors.New("repository URL is empty")
	}
	if strings.TrimSpace(raw) != raw || strings.IndexFunc(raw, func(r rune) bool {
		return r <= ' ' || r == 0x7f
	}) >= 0 {
		return "", errors.New("repository URL must not contain whitespace or control characters")
	}

	if match := httpsPattern.FindStringSubmatch(raw); match != nil {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", fmt.Errorf("invalid HTTPS repository URL: %w", err)
		}
		if parsed.Scheme != "https" || parsed.User != nil || parsed.Host != match[1] || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.RawPath != "" {
			return "", errors.New("HTTPS repository URL contains unsupported components")
		}
		if err := validatePath(match[1], match[2]); err != nil {
			return "", err
		}
		return URL(raw), nil
	}

	if match := sshPattern.FindStringSubmatch(raw); match != nil {
		if err := validatePath(match[1], match[2]); err != nil {
			return "", err
		}
		return URL(raw), nil
	}

	return "", errors.New("unsupported repository URL; expected GitHub or GitLab HTTPS/SSH format")
}

func validatePath(host, path string) error {
	segments := strings.Split(path, "/")
	if host == "github.com" && len(segments) != 2 {
		return errors.New("GitHub repository URL must contain exactly an owner and repository")
	}
	if host == "gitlab.com" && len(segments) < 2 {
		return errors.New("GitLab repository URL must contain a namespace and repository")
	}
	if host == "github.com" && !validGitHubOwner(segments[0]) {
		return errors.New("GitHub owner must be 1-39 alphanumeric or single-hyphen characters and cannot start or end with a hyphen")
	}
	for i, segment := range segments {
		name := segment
		if i == len(segments)-1 {
			name = strings.TrimSuffix(name, ".git")
		}
		if name == "" || name == "." || name == ".." || strings.HasSuffix(name, ".git") {
			return errors.New("repository URL contains an invalid path segment")
		}
		if host == "gitlab.com" && !validGitLabSlug(name) {
			return errors.New("GitLab path segments must start and end with an alphanumeric character and cannot contain consecutive special characters or end in .atom")
		}
	}
	return nil
}

func validGitHubOwner(owner string) bool {
	if len(owner) == 0 || len(owner) > 39 || !isASCIIAlphanumeric(owner[0]) || !isASCIIAlphanumeric(owner[len(owner)-1]) {
		return false
	}
	previousHyphen := false
	for i := 0; i < len(owner); i++ {
		if isASCIIAlphanumeric(owner[i]) {
			previousHyphen = false
			continue
		}
		if owner[i] != '-' || previousHyphen {
			return false
		}
		previousHyphen = true
	}
	return true
}

func validGitLabSlug(slug string) bool {
	if !isASCIIAlphanumeric(slug[0]) || !isASCIIAlphanumeric(slug[len(slug)-1]) || strings.HasSuffix(slug, ".atom") {
		return false
	}
	previousSpecial := false
	for i := 0; i < len(slug); i++ {
		special := !isASCIIAlphanumeric(slug[i])
		if special && previousSpecial {
			return false
		}
		previousSpecial = special
	}
	return true
}

func isASCIIAlphanumeric(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9'
}

// Validate reports whether raw is an accepted repository URL.
func Validate(raw string) error {
	_, err := Parse(raw)
	return err
}
