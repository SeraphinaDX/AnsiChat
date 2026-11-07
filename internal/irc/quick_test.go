package irc

import (
	"ansichat/internal/config"
	"testing"
)

func TestQuickConfig(t *testing.T) {
	general := config.GeneralConfig{Nick: "me", User: "myuser", RealName: "My Name"}
	for _, tc := range []struct {
		args, host, nick string
		port             int
		tls              bool
	}{
		{"irc.example.org", "irc.example.org", "me", 6697, true},
		{"-plain irc.example.org", "irc.example.org", "me", 6667, false},
		{"irc.example.org 7000 other -tls", "irc.example.org", "other", 7000, true},
		{"IRC.example.org:7000 other", "irc.example.org", "other", 7000, true},
		{"[::1]:7000", "::1", "me", 7000, true},
		{"::1", "::1", "me", 6697, true},
	} {
		t.Run(tc.args, func(t *testing.T) {
			sc, err := quickConfig(tc.args, general)
			if err != nil {
				t.Fatal(err)
			}
			if sc.Host != tc.host || sc.Port != tc.port || sc.TLS != tc.tls || sc.Nick != tc.nick || sc.User != general.User || sc.RealName != general.RealName {
				t.Fatalf("unexpected config: %#v", sc)
			}
			if sc.Password != "" || sc.SASL.Mechanism != "" || sc.TLSCertFile != "" || len(sc.Channels) != 0 || sc.SkipVerify {
				t.Fatalf("unexpected network settings: %#v", sc)
			}
		})
	}
	for _, args := range []string{"", "host 0", "host 65536", "host abc", "host 6697 123bad", "host 6697 nick extra", "host\nJOIN #bad", "irc://host", "-bad", "host -plain -tls", "host:bad", "host:", "host:6697 nick extra"} {
		if _, err := quickConfig(args, general); err == nil {
			t.Errorf("accepted invalid args %q", args)
		}
	}
}

func TestQuickDoesNotReuseConfiguredSession(t *testing.T) {
	cfg := &config.Config{General: config.GeneralConfig{Nick: "me"}, Servers: []config.ServerConfig{{Name: "host:6697/tls", Host: "host", Port: 6697, TLS: true, Nick: "me", User: "me", Password: "secret"}}}
	m := New(cfg, nil)
	defer m.Stop("test")
	if _, err := m.QuickConnect("host"); err == nil {
		t.Fatal("quick reused a configured session with credentials")
	}
	if len(cfg.Servers) != 1 || len(m.ServerNames()) != 1 {
		t.Fatal("quick changed configured servers")
	}
}
