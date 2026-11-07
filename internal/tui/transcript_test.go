package tui

import (
	"fmt"
	"image"
	"reflect"
	"strings"
	"testing"
	"time"

	"ansichat/internal/config"
	"ansichat/internal/model"

	ui "github.com/metaspartan/gotui/v5"
	"github.com/metaspartan/gotui/v5/widgets"
)

func TestTranscriptViewportIncludesNewestAfterWrappedRow(t *testing.T) {
	list := &transcriptList{List: widgets.NewList()}
	list.WrapText = true
	// With borders this gives an inner viewport of 12 columns x 3 rows.
	list.SetRect(0, 0, 14, 5)
	list.Rows = []string{
		"old",
		"this message wraps onto more than one terminal line",
		"previous",
		"newest",
	}
	list.SelectedRow = len(list.Rows) - 1

	lines, _, _ := list.visiblePhysicalLines()
	if len(lines) == 0 {
		t.Fatal("visiblePhysicalLines returned no lines")
	}
	if got := lines[len(lines)-1].logicalRow; got != len(list.Rows)-1 {
		t.Fatalf("last visible logical row = %d, want newest row %d", got, len(list.Rows)-1)
	}
	if len(lines) > list.Inner.Dy() {
		t.Fatalf("rendered %d physical lines into height %d", len(lines), list.Inner.Dy())
	}
}

func TestTranscriptScrollBottomNeverLagsOneLogicalRow(t *testing.T) {
	list := &transcriptList{List: widgets.NewList()}
	list.WrapText = true
	list.SetRect(0, 0, 18, 6)
	list.Rows = []string{
		"one",
		"two wraps because this line is deliberately long",
		"three",
		"four",
	}

	for i := 0; i < 20; i++ {
		list.Rows = append(list.Rows, "new message")
		if len(list.Rows) > 10 {
			list.Rows = list.Rows[1:]
		}
		list.ScrollBottom()
		lines, _, _ := list.visiblePhysicalLines()
		if len(lines) == 0 {
			t.Fatalf("iteration %d: no visible lines", i)
		}
		if got := lines[len(lines)-1].logicalRow; got != len(list.Rows)-1 {
			t.Fatalf("iteration %d: last visible logical row = %d, want %d", i, got, len(list.Rows)-1)
		}
	}
}

func TestTranscriptMentionWithBracketNickDoesNotCorruptMarkup(t *testing.T) {
	theme := uiTheme{cfg: config.ThemeConfig{
		Timestamp: "#77839a",
		Mention:   "#ff6f91",
	}}
	msg := model.Message{
		Time:    time.Unix(0, 0),
		Nick:    "[",
		Text:    "britney: https://en.wikipedia.org/wiki/Legal_fiction",
		Kind:    model.KindMessage,
		Mention: true,
	}

	row := theme.formatMessage(msg, "X")
	got := ui.CellsToString(parseTranscriptStyles(row, ui.NewStyle(ui.ColorWhite)))
	want := "X ! <[> britney: https://en.wikipedia.org/wiki/Legal_fiction"
	if got != want {
		t.Fatalf("rendered mention = %q, want %q", got, want)
	}
	if strings.Contains(got, "](fg:") || strings.Contains(got, "(fg:") {
		t.Fatalf("renderer leaked style markup into visible text: %q", got)
	}
}

func TestTranscriptDoesNotInterpretIRCTextAsGotuiMarkup(t *testing.T) {
	theme := uiTheme{cfg: config.ThemeConfig{
		Timestamp:  "#77839a",
		Foreground: "#ffffff",
		NickColors: []string{"#64d8cb"},
	}}
	msg := model.Message{
		Time: time.Unix(0, 0),
		Nick: "someone",
		Text: "literal [not markup](fg:#ff0000) stays literal",
		Kind: model.KindMessage,
	}

	row := theme.formatMessage(msg, "X")
	got := ui.CellsToString(parseTranscriptStyles(row, ui.NewStyle(ui.ColorWhite)))
	want := "X <someone> literal [not markup](fg:#ff0000) stays literal"
	if got != want {
		t.Fatalf("rendered message = %q, want %q", got, want)
	}
}

func TestTranscriptViewportCacheMatchesFreshRendering(t *testing.T) {
	list := &transcriptList{List: widgets.NewList()}
	list.WrapText = true
	list.SetRect(0, 0, 30, 8)
	list.Rows = []string{"first", "long line with wide characters 日本語 and search", "last"}
	list.SelectedRow = 2
	changes := []func(){func() {}, func() { list.SetRect(0, 0, 18, 8) }, func() { list.SelectedRow = 1 }, func() { list.searchQuery = "line" }, func() { list.searchMinRow = 2 }, func() { list.unread = true; list.unreadBefore = 1 }, func() { list.Rows[1] = "changed line" }, func() { list.TextStyle.Fg = ui.ColorRed }, func() { list.WrapText = false }, func() { list.Rows = []string{"replacement"}; list.SelectedRow = 0 }}
	for i, change := range changes {
		change()
		got, older, newer := list.visiblePhysicalLines()
		cached := list.wrappedRows
		list.wrappedRows = nil
		want, wantOlder, wantNewer := list.visiblePhysicalLines()
		if !reflect.DeepEqual(got, want) || older != wantOlder || newer != wantNewer {
			t.Fatalf("change %d produced stale rendering", i)
		}
		if len(cached) > list.Inner.Dy() {
			t.Fatal("cache exceeds viewport")
		}
		list.wrappedRows = cached
	}
}

func BenchmarkTranscriptViewport(b *testing.B) {
	for _, cached := range []bool{false, true} {
		b.Run(fmt.Sprintf("cached=%v", cached), func(b *testing.B) {
			list := &transcriptList{List: widgets.NewList()}
			list.WrapText = true
			list.SetRect(0, 0, 100, 30)
			for i := 0; i < 1000; i++ {
				list.Rows = append(list.Rows, fmt.Sprintf("%d %s", i, strings.Repeat("a chat message ", 8)))
			}
			list.SelectedRow = 999
			list.visiblePhysicalLines()
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if !cached {
					list.wrappedRows = nil
				}
				list.visiblePhysicalLines()
			}
		})
	}
}

func BenchmarkTranscriptCellDrawing(b *testing.B) {
	for _, previous := range []bool{true, false} {
		b.Run(fmt.Sprintf("previous=%v", previous), func(b *testing.B) {
			list := &transcriptList{List: widgets.NewList()}
			list.WrapText = true
			list.SetRect(0, 0, 100, 30)
			for i := 0; i < 1000; i++ {
				list.Rows = append(list.Rows, fmt.Sprintf("%d %s", i, strings.Repeat("a chat message ", 8)))
			}
			list.SelectedRow = 999
			lines, _, _ := list.visiblePhysicalLines()
			buf := ui.NewBuffer(image.Rect(0, 0, 100, 30))
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				for i, line := range lines {
					if previous {
						line.positioned = ui.BuildCellWithXArray(ui.TrimCells(line.cells, list.Inner.Dx()))
					}
					list.drawPhysicalLine(buf, line, list.Inner.Min.Y+i)
				}
			}
		})
	}
}
