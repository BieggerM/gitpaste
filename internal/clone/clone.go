// Package clone confirms and executes safe git clone operations.
package clone

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/bieggerm/gitpaste/internal/repositoryurl"
)

// Options controls a clone operation.
type Options struct {
	Yes    bool
	Input  io.Reader
	Output io.Writer
	Error  io.Writer
}

// Run validates url, confirms the operation, and invokes Git. It returns the
// desired process exit code and an optional user-facing error.
func Run(ctx context.Context, rawURL string, options Options) (int, error) {
	repository, err := repositoryurl.Parse(rawURL)
	if err != nil {
		return 2, fmt.Errorf("invalid repository URL: %w", err)
	}
	if options.Input == nil || options.Output == nil || options.Error == nil {
		return 1, errors.New("internal error: clone streams are not configured")
	}

	if !options.Yes {
		confirmed, err := confirm(options.Input, options.Error, repository.String())
		if err != nil {
			return 1, err
		}
		if !confirmed {
			_, _ = fmt.Fprintln(options.Error, "gitpaste: clone cancelled")
			return 0, nil
		}
	}

	// The option terminator prevents even a future validator regression from
	// turning the repository value into a Git command-line option.
	command := exec.CommandContext(ctx, "git", "clone", "--", repository.String())
	command.Stdin = options.Input
	command.Stdout = options.Output
	command.Stderr = options.Error
	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return exitError.ExitCode(), nil
		}
		var execError *exec.Error
		if errors.As(err, &execError) {
			return 127, fmt.Errorf("could not start git: %w", err)
		}
		return 1, fmt.Errorf("git clone failed: %w", err)
	}
	return 0, nil
}

func confirm(input io.Reader, output io.Writer, repository string) (bool, error) {
	reader := bufio.NewReader(input)
	for {
		if _, err := fmt.Fprintf(output, "gitpaste: Clone %s? [Y/n] ", repository); err != nil {
			return false, fmt.Errorf("write confirmation prompt: %w", err)
		}
		answer, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return false, fmt.Errorf("read confirmation: %w", err)
		}
		if errors.Is(err, io.EOF) && answer == "" {
			return false, errors.New("confirmation input ended before an answer")
		}
		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "", "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			if errors.Is(err, io.EOF) {
				return false, errors.New("confirmation input ended before a valid answer")
			}
			_, _ = fmt.Fprintln(output, "gitpaste: please answer yes or no")
		}
	}
}
