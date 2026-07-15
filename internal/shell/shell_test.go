package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallAndUninstall(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, "config", "gitpaste")
	bashrc := filepath.Join(home, ".bashrc")
	zshrc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(bashrc, []byte("# existing bash config\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(zshrc, []byte("# existing zsh config\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := Manager{HomeDir: home, ConfigDir: config}
	changed, err := m.Install()
	if err != nil || len(changed) != 4 {
		t.Fatalf("Install() = (%v, %v)", changed, err)
	}
	if changed, err = m.Install(); err != nil || len(changed) != 0 {
		t.Fatalf("second Install() = (%v, %v)", changed, err)
	}
	if err := os.WriteFile(filepath.Join(config, "gitpaste.bash"), []byte("outdated"), 0o600); err != nil {
		t.Fatal(err)
	}
	if changed, err = m.Install(); err != nil || len(changed) != 1 || changed[0] != filepath.Join(config, "gitpaste.bash") {
		t.Fatalf("upgrade Install() = (%v, %v)", changed, err)
	}
	for _, name := range []string{"bash", "zsh"} {
		if _, err := os.Stat(filepath.Join(config, "gitpaste."+name)); err != nil {
			t.Fatal(err)
		}
	}
	changed, err = m.Uninstall()
	if err != nil || len(changed) != 2 {
		t.Fatalf("Uninstall() = (%v, %v)", changed, err)
	}
	for _, path := range []string{bashrc, zshrc} {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(got), "gitpaste") || !strings.Contains(string(got), "existing") {
			t.Fatalf("unexpected rc contents: %q", got)
		}
	}
}

func TestInstallQuotesConfigPath(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, "config's dir")
	m := Manager{HomeDir: home, ConfigDir: config, Shell: "/bin/bash"}
	if _, err := m.Install(); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(home, ".bashrc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), `config'\''s dir/gitpaste.bash'`) {
		t.Fatalf("path was not safely quoted: %q", contents)
	}
}

func TestUninstallRejectsBrokenMarker(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("# >>> gitpaste bash shell hook >>>\nkeep me\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := Manager{HomeDir: home, ConfigDir: filepath.Join(home, ".config", "gitpaste")}
	if _, err := m.Uninstall(); err == nil {
		t.Fatal("Uninstall() unexpectedly accepted a broken marker block")
	}
}

func TestNewManagerRejectsRelativeXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "relative/config")
	if _, err := NewManager(); err == nil || !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("NewManager() error = %v", err)
	}
}

func TestInstallPreservesSymlinkedRCFile(t *testing.T) {
	home := t.TempDir()
	dotfiles := t.TempDir()
	target := filepath.Join(dotfiles, "zshrc")
	if err := os.WriteFile(target, []byte("# managed elsewhere\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, ".zshrc")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	m := Manager{HomeDir: home, ConfigDir: filepath.Join(home, ".config", "gitpaste")}
	if _, err := m.Install(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("rc symlink was not preserved: info=%v err=%v", info, err)
	}
	contents, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "gitpaste zsh shell hook") {
		t.Fatalf("symlink target was not updated: %q", contents)
	}
	if info, err := os.Stat(target); err != nil || info.Mode().Perm() != 0o640 {
		t.Fatalf("rc mode was not preserved: info=%v err=%v", info, err)
	}
}

func TestInstallRefusesSymlinkedHookFile(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, ".config", "gitpaste")
	if err := os.MkdirAll(config, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	victim := filepath.Join(home, "victim")
	if err := os.WriteFile(victim, []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, filepath.Join(config, "gitpaste.bash")); err != nil {
		t.Fatal(err)
	}
	m := Manager{HomeDir: home, ConfigDir: config}
	if _, err := m.Install(); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("Install() error = %v", err)
	}
	contents, err := os.ReadFile(victim)
	if err != nil || string(contents) != "keep me" {
		t.Fatalf("symlink target changed: %q, %v", contents, err)
	}
}

func TestUninstallRefusesSymlinkedConfigDirectory(t *testing.T) {
	home := t.TempDir()
	target := t.TempDir()
	config := filepath.Join(home, "gitpaste-config")
	if err := os.Symlink(target, config); err != nil {
		t.Fatal(err)
	}
	m := Manager{HomeDir: home, ConfigDir: config}
	if _, err := m.Uninstall(); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("Uninstall() error = %v", err)
	}
}

func TestRemoveBlockIgnoresMarkerSubstring(t *testing.T) {
	contents := []byte("alias note='# >>> gitpaste bash shell hook >>>'\n")
	got, changed, err := removeBlock(contents, "bash")
	if err != nil || changed || string(got) != string(contents) {
		t.Fatalf("removeBlock() = (%q, %v, %v)", got, changed, err)
	}
}

func TestDistributedScriptsMatchEmbeddedAssets(t *testing.T) {
	tests := []struct {
		path string
		want []byte
	}{{"../../shell/gitpaste.bash", bashScript}, {"../../shell/gitpaste.zsh", zshScript}}
	for _, test := range tests {
		got, err := os.ReadFile(test.path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(test.want) {
			t.Fatalf("%s differs from embedded asset", test.path)
		}
	}
}
