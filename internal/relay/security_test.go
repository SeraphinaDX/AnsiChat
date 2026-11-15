package relay

import (
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ansichat/internal/config"
)

func TestRequestFrameLimit(t *testing.T) {
	for _, input := range []string{
		`{"type":"request","text":"` + strings.Repeat("x", maxRequestBytes) + `"}` + "\n",
		strings.Repeat(" ", maxRequestBytes) + "{}\n",
	} {
		var f frame
		if err := newRequestDecoder(strings.NewReader(input)).Decode(&f); err == nil {
			t.Fatal("accepted oversized frame")
		}
	}
	d := newRequestDecoder(strings.NewReader("{\"type\":\"hello\"}\n{\"type\":\"request\"}\n"))
	var f frame
	for _, want := range []string{"hello", "request"} {
		if err := d.Decode(&f); err != nil || f.Type != want {
			t.Fatalf("decode = %v, %#v", err, f)
		}
	}
	if err := d.Decode(&f); err != io.EOF {
		t.Fatalf("end = %v", err)
	}
}

func TestAuthorizedKeyOptionsFailClosed(t *testing.T) {
	dir := t.TempDir()
	_, _, pub, err := loadOrCreateClientSigner(filepath.Join(dir, "client"), "")
	if err != nil {
		t.Fatal(err)
	}
	key, err := os.ReadFile(pub)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "authorized")
	if err := os.WriteFile(path, append([]byte(`from="127.0.0.1" `), key...), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadAuthorizedKeys(path); err == nil {
		t.Fatal("silently ignored key restrictions")
	}
}

func TestRelayRejectsSensitiveActionsByDefault(t *testing.T) {
	for _, action := range []string{"dcc_send", "dcc_accept", "quick"} {
		p := &serverPeer{server: &Server{cfg: &config.Config{}}, sendQ: make(chan frame, 1), closed: make(chan struct{})}
		p.handleRequest(frame{ID: 1, Action: action, Text: "/etc/passwd"})
		if response := <-p.sendQ; response.Error == "" {
			t.Fatalf("%s accepted", action)
		}
	}
}

func TestSSHChannelLimitFrameLimitAndRevocation(t *testing.T) {
	dir := t.TempDir()
	signer, _, pubPath, err := loadOrCreateClientSigner(filepath.Join(dir, "client"), "")
	if err != nil {
		t.Fatal(err)
	}
	pub, err := os.ReadFile(pubPath)
	if err != nil {
		t.Fatal(err)
	}
	authorized := filepath.Join(dir, "authorized")
	if err := os.WriteFile(authorized, pub, 0600); err != nil {
		t.Fatal(err)
	}
	no := false
	cfg := &config.Config{General: config.GeneralConfig{Logging: &no}, Relay: config.RelayConfig{HostKey: filepath.Join(dir, "host"), AuthorizedKeys: authorized}}
	s, err := NewServer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go s.serveConn(c)
		}
	}()
	dial := func() *ssh.Client {
		t.Helper()
		c, err := ssh.Dial("tcp", ln.Addr().String(), &ssh.ClientConfig{User: "test", Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)}, HostKeyCallback: ssh.FixedHostKey(mustHostKey(t, cfg.Relay.HostKey)), Timeout: time.Second})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { c.Close() })
		return c
	}
	c := dial()
	ch, requests, err := c.OpenChannel(channelType, nil)
	if err != nil {
		t.Fatal(err)
	}
	go ssh.DiscardRequests(requests)
	if extra, _, err := c.OpenChannel(channelType, nil); err == nil {
		extra.Close()
		t.Fatal("accepted second relay channel")
	}
	// An oversized initial frame must close the transport, before a hello is accepted.
	_, _ = ch.Write([]byte(strings.Repeat("x", maxRequestBytes) + "\n"))
	closed := make(chan error, 1)
	go func() { closed <- c.Wait() }()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("oversized frame did not close transport")
	}
	c = dial()
	closed = make(chan error, 1)
	go func(client *ssh.Client) { closed <- client.Wait() }(c)
	if err := os.WriteFile(authorized, nil, 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-closed:
	case <-time.After(7 * time.Second):
		t.Fatal("revoked key retained its transport")
	}
}

func mustHostKey(t *testing.T, path string) ssh.PublicKey {
	t.Helper()
	signer, err := loadOrCreateHostSigner(path)
	if err != nil {
		t.Fatal(err)
	}
	return signer.PublicKey()
}
