package relay

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"ansichat/internal/config"
	"ansichat/internal/model"
)

func historyTestServer(path string, limit int) *Server {
	return &Server{cfg: &config.Config{Relay: config.RelayConfig{HistoryFile: path}}, state: model.New(limit)}
}

func TestHistoryRestartRetainsIdentityAndBounds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "history.json")
	s := historyTestServer(path, 3)
	for i := 0; i < 5; i++ {
		s.state.Add(model.Message{RelayID: newMessageID(), Time: time.Unix(123, 0), Server: "test", Target: "#go", Nick: "alice", Text: "identical", Kind: model.KindMessage})
	}
	if err := s.saveHistory(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permissions %o", info.Mode().Perm())
	}
	restarted := historyTestServer(path, 2)
	if err := restarted.restoreHistory(); err != nil {
		t.Fatal(err)
	}
	messages := restarted.replayMessages()
	if len(messages) != 2 {
		t.Fatalf("retained %d", len(messages))
	}
	if messages[0].RelayID == messages[1].RelayID || !messages[0].Replay {
		t.Fatal("lost stable identity or replay marker")
	}
	client := model.New(10)
	client.Add(messages[0])
	if !client.ContainsMessage(messages[0]) || client.ContainsMessage(messages[1]) {
		t.Fatal("identical text was incorrectly deduplicated")
	}
	if err := restarted.saveHistory(); err != nil {
		t.Fatal(err)
	}
	again := historyTestServer(path, 10)
	if err := again.restoreHistory(); err != nil {
		t.Fatal(err)
	}
	if again.replayMessages()[1].RelayID != messages[1].RelayID {
		t.Fatal("ID changed across second restart")
	}
}

func TestHistoryCorruptionDoesNotChangeState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	bad := []byte(`{"Version":1,"Messages":[{"Server":"test","Target":"#go"}]}`)
	if err := os.WriteFile(path, bad, 0o600); err != nil {
		t.Fatal(err)
	}
	s := historyTestServer(path, 10)
	if err := s.restoreHistory(); err == nil {
		t.Fatal("accepted missing message ID")
	}
	if len(s.replayMessages()) != 0 {
		t.Fatal("partially restored corrupt history")
	}
	got, _ := os.ReadFile(path)
	if string(got) != string(bad) {
		t.Fatal("overwrote corrupt file")
	}
}

func TestDisabledHistoryDoesNotCreateFiles(t *testing.T) {
	s := historyTestServer("", 10)
	if err := s.restoreHistory(); err != nil {
		t.Fatal(err)
	}
	if err := s.saveHistory(); err != nil {
		t.Fatal(err)
	}
}

func TestStreamHistoryFormat(t *testing.T) {
	for _, messages := range [][]model.Message{nil, {{RelayID: "id", Server: "test", Target: "#go", Text: "quote \" newline\n <tag>", Tags: map[string]string{"key": "value"}}}} {
		var out bytes.Buffer
		if err := writeHistory(&out, []*model.Buffer{{Messages: messages}, {}}); err != nil {
			t.Fatal(err)
		}
		var got historySnapshot
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Version != 1 || len(got.Messages) != len(messages) {
			t.Fatalf("unexpected history: %#v", got)
		}
		if len(messages) > 0 && !reflect.DeepEqual(got.Messages, messages) {
			t.Fatal("message changed")
		}
	}
}

func BenchmarkHistoryEncoding(b *testing.B) {
	buffers := make([]*model.Buffer, 20)
	for i := range buffers {
		buffers[i] = &model.Buffer{Messages: make([]model.Message, 1000)}
		for j := range buffers[i].Messages {
			buffers[i].Messages[j] = model.Message{RelayID: "0123456789abcdef0123456789abcdef", Server: "test", Target: "#go", Text: strings.Repeat("message ", 20), Kind: model.KindMessage}
		}
	}
	b.Run("previous", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			data := historySnapshot{Version: 1}
			for _, buffer := range buffers {
				data.Messages = append(data.Messages, buffer.Messages...)
			}
			if err := json.NewEncoder(io.Discard).Encode(data); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("streamed", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if err := writeHistory(io.Discard, buffers); err != nil {
				b.Fatal(err)
			}
		}
	})
}
