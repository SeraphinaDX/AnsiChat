package tui

import (
	"github.com/gdamore/tcell/v3"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// tcell 3.0.5 passes atEOF=true for each TTY read, although a read can end
// inside a UTF-8 character. Its validating decoder then reports an error with
// no progress, and scanInput spins forever before the paste-end marker arrives.
// Register a streaming decoder through the public encoding API, preserving
// incomplete characters for the next read and replacing genuinely invalid bytes.
// Newer tcell releases fix the EOF flag but remove APIs required by gotui 5.0.3.
func configureTerminalUTF8() {
	tcell.RegisterEncoding("UTF-8", terminalUTF8Encoding{})
	tcell.RegisterEncoding("UTF8", terminalUTF8Encoding{})
}

type terminalUTF8Encoding struct{}

func (terminalUTF8Encoding) NewEncoder() *encoding.Encoder {
	return unicode.UTF8.NewEncoder()
}

func (terminalUTF8Encoding) NewDecoder() *encoding.Decoder {
	return &encoding.Decoder{Transformer: terminalUTF8Decoder{unicode.UTF8.NewDecoder()}}
}

type terminalUTF8Decoder struct{ transform.Transformer }

func (d terminalUTF8Decoder) Transform(dst, src []byte, _ bool) (int, int, error) {
	// TTY input is an ongoing byte stream, not a complete file per read.
	return d.Transformer.Transform(dst, src, false)
}
