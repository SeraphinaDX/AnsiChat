package relay

import (
	"encoding/json"
	"strings"
	"testing"

	ircclient "ansichat/internal/irc"
)

func TestCloneSnapshotIsIndependent(t *testing.T) {
	in := ircclient.StateSnapshot{Servers: []ircclient.ServerSnapshot{{
		Name:         "test",
		Capabilities: []string{"message-tags"},
		Targets: []ircclient.TargetSnapshot{{
			Target: "#test",
			Users:  []ircclient.UserSnapshot{{Nick: "alice", Prefix: "@"}},
		}},
	}}}
	out := cloneSnapshot(in)
	out.Servers[0].Capabilities[0] = "changed"
	out.Servers[0].Targets[0].Users[0].Nick = "bob"
	if in.Servers[0].Capabilities[0] != "message-tags" || in.Servers[0].Targets[0].Users[0].Nick != "alice" {
		t.Fatal("cloneSnapshot shares mutable slices with its input")
	}
}

func TestClientAccessorsDoNotExposeSnapshotSlices(t *testing.T) {
	c := &Client{snapshot: ircclient.StateSnapshot{Servers: []ircclient.ServerSnapshot{{Name: "test", Capabilities: []string{"message-tags"}, Targets: []ircclient.TargetSnapshot{{Target: "#test", TypingUsers: []string{"alice"}, Users: []ircclient.UserSnapshot{{Nick: "alice", Prefix: "@"}}}}}}}}
	c.Capabilities("test")[0] = "changed"
	c.Names("test", "#test")[0] = "changed"
	c.TypingUsers("test", "#test")[0] = "changed"
	c.KnownTargets("test")[0] = "changed"
	if c.Capabilities("test")[0] != "message-tags" || c.Names("test", "#test")[0] != "alice" || c.TypingUsers("test", "#test")[0] != "alice" || c.KnownTargets("test")[0] != "#test" || c.NickPrefix("test", "#test", "alice") != "@" {
		t.Fatal("accessor mutated retained snapshot")
	}
}

func BenchmarkClientNickPrefix(b *testing.B) {
	s := ircclient.ServerSnapshot{Name: "test"}
	for i := 0; i < 20; i++ {
		users := make([]ircclient.UserSnapshot, 1000)
		users[0] = ircclient.UserSnapshot{Nick: "alice", Prefix: "@"}
		s.Targets = append(s.Targets, ircclient.TargetSnapshot{Target: "#test", Users: users})
	}
	c := &Client{snapshot: ircclient.StateSnapshot{Servers: []ircclient.ServerSnapshot{s}}}
	b.Run("previous", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			snapshot := cloneServerSnapshot(s)
			if snapshot.Targets[0].Users[0].Prefix != "@" {
				b.Fatal("missing prefix")
			}
		}
	})
	b.Run("lookup", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if c.NickPrefix("test", "#test", "alice") != "@" {
				b.Fatal("missing prefix")
			}
		}
	})
}

func TestInitialSyncPreservesBufferedLiveFrames(t *testing.T) {
	c := &Client{dec: json.NewDecoder(strings.NewReader(`{"type":"hello","version":2}
{"type":"history","messages":[{"Text":"retained"}]}
{"type":"ready"}
{"type":"message","message":{"Text":"live"}}
`))}
	if err := c.initialSync(); err != nil {
		t.Fatal(err)
	}
	if len(c.pendingHistory) != 1 || c.pendingHistory[0].Text != "retained" || !c.pendingHistory[0].Replay {
		t.Fatal("lost history")
	}
	var next frame
	if err := c.dec.Decode(&next); err != nil {
		t.Fatal(err)
	}
	if next.Message == nil || next.Message.Text != "live" {
		t.Fatal("lost buffered live frame")
	}
}
