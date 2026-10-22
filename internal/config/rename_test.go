package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenamedDefaultConfigPath(t *testing.T) {
	for _, xdg := range []bool{false, true} {
		t.Run(map[bool]string{false: "home", true: "xdg"}[xdg], func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("XDG_CONFIG_HOME", "")
			base := filepath.Join(home, ".config")
			if xdg {
				base = t.TempDir()
				t.Setenv("XDG_CONFIG_HOME", base)
			}
			current := filepath.Join(base, "ansichat", "config.toml")
			legacy := filepath.Join(base, "copperline", "config.toml")
			if got := DefaultPath(); got != current {
				t.Fatalf("fresh path = %q, want %q", got, current)
			}
			writeRenameFixture(t, legacy, "[relay]\nmode = 'client'\naddress = 'relay.example:2222'\nhost_key_fingerprint = 'SHA256:test'\n")
			if got := DefaultPath(); got != legacy {
				t.Fatalf("upgrade path = %q, want %q", got, legacy)
			}
			if _, err := Load(""); err != nil {
				t.Fatalf("load legacy configuration: %v", err)
			}
			// An invalid new config must report its error, never use the old one.
			writeRenameFixture(t, current, "invalid = [")
			if got := DefaultPath(); got != current {
				t.Fatalf("new config did not take precedence: %q", got)
			}
			if _, err := Load(""); err == nil {
				t.Fatal("invalid AnsiChat config silently fell back to Copperline")
			}
		})
	}
}

func TestRenameReusesExistingDefaultDataAndKeys(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	paths := []string{
		".config/copperline/relay_host_ed25519",
		".config/copperline/relay_client_ed25519",
		".config/copperline/relay_authorized_keys",
		".config/copperline/ignores.toml",
		".config/copperline/scripts/hello.lua",
		".local/state/copperline/logs/channel.log",
	}
	for _, path := range paths {
		writeRenameFixture(t, filepath.Join(home, filepath.FromSlash(path)), "existing data")
	}
	cfg := Config{}
	cfg.applyDefaults()
	for name, path := range map[string]string{
		"host key": cfg.Relay.HostKey, "client key": cfg.Relay.PrivateKey,
		"authorized keys": cfg.Relay.AuthorizedKeys, "ignore file": cfg.General.IgnoreFile,
		"scripts": cfg.Scripting.Dir, "logs": cfg.General.LogDir,
	} {
		if _, err := os.Stat(ExpandPath(path)); err != nil {
			t.Errorf("%s no longer points to existing data: %q (%v)", name, path, err)
		}
	}
	// New keys take precedence; the legacy bytes are never moved or overwritten.
	newKey := filepath.Join(home, ".config", "ansichat", "relay_client_ed25519")
	writeRenameFixture(t, newKey, "new key")
	cfg = Config{}
	cfg.applyDefaults()
	if got := ExpandPath(cfg.Relay.PrivateKey); got != newKey {
		t.Fatalf("new private key = %q, want %q", got, newKey)
	}
	oldKey := filepath.Join(home, ".config", "copperline", "relay_client_ed25519")
	if data, err := os.ReadFile(oldKey); err != nil || string(data) != "existing data" {
		t.Fatalf("legacy key modified: %q, %v", data, err)
	}
}

func writeRenameFixture(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}
