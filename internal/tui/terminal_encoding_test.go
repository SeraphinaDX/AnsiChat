//go:build !js && !wasm && !plan9

package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/mock"
	ui "github.com/metaspartan/gotui/v5"
)

// Force real TTY reads to split inside UTF-8 codepoints. Injecting EventKey
// values into a simulation screen bypasses the decoder and misses this freeze.
type fragmentedReadTty struct {
	tcell.Tty
	maxRead int
}

func (tty fragmentedReadTty) Read(data []byte) (int, error) {
	return tty.Tty.Read(data[:min(len(data), tty.maxRead)])
}

func TestRawTerminalUnicodePaste(t *testing.T) {
	const helperEnv = "ANSICHAT_TEST_RAW_TERMINAL_PASTE"
	if os.Getenv(helperEnv) != "1" {
		// A regressed decoder busy-loops and can make Screen.Fini hang. Isolate
		// it so failure is bounded and cannot freeze the rest of the test suite.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRawTerminalUnicodePaste$", "-test.v")
		cmd.Env = append(os.Environ(), helperEnv+"=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("raw-terminal paste failed (%v; deadline=%v):\n%s", err, ctx.Err(), out)
		}
		t.Logf("raw-terminal child:\n%s", out)
		return
	}

	t.Setenv("LC_ALL", "en_US.UTF-8")
	configureTerminalUTF8()
	texts := []string{
		"Before writing this article, I looked up a god damned list of bad words. The list had “Jesus Christ” and I was like, “Jesus Christ! Jesus Christ is a bad word?”",
		"Café — naïve 🌸 👩‍💻 e\u0301 「文字」\nnext\tline\n/quit",
	}
	for _, maxRead := range []int{1, 128} {
		for i, text := range texts {
			t.Run(fmt.Sprintf("read=%d/text=%d", maxRead, i), func(t *testing.T) {
				rawTerminalPaste(t, text, maxRead)
			})
		}
	}
}

func rawTerminalPaste(t *testing.T, text string, maxRead int) {
	t.Helper()
	tty := &mock.MockTty{}
	tty.Reset()
	// The old mock's default extended-attributes reply is malformed; suppress
	// that unrelated startup reply so the test exercises only paste input.
	tty.ExtendedAttributes = ""
	screen, err := tcell.NewTerminfoScreenFromTty(fragmentedReadTty{Tty: tty, maxRead: maxRead})
	if err != nil {
		t.Fatal(err)
	}
	if err = screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.EnablePaste()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := pasteEvents(ctx, screen.EventQ())
	go func() {
		for _, b := range []byte("\x1b[200~" + text + "\x1b[201~") {
			select {
			case tty.ReadQ <- b:
			case <-ctx.Done():
				return
			}
		}
	}()
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	for {
		select {
		case event := <-events:
			key, ok := event.(*tcell.EventKey)
			if !ok || key.Key() != pasteKey {
				continue
			}
			a := newCatchupApp(t)
			a.selectBuffer(a.irc.ServerNames()[0], "#chat")
			a.handleUIEvent(ui.Event{Type: ui.KeyboardEvent, Payload: key})
			want := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
			if a.input.Text != want || a.input.Cursor != utf8.RuneCountInString(want) {
				t.Fatalf("raw-byte paste lost text: got %q, want %q", a.input.Text, want)
			}
			if a.stopped.Load() {
				t.Fatal("pasted /quit executed")
			}
			return
		case <-deadline.C:
			// Only reached inside the helper process. Do not call Fini on a
			// spinning decoder: terminate it without hanging the parent suite.
			fmt.Fprintln(os.Stderr, "terminal decoder stalled before completing the short Unicode paste")
			os.Exit(2)
		}
	}
}
