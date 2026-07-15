package main

import (
	"context"
	"fmt"
	"io"
	"os"

	clonepkg "github.com/bieggerm/gitpaste/internal/clone"
	"github.com/bieggerm/gitpaste/internal/repositoryurl"
	"github.com/bieggerm/gitpaste/internal/shell"
)

var version = "dev"

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}

	switch args[0] {
	case "clone":
		rawURL, yes, err := parseCloneArgs(args[1:])
		if err != nil {
			fmt.Fprintf(stderr, "gitpaste: %v\n", err)
			fmt.Fprintln(stderr, "usage: gitpaste clone [--yes] [--] <repository-url>")
			return 2
		}
		code, err := clonepkg.Run(ctx, rawURL, clonepkg.Options{Yes: yes, Input: stdin, Output: stdout, Error: stderr})
		if err != nil {
			fmt.Fprintf(stderr, "gitpaste: %v\n", err)
		}
		return code

	case "validate":
		rawURL, err := singleURLArg(args[1:])
		if err != nil {
			fmt.Fprintf(stderr, "gitpaste: %v\n", err)
			fmt.Fprintln(stderr, "usage: gitpaste validate [--] <repository-url>")
			return 2
		}
		if err := repositoryurl.Validate(rawURL); err != nil {
			fmt.Fprintf(stderr, "gitpaste: invalid repository URL: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "valid repository URL: %s\n", rawURL)
		return 0

	case "setup", "install-shell-hook":
		if len(args) != 1 {
			fmt.Fprintf(stderr, "gitpaste: %s does not accept arguments\n", args[0])
			return 2
		}
		manager, err := shell.NewManager()
		if err != nil {
			fmt.Fprintf(stderr, "gitpaste: %v\n", err)
			return 1
		}
		changed, err := manager.Install()
		if err != nil {
			fmt.Fprintf(stderr, "gitpaste: install shell hook: %v\n", err)
			return 1
		}
		if len(changed) == 0 {
			fmt.Fprintln(stdout, "gitpaste: shell hooks are already installed")
		} else {
			for _, path := range changed {
				fmt.Fprintf(stdout, "gitpaste: updated %s\n", path)
			}
			fmt.Fprintln(stdout, "gitpaste: restart your shell or source the updated rc file")
		}
		return 0

	case "uninstall-shell-hook":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "gitpaste: uninstall-shell-hook does not accept arguments")
			return 2
		}
		manager, err := shell.NewManager()
		if err != nil {
			fmt.Fprintf(stderr, "gitpaste: %v\n", err)
			return 1
		}
		changed, err := manager.Uninstall()
		if err != nil {
			fmt.Fprintf(stderr, "gitpaste: uninstall shell hook: %v\n", err)
			return 1
		}
		if len(changed) == 0 {
			fmt.Fprintln(stdout, "gitpaste: no shell hook markers found")
		} else {
			for _, path := range changed {
				fmt.Fprintf(stdout, "gitpaste: updated %s\n", path)
			}
		}
		return 0

	case "version":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "gitpaste: version does not accept arguments")
			return 2
		}
		fmt.Fprintf(stdout, "gitpaste %s\n", version)
		return 0

	case "help", "--help", "-h":
		usage(stdout)
		return 0

	default:
		fmt.Fprintf(stderr, "gitpaste: unknown command %q\n", args[0])
		usage(stderr)
		return 2
	}
}

func parseCloneArgs(args []string) (string, bool, error) {
	yes := false
	for len(args) > 0 {
		switch args[0] {
		case "--yes":
			if yes {
				return "", false, fmt.Errorf("--yes specified more than once")
			}
			yes = true
			args = args[1:]
		case "--":
			args = args[1:]
			if len(args) != 1 {
				return "", false, fmt.Errorf("clone requires exactly one repository URL")
			}
			return args[0], yes, nil
		default:
			if len(args) != 1 {
				return "", false, fmt.Errorf("clone requires exactly one repository URL")
			}
			return args[0], yes, nil
		}
	}
	return "", false, fmt.Errorf("clone requires exactly one repository URL")
}

func singleURLArg(args []string) (string, error) {
	if len(args) == 2 && args[0] == "--" {
		return args[1], nil
	}
	if len(args) != 1 {
		return "", fmt.Errorf("validate requires exactly one repository URL")
	}
	return args[0], nil
}

func usage(output io.Writer) {
	fmt.Fprintln(output, `usage:
  gitpaste clone [--yes] [--] <repository-url>
  gitpaste validate [--] <repository-url>
  gitpaste setup
  gitpaste install-shell-hook
  gitpaste uninstall-shell-hook
  gitpaste version`)
}
