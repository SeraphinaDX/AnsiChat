package tui

import (
	"context"
	"fmt"
	"image"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gdamore/tcell/v3"
	ui "github.com/metaspartan/gotui/v5"
	"github.com/metaspartan/gotui/v5/widgets"
)

func textKey(text string) ui.Event {
	return ui.Event{Type: ui.KeyboardEvent, ID: text, Payload: tcell.NewEventKey(tcell.KeyRune, text, tcell.ModNone)}
}

func TestLargeTerminalPasteKeepsFullEditableText(t *testing.T) {
	for _, bracketed := range []bool{false, true} {
		t.Run(fmt.Sprintf("bracketed=%v", bracketed), func(t *testing.T) {
			screen := tcell.NewSimulationScreen("UTF-8")
			if err := screen.Init(); err != nil {
				t.Fatal(err)
			}
			screen.SetSize(100, 24)
			previous := ui.DefaultBackend
			ui.DefaultBackend = &ui.Backend{Screen: screen}
			t.Cleanup(func() { ui.DefaultBackend = previous; screen.Fini() })
			a := newCatchupApp(t)
			a.selectBuffer(a.irc.ServerNames()[0], "#chat")
			backend := &pasteBackend{calls: make(chan string, 1)}
			a.irc = backend
			a.input.SetRect(0, 20, 100, 23)
			text := strings.Repeat("website héllo 🌸 <literal> ", 2048)
			if bracketed {
				text += "\n/quit\tstill literal"
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			source := make(chan tcell.Event, 256)
			adapted := &ui.Backend{Screen: &pasteScreen{Screen: screen, events: pasteEvents(ctx, source)}}
			events := adapted.PollEventsWithContext(ctx)
			go func() {
				defer close(source)
				if bracketed {
					source <- tcell.NewEventPaste(true)
				}
				for _, r := range text {
					source <- tcell.NewEventKey(tcell.KeyRune, string(r), tcell.ModNone)
				}
				if bracketed {
					source <- tcell.NewEventPaste(false)
				}
			}()
			frames := 0
			for first := range events {
				open := a.handleInputEvents(first, events)
				ui.Render(a.inputForDisplay())
				frames++
				if !open {
					break
				}
			}
			if a.input.Text != text || a.input.Cursor != utf8.RuneCountInString(text) {
				t.Fatal("large paste changed or truncated the draft/cursor")
			}
			if frames >= utf8.RuneCountInString(text)/8 {
				t.Fatalf("text was redrawn almost character by character: %d frames", frames)
			}
			select {
			case <-backend.calls:
				t.Fatal("inserting a paste sent a message")
			default:
			}
			if a.stopped.Load() {
				t.Fatal("literal /quit executed")
			}
			// Edit within the original Unicode draft after displaying the preview.
			a.handleKey(ui.Event{Type: ui.KeyboardEvent, ID: "<Left>"})
			a.handleKey(textKey("新"))
			runes := []rune(text)
			want := string(runes[:len(runes)-1]) + "新" + string(runes[len(runes)-1:])
			if a.input.Text != want {
				t.Fatal("editing the preview lost or misplaced original text")
			}
			a.handleKey(ui.Event{Type: ui.KeyboardEvent, ID: "<Backspace>"})
			if a.input.Text != text {
				t.Fatal("backspace after paste did not restore the draft")
			}
			t.Logf("%d bytes accepted and drawn in %d frame(s)", len(text), frames)
		})
	}
}

func TestInputBurstPreservesEditingAndShortcutOrder(t *testing.T) {
	for _, clearBinding := range []string{"Ctrl+U", "x", "<"} {
		t.Run(clearBinding, func(t *testing.T) {
			a := newCatchupApp(t)
			a.selectBuffer(a.irc.ServerNames()[0], "#chat")
			a.cfg.Keybindings.ClearInput = clearBinding
			a.printableShortcuts = a.cfg.Keybindings.PrintableShortcuts()
			clear := textKey(clearBinding)
			if clearBinding == "Ctrl+U" {
				clear = ui.Event{Type: ui.KeyboardEvent, ID: "<C-u>", Payload: tcell.NewEventKey(tcell.KeyCtrlU, "", tcell.ModCtrl)}
			}
			events := make(chan ui.Event, 7)
			for _, e := range []ui.Event{textKey("a"), textKey("b"), clear, textKey("é"), textKey("🌸"), {Type: ui.KeyboardEvent, ID: "<Left>"}, textKey("新")} {
				events <- e
			}
			close(events)
			for first := range events {
				if !a.handleInputEvents(first, events) {
					break
				}
			}
			if a.input.Text != "é新🌸" || a.input.Cursor != 2 {
				t.Fatalf("shortcut/edit order changed: %q cursor=%d", a.input.Text, a.input.Cursor)
			}
		})
	}
}

func TestInputBurstSpecialKeyNamesStayLiteral(t *testing.T) {
	for _, text := range []string{"<Enter>", "<C-c>", "<Home>", "/quit"} {
		t.Run(text, func(t *testing.T) {
			a := newCatchupApp(t)
			a.selectBuffer(a.irc.ServerNames()[0], "#chat")
			events := make(chan ui.Event, len(text))
			for _, r := range text {
				events <- textKey(string(r))
			}
			close(events)
			for first := range events {
				if !a.handleInputEvents(first, events) {
					break
				}
			}
			if a.input.Text != text || a.input.Cursor != len(text) || a.stopped.Load() {
				t.Fatalf("literal text interpreted as a key/command: %q cursor=%d", a.input.Text, a.input.Cursor)
			}
		})
	}
}

func TestInputBurstPreservesJumpModeDigits(t *testing.T) {
	a := newCatchupApp(t)
	a.jumpMode = true
	events := make(chan ui.Event, 2)
	events <- textKey("1")
	events <- textKey("2")
	close(events)
	for first := range events {
		a.handleInputEvents(first, events)
	}
	if a.jumpDigits != "12" || a.input.Text != "" {
		t.Fatal("jump-mode digits were combined into text input")
	}
}

func TestLargeInputPreviewIsBoundedAndDoesNotChangeDraft(t *testing.T) {
	a := &App{input: widgets.NewInput()}
	a.input.SetRect(0, 0, 80, 3)
	text := strings.Repeat("é🌸\n\t", 1<<16)
	a.input.Text = text
	for _, cursor := range []int{utf8.RuneCountInString(text), 10, 0, -1, 1 << 22} {
		a.input.Cursor = cursor
		preview := a.inputForDisplay()
		preview.Draw(ui.NewBuffer(image.Rect(0, 0, 80, 3)))
		if utf8.RuneCountInString(preview.Text) > 2*a.input.Inner.Dx() {
			t.Fatal("renderer received the entire large draft")
		}
		if strings.ContainsAny(preview.Text, "\n\t\r") || !utf8.ValidString(preview.Text) {
			t.Fatal("preview contains line breaks or damaged Unicode")
		}
		if a.input.Text != text || a.input.Cursor != cursor {
			t.Fatal("drawing changed the original draft or cursor")
		}
	}
}

func BenchmarkLargeInput(b *testing.B) {
	b.Run("insert_50KiB", func(b *testing.B) {
		text := strings.Repeat("website text ", 3846)
		a := &App{input: widgets.NewInput()}
		b.SetBytes(int64(len(text)))
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			a.input.Text, a.input.Cursor = "", 0
			for start := 0; start < len(text); start += inputBurstBytes {
				a.insertInputText(text[start:min(start+inputBurstBytes, len(text))])
			}
		}
	})
	for _, size := range []int{50 << 10, 1 << 20} {
		b.Run(fmt.Sprintf("draw_%d_bytes", size), func(b *testing.B) {
			a := &App{input: widgets.NewInput()}
			a.input.SetRect(0, 0, 100, 3)
			a.insertInputText(strings.Repeat("x", size))
			buf := ui.NewBuffer(image.Rect(0, 0, 100, 3))
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				a.inputForDisplay().Draw(buf)
			}
		})
	}
}
