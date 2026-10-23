package tui

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gdamore/tcell/v3"
	ui "github.com/metaspartan/gotui/v5"
	"github.com/metaspartan/gotui/v5/widgets"
)

const inputBurstBytes = 8 << 10

// Some terminals/paste methods deliver ordinary key events instead of bracketed
// paste. Collect a small burst before editing/rendering, preserving all action
// keys and ordering. The deadline also keeps IRC updates responsive during a
// continuous stream of input; ordinary typing waits at most two milliseconds.
func (a *App) handleInputEvents(first ui.Event, events <-chan ui.Event) bool {
	if !a.batchableInput(first) {
		a.handleUIEvent(first)
		return true
	}
	var text strings.Builder
	text.WriteString(printableInputText(first))
	flush := func() {
		s := text.String()
		// Text such as "<Enter>" must not become gotui's special-key notation.
		a.handleUIEvent(ui.Event{Type: ui.KeyboardEvent, ID: "<InputText>", Payload: tcell.NewEventKey(tcell.KeyRune, s, tcell.ModNone)})
	}
	timer := time.NewTimer(2 * time.Millisecond)
	defer timer.Stop()
	for text.Len() < inputBurstBytes {
		select {
		case next, ok := <-events:
			if !ok {
				flush()
				return false
			}
			if !a.batchableInput(next) {
				flush()
				a.handleUIEvent(next)
				return true
			}
			text.WriteString(printableInputText(next))
		case <-timer.C:
			flush()
			return true
		}
	}
	flush()
	return true
}

func (a *App) batchableInput(e ui.Event) bool {
	if a.copyMode || a.jumpMode || e.Type != ui.KeyboardEvent {
		return false
	}
	key, ok := e.Payload.(*tcell.EventKey)
	if !ok || key == nil || key.Key() != tcell.KeyRune || key.Modifiers()&^tcell.ModShift != 0 {
		return false
	}
	text := key.Str()
	return text != "" && !a.printableShortcuts[text]
}

// Cache the byte position of the rune-indexed cursor. Cursor movement in a
// large, unchanged draft must not rescan/allocate the whole draft on each draw.
type inputPosition struct {
	text   string
	cursor int
	byte   int
}

func (p *inputPosition) locate(text string, cursor int) (int, int) {
	if p.text != text {
		*p = inputPosition{text: text}
	}
	cursor = max(cursor, 0)
	for p.cursor < cursor && p.byte < len(text) {
		_, size := utf8.DecodeRuneInString(text[p.byte:])
		p.byte += size
		p.cursor++
	}
	for p.cursor > cursor {
		_, size := utf8.DecodeLastRuneInString(text[:p.byte])
		p.byte -= size
		p.cursor--
	}
	return p.byte, p.cursor
}

func (a *App) insertInputText(text string) {
	if text == "" {
		return
	}
	at, cursor := a.inputPosition.locate(a.input.Text, a.input.Cursor)
	a.input.Text = a.input.Text[:at] + text + a.input.Text[at:]
	a.input.Cursor = cursor + utf8.RuneCountInString(text)
	a.inputPosition = inputPosition{text: a.input.Text, cursor: a.input.Cursor, byte: at + len(text)}
}

// gotui's Input scans all runes several times per Draw. Give it only a small
// window around the cursor, while keeping the complete editable/sendable text
// in a.input. Newlines remain visible as arrows without expanding a large paste.
func (a *App) inputForDisplay() *widgets.Input {
	width := max(a.input.Inner.Dx(), 1)
	if len(a.input.Text) <= width && !strings.ContainsAny(a.input.Text, "\n\t\r") {
		a.inputPosition = inputPosition{}
		return a.input
	}
	at, _ := a.inputPosition.locate(a.input.Text, a.input.Cursor)
	start, end, cursor := at, at, 0
	for cursor < width && start > 0 {
		_, size := utf8.DecodeLastRuneInString(a.input.Text[:start])
		start -= size
		cursor++
	}
	for n := 0; n < width && end < len(a.input.Text); n++ {
		_, size := utf8.DecodeRuneInString(a.input.Text[end:])
		end += size
	}
	preview := widgets.NewInput()
	a.theme.applyInput(preview)
	preview.SetRect(a.input.Min.X, a.input.Min.Y, a.input.Max.X, a.input.Max.Y)
	preview.Title = a.input.Title
	preview.Text = strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\r':
			return '↵'
		case '\t':
			return '→'
		}
		return r
	}, a.input.Text[start:end])
	preview.Cursor = cursor
	return preview
}
