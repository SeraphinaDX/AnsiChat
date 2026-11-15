package relay

import (
	"bufio"
	"encoding/json"
	"io"
)

// Frames written by json.Encoder are newline terminated. Bound the wire frame
// before JSON allocation, including unknown fields and whitespace.
const maxRequestBytes = 64 * 1024

type requestDecoder struct{ scanner *bufio.Scanner }

func newRequestDecoder(r io.Reader) *requestDecoder {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 4096), maxRequestBytes)
	return &requestDecoder{scanner: s}
}

func (d *requestDecoder) Decode(v any) error {
	if !d.scanner.Scan() {
		if err := d.scanner.Err(); err != nil {
			return err
		}
		return io.EOF
	}
	return json.Unmarshal(d.scanner.Bytes(), v)
}
