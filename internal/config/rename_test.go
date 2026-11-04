package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigNeverFallsBackToCopperline(t *testing.T) {
	for _, xdg := range []bool{false, true} {
		t.Run(map[bool]string{false: "home", true: "xdg"}[xdg], func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("XDG_CONFIG_HOME", "")
			if xdg {
				t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			}
			root, err := os.UserConfigDir()
			if err != nil {
				t.Fatal(err)
			}
			current := filepath.Join(root, "ansichat", "config.toml")
			legacy := filepath.Join(root, "copperline", "config.toml")
			if got := DefaultPath(); got != current {
				t.Fatalf("fresh path = %q, want %q", got, current)
			}

			// An old config may describe a completely different relay. Do not
			// use it unless the operator explicitly passes its path.
			valid := "[relay]\nmode = 'client'\naddress = 'relay.example:2222'\nhost_key_fingerprint = 'SHA256:test'\n"
			writeRenameFixture(t, legacy, valid)
			if got := DefaultPath(); got != current {
				t.Fatalf("legacy config changed default: %q", got)
			}
			if _, err := Load(""); !os.IsNotExist(err) {
				t.Fatalf("missing AnsiChat config should fail, got %v", err)
			}
			if _, err := Load(legacy); err != nil {
				t.Fatalf("explicit legacy config must still work: %v", err)
			}
			writeRenameFixture(t, current, valid)
			if _, err := Load(""); err != nil {
				t.Fatalf("load AnsiChat config: %v", err)
			}
			writeRenameFixture(t, current, "invalid = [")
			if _, err := Load(""); err == nil {
				t.Fatal("invalid AnsiChat config unexpectedly succeeded")
			}
		})
	}
}

func TestDefaultRelayFilesNeverFallBackToCopperline(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	legacyDir := filepath.Join(root, "copperline")
	newDir := filepath.Join(root, "ansichat")
	for _, filename := range []string{
		"relay_host_ed25519",
		"relay_client_ed25519",
		"relay_authorized_keys",
		"ignores.toml",
		"scripts/hello.lua",
	} {
		writeRenameFixture(t, filepath.Join(legacyDir, filename), "existing legacy data")
	}
	writeRenameFixture(t, filepath.Join(home, ".local", "state", "copperline", "logs", "channel.log"), "legacy logs")

	cfg := Config{}
	cfg.applyDefaults()
	for name, tc := range map[string]struct{ got, want string }{
		"host key":       {cfg.Relay.HostKey, filepath.Join(newDir, "relay_host_ed25519")},
		"client key":     {cfg.Relay.PrivateKey, filepath.Join(newDir, "relay_client_ed25519")},
		"authorized keys": {cfg.Relay.AuthorizedKeys, filepath.Join(newDir, "relay_authorized_keys")},
		"ignore file":    {cfg.General.IgnoreFile, filepath.Join(newDir, "ignores.toml")},
		"scripts":        {cfg.Scripting.Dir, filepath.Join(newDir, "scripts")},
		"logs":           {cfg.General.LogDir, "~/.local/state/ansichat/logs"},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", name, tc.got, tc.want)
		}
	}
	// User-specified paths, including intentional legacy paths, are respected.
	cfg = Config{Relay: RelayConfig{HostKey: filepath.Join(legacyDir, "relay_host_ed25519"), AuthorizedKeys: filepath.Join(legacyDir, "relay_authorized_keys")}}
	cfg.applyDefaults()
	if cfg.Relay.HostKey != filepath.Join(legacyDir, "relay_host_ed25519") ||
		cfg.Relay.AuthorizedKeys != filepath.Join(legacyDir, "relay_authorized_keys") {
		t.Fatal("explicit relay paths were overridden")
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
