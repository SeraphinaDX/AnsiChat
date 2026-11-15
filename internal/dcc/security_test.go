package dcc

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRejectInvalidSendSizes(t *testing.T) {
	m := New(t.TempDir(), "127.0.0.1:0", "127.0.0.1", nil)
	for _, size := range []string{"-1", "1073741825", "bad"} {
		if _, err := m.ParseOffer("test", "peer", "SEND test 2130706433 1234 "+size); err == nil {
			t.Fatalf("accepted size %s", size)
		}
	}
}

func TestReceiveEnforcesAdvertisedSize(t *testing.T) {
	for _, tc := range []struct {
		name, payload string
		size          int64
		wantFile      bool
	}{
		{"excess", "abcdef", 3, true}, {"short", "ab", 3, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer ln.Close()
			done := make(chan struct{})
			go func() {
				defer close(done)
				c, err := ln.Accept()
				if err != nil {
					return
				}
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(time.Second))
				_, _ = c.Write([]byte(tc.payload))
				_ = c.(*net.TCPConn).CloseWrite()
				var ack [4]byte
				_, _ = c.Read(ack[:])
			}()
			m := New(dir, "", "", nil)
			m.receiveFile(Offer{Filename: "test", Address: "127.0.0.1", Port: ln.Addr().(*net.TCPAddr).Port, Size: tc.size})
			<-done
			data, err := os.ReadFile(filepath.Join(dir, "test"))
			if tc.wantFile {
				if err != nil || string(data) != "abc" {
					t.Fatalf("file = %q, %v", data, err)
				}
			} else if !os.IsNotExist(err) {
				t.Fatalf("partial file remains: %v", err)
			}
		})
	}
}

func TestOffersAndTransferConcurrencyAreBounded(t *testing.T) {
	m := New(t.TempDir(), "", "", nil)
	for i := 0; i < 128; i++ {
		if _, err := m.ParseOffer("test", fmt.Sprint(i), "SEND test 2130706433 1234 1"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := m.ParseOffer("test", "overflow", "SEND test 2130706433 1234 1"); err == nil {
		t.Fatal("offer limit ignored")
	}
	for i := 0; i < 4; i++ {
		if !m.acquire() {
			t.Fatal("slot unavailable")
		}
	}
	if err := m.SendFile("test", "peer", "ignored", func(string) {}); err == nil || !strings.Contains(err.Error(), "too many") {
		t.Fatalf("transfer cap = %v", err)
	}
	for i := 0; i < 4; i++ {
		m.release()
	}
}

func TestSendUsesFileOpenedBeforeOffer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "send")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	payload := ""
	m := New(dir, "127.0.0.1:0", "127.0.0.1", nil)
	if err := m.SendFile("test", "peer", path, func(p string) { payload = p }); err != nil {
		t.Fatal(err)
	}
	// Replace the directory entry while the sender is waiting for its peer.
	if err := os.Rename(path, path+".old"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("secret replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	offer, err := m.ParseOffer("test", "peer", payload)
	if err != nil {
		t.Fatal(err)
	}
	c, err := net.DialTimeout("tcp", net.JoinHostPort(offer.Address, strconv.Itoa(offer.Port)), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(time.Second))
	data, err := io.ReadAll(c)
	if err != nil || string(data) != "original" {
		t.Fatalf("sent %q, error %v", data, err)
	}
}
