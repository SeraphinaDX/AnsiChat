package relay

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"ansichat/internal/config"
	"ansichat/internal/model"
)

type historySnapshot struct {
	Version  int
	Messages []model.Message
}

func newMessageID() string {
	var id [16]byte
	_, _ = rand.Read(id[:]) // crypto/rand.Read cannot fail on supported Go versions.
	return hex.EncodeToString(id[:])
}

func (s *Server) restoreHistory() error {
	if s.cfg.Relay.HistoryFile == "" {
		return nil
	}
	f, err := os.Open(config.ExpandPath(s.cfg.Relay.HistoryFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	var data historySnapshot
	dec := json.NewDecoder(f)
	if err := dec.Decode(&data); err != nil {
		return err
	}
	if data.Version != 1 {
		return fmt.Errorf("unsupported history format %d", data.Version)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return errors.New("trailing data in relay history")
	}
	// Validate completely before changing live state. A corrupt file is never
	// silently overwritten by an empty session.
	for _, msg := range data.Messages {
		if msg.RelayID == "" || msg.Server == "" || msg.Target == "" {
			return errors.New("invalid relay history message")
		}
	}
	for _, msg := range data.Messages {
		msg.Replay = false
		msg.SuppressNotify = false
		s.state.Add(msg) // enforces the current per-buffer history limit
	}
	return nil
}

func (s *Server) saveHistory() error {
	if s.cfg.Relay.HistoryFile == "" {
		return nil
	}
	path := config.ExpandPath(s.cfg.Relay.HistoryFile)
	buffers, _ := s.state.Snapshot()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".ansichat-history-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := writeHistory(f, buffers); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

// Stream individual messages so the JSON encoder never builds a second
// history-sized buffer. Keep the snapshot outside disk I/O to avoid blocking
// incoming IRC messages on a slow filesystem.
func writeHistory(w io.Writer, buffers []*model.Buffer) error {
	out := bufio.NewWriter(w)
	if _, err := out.WriteString(`{"Version":1,"Messages":[`); err != nil {
		return err
	}
	enc := json.NewEncoder(out)
	first := true
	for _, b := range buffers {
		for i := range b.Messages {
			if !first {
				if err := out.WriteByte(','); err != nil {
					return err
				}
			}
			first = false
			if err := enc.Encode(&b.Messages[i]); err != nil {
				return err
			}
		}
	}
	if _, err := out.WriteString("]}\n"); err != nil {
		return err
	}
	return out.Flush()
}

// Disk I/O runs independently of IRC delivery. Checkpoint once per second;
// clean shutdown joins this worker before writing the final snapshot.
func (s *Server) historyLoop(ctx context.Context) {
	if s.cfg.Relay.HistoryFile == "" {
		return
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !s.historyDirty.Swap(false) {
				continue
			}
			if err := s.saveHistory(); err != nil {
				s.historyDirty.Store(true)
				log.Printf("save relay history: %v", err)
			}
		}
	}
}
