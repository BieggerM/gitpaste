// Package shell installs and removes gitpaste command-not-found hooks.
package shell

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

//go:embed assets/gitpaste.bash
var bashScript []byte

//go:embed assets/gitpaste.zsh
var zshScript []byte

type shellConfig struct {
	name   string
	rcName string
	script []byte
}

var supportedShells = []shellConfig{
	{name: "bash", rcName: ".bashrc", script: bashScript},
	{name: "zsh", rcName: ".zshrc", script: zshScript},
}

// Manager installs hook scripts below ConfigDir and edits rc files below HomeDir.
type Manager struct {
	HomeDir   string
	ConfigDir string
	Shell     string
}

// NewManager discovers the current user's paths from the environment.
func NewManager() (Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Manager{}, fmt.Errorf("find home directory: %w", err)
	}
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		config = filepath.Join(home, ".config")
	} else if !filepath.IsAbs(config) {
		return Manager{}, errors.New("XDG_CONFIG_HOME must be an absolute path")
	}
	return Manager{HomeDir: home, ConfigDir: filepath.Join(config, "gitpaste"), Shell: os.Getenv("SHELL")}, nil
}

// Install writes scripts and adds marker-bounded source blocks. Existing rc
// files are configured; if neither exists, the current shell's rc is created.
func (m Manager) Install() ([]string, error) {
	if err := m.validatePaths(); err != nil {
		return nil, err
	}
	if err := ensurePrivateDirectory(m.ConfigDir); err != nil {
		return nil, fmt.Errorf("create config directory: %w", err)
	}

	targets := m.targets()
	if len(targets) == 0 {
		return nil, errors.New("could not select Bash or Zsh; create ~/.bashrc or ~/.zshrc, or set SHELL")
	}
	var changed []string
	for _, shell := range targets {
		scriptPath := filepath.Join(m.ConfigDir, "gitpaste."+shell.name)
		scriptChanged, err := installHookFile(scriptPath, shell.script)
		if err != nil {
			return changed, fmt.Errorf("install %s hook: %w", shell.name, err)
		}
		if scriptChanged {
			changed = append(changed, scriptPath)
		}
		rcPath := filepath.Join(m.HomeDir, shell.rcName)
		updated, didChange, err := addBlock(rcPath, block(shell.name, scriptPath))
		if err != nil {
			return changed, err
		}
		if didChange {
			if err := atomicWriteFile(rcPath, updated, 0o600, true); err != nil {
				return changed, fmt.Errorf("update %s: %w", rcPath, err)
			}
			changed = append(changed, rcPath)
		}
	}
	return changed, nil
}

// Uninstall removes only gitpaste marker blocks and installed hook scripts.
func (m Manager) Uninstall() ([]string, error) {
	if err := m.validatePaths(); err != nil {
		return nil, err
	}
	if err := inspectConfigDirectory(m.ConfigDir); err != nil {
		return nil, fmt.Errorf("inspect config directory: %w", err)
	}
	var changed []string
	for _, shell := range supportedShells {
		rcPath := filepath.Join(m.HomeDir, shell.rcName)
		contents, err := os.ReadFile(rcPath)
		if err == nil {
			updated, didChange, removeErr := removeBlock(contents, shell.name)
			if removeErr != nil {
				return changed, fmt.Errorf("update %s: %w", rcPath, removeErr)
			}
			if didChange {
				if err := atomicWriteFile(rcPath, updated, 0o600, true); err != nil {
					return changed, fmt.Errorf("update %s: %w", rcPath, err)
				}
				changed = append(changed, rcPath)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return changed, fmt.Errorf("read %s: %w", rcPath, err)
		}
		if err := removeHookFile(filepath.Join(m.ConfigDir, "gitpaste."+shell.name)); err != nil {
			return changed, fmt.Errorf("remove %s hook: %w", shell.name, err)
		}
	}
	if err := os.Remove(m.ConfigDir); err != nil &&
		!errors.Is(err, os.ErrNotExist) &&
		!errors.Is(err, syscall.ENOTEMPTY) &&
		!errors.Is(err, syscall.EEXIST) {
		return changed, fmt.Errorf("remove config directory: %w", err)
	}
	return changed, nil
}

func (m Manager) validatePaths() error {
	if m.HomeDir == "" || m.ConfigDir == "" {
		return errors.New("home and config directories must be configured")
	}
	if !filepath.IsAbs(m.HomeDir) || !filepath.IsAbs(m.ConfigDir) {
		return errors.New("home and config directories must be absolute paths")
	}
	return nil
}

func ensurePrivateDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	if err := inspectConfigDirectory(path); err != nil {
		return err
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return fmt.Errorf("set config directory permissions: %w", err)
	}
	return nil
}

func inspectConfigDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("config path must be a real directory, not a symlink or other file")
	}
	return nil
}

func installHookFile(path string, contents []byte) (bool, error) {
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return false, errors.New("hook path must be a regular file, not a symlink or other file")
		}
		existing, readErr := os.ReadFile(path)
		if readErr != nil {
			return false, readErr
		}
		if bytes.Equal(existing, contents) {
			if info.Mode().Perm() == 0o600 {
				return false, nil
			}
			if err := os.Chmod(path, 0o600); err != nil {
				return false, err
			}
			return true, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err := atomicWriteFile(path, contents, 0o600, false); err != nil {
		return false, err
	}
	return true, nil
}

func removeHookFile(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("hook path is not a regular file; refusing to remove it")
	}
	return os.Remove(path)
}

// atomicWriteFile replaces path only after the complete new contents have
// reached disk. Symlinked rc files are resolved deliberately so dotfile-manager
// links remain intact; hook assets never permit symlinks.
func atomicWriteFile(path string, contents []byte, defaultMode os.FileMode, allowSymlink bool) error {
	resolved := path
	mode := defaultMode
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			if !allowSymlink {
				return errors.New("refusing to replace a symlink")
			}
			resolved, err = filepath.EvalSymlinks(path)
			if err != nil {
				return fmt.Errorf("resolve symlink: %w", err)
			}
			info, err = os.Stat(resolved)
			if err != nil {
				return fmt.Errorf("inspect symlink target: %w", err)
			}
		}
		if !info.Mode().IsRegular() {
			return errors.New("destination must be a regular file")
		}
		mode = info.Mode().Perm()
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	directory := filepath.Dir(resolved)
	temporary, err := os.CreateTemp(directory, ".gitpaste-tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return fmt.Errorf("set temporary file permissions: %w", err)
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := os.Rename(temporaryPath, resolved); err != nil {
		return fmt.Errorf("replace destination: %w", err)
	}
	directoryHandle, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open destination directory: %w", err)
	}
	defer directoryHandle.Close()
	if err := directoryHandle.Sync(); err != nil {
		return fmt.Errorf("sync destination directory: %w", err)
	}
	return nil
}

func (m Manager) targets() []shellConfig {
	var targets []shellConfig
	for _, shell := range supportedShells {
		if _, err := os.Stat(filepath.Join(m.HomeDir, shell.rcName)); err == nil {
			targets = append(targets, shell)
		}
	}
	if len(targets) != 0 {
		return targets
	}
	current := filepath.Base(m.Shell)
	for _, shell := range supportedShells {
		if current == shell.name {
			return []shellConfig{shell}
		}
	}
	return nil
}

func markers(name string) (string, string) {
	return "# >>> gitpaste " + name + " shell hook >>>", "# <<< gitpaste " + name + " shell hook <<<"
}

func block(name, path string) []byte {
	start, end := markers(name)
	quoted := shellQuote(path)
	return []byte(start + "\n[ -r " + quoted + " ] && . " + quoted + "\n" + end + "\n")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func addBlock(path string, wanted []byte) ([]byte, bool, error) {
	contents, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, false, fmt.Errorf("read %s: %w", path, err)
	}
	name := "bash"
	if strings.Contains(string(wanted), "gitpaste zsh") {
		name = "zsh"
	}
	cleaned, found, err := removeBlock(contents, name)
	if err != nil {
		return nil, false, fmt.Errorf("inspect %s: %w", path, err)
	}
	if found && strings.Contains(string(contents), string(wanted)) {
		return contents, false, nil
	}
	if len(cleaned) > 0 && cleaned[len(cleaned)-1] != '\n' {
		cleaned = append(cleaned, '\n')
	}
	cleaned = append(cleaned, wanted...)
	return cleaned, true, nil
}

func removeBlock(contents []byte, name string) ([]byte, bool, error) {
	start, end := markers(name)
	lines := strings.SplitAfter(string(contents), "\n")
	startIndex, endIndex := -1, -1
	for i, line := range lines {
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		switch line {
		case start:
			if startIndex >= 0 {
				return nil, false, errors.New("found duplicate start markers")
			}
			startIndex = i
		case end:
			if endIndex >= 0 {
				return nil, false, errors.New("found duplicate end markers")
			}
			endIndex = i
		}
	}
	if startIndex < 0 && endIndex < 0 {
		return contents, false, nil
	}
	if startIndex < 0 {
		return nil, false, errors.New("found end marker without matching start marker")
	}
	if endIndex < 0 {
		return nil, false, errors.New("found start marker without matching end marker")
	}
	if endIndex < startIndex {
		return nil, false, errors.New("found shell hook markers in the wrong order")
	}
	updated := append([]string{}, lines[:startIndex]...)
	updated = append(updated, lines[endIndex+1:]...)
	return []byte(strings.Join(updated, "")), true, nil
}
