package dcc

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Kind string

const (
	Send            Kind  = "SEND"
	Chat            Kind  = "CHAT"
	maxFileBytes    int64 = 1 << 30
	transferTimeout       = 10 * time.Minute
)

type Offer struct {
	ID       string
	Server   string
	Peer     string
	Kind     Kind
	Filename string
	Address  string
	Port     int
	Size     int64
	Token    string
	Created  time.Time
}

type Event struct {
	Server string
	Peer   string
	Text   string
}

type Manager struct {
	mu          sync.Mutex
	active      chan struct{}
	offers      map[string]Offer
	downloadDir string
	listenAddr  string
	advertiseIP string
	emit        func(Event)
}

func New(downloadDir, listenAddr, advertiseIP string, emit func(Event)) *Manager {
	return &Manager{
		offers:      make(map[string]Offer),
		active:      make(chan struct{}, 4),
		downloadDir: downloadDir,
		listenAddr:  listenAddr,
		advertiseIP: advertiseIP,
		emit:        emit,
	}
}

func (m *Manager) ParseOffer(server, peer, text string) (Offer, error) {
	parts, err := splitArgs(text)
	if err != nil || len(parts) == 0 {
		return Offer{}, errors.New("invalid DCC payload")
	}
	switch strings.ToUpper(parts[0]) {
	case "SEND":
		if len(parts) < 5 {
			return Offer{}, errors.New("invalid DCC SEND")
		}
		port, err := strconv.Atoi(parts[3])
		if err != nil || port < 0 || port > 65535 {
			return Offer{}, errors.New("invalid DCC port")
		}
		size, err := strconv.ParseInt(parts[4], 10, 64)
		if err != nil || size < 0 || size > maxFileBytes {
			return Offer{}, errors.New("DCC size must be between 0 and 1 GiB")
		}
		addr, err := decodeAddress(parts[2])
		if err != nil {
			return Offer{}, err
		}
		name := filepath.Base(parts[1])
		if name == "." || name == ".." || name == string(filepath.Separator) {
			return Offer{}, errors.New("invalid DCC filename")
		}
		o := Offer{ID: id(server, peer), Server: server, Peer: peer, Kind: Send, Filename: filepath.Base(parts[1]), Address: addr, Port: port, Size: size, Created: time.Now()}
		if len(parts) > 5 {
			o.Token = parts[5]
		}
		m.mu.Lock()
		for key, offer := range m.offers {
			if time.Since(offer.Created) > 10*time.Minute {
				delete(m.offers, key)
			}
		}
		if len(m.offers) >= 128 {
			m.mu.Unlock()
			return Offer{}, errors.New("too many pending DCC offers")
		}
		m.offers[o.ID] = o
		m.mu.Unlock()
		return o, nil
	case "CHAT":
		if len(parts) < 4 {
			return Offer{}, errors.New("invalid DCC CHAT")
		}
		port, err := strconv.Atoi(parts[3])
		if err != nil || port < 1 || port > 65535 {
			return Offer{}, errors.New("invalid DCC port")
		}
		addr, err := decodeAddress(parts[2])
		if err != nil {
			return Offer{}, err
		}
		o := Offer{ID: id(server, peer), Server: server, Peer: peer, Kind: Chat, Address: addr, Port: port, Created: time.Now()}
		m.mu.Lock()
		for key, offer := range m.offers {
			if time.Since(offer.Created) > 10*time.Minute {
				delete(m.offers, key)
			}
		}
		if len(m.offers) >= 128 {
			m.mu.Unlock()
			return Offer{}, errors.New("too many pending DCC offers")
		}
		m.offers[o.ID] = o
		m.mu.Unlock()
		return o, nil
	default:
		return Offer{}, fmt.Errorf("unsupported DCC command %q", parts[0])
	}
}

func (m *Manager) Offers() []Offer {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Offer, 0, len(m.offers))
	for _, o := range m.offers {
		out = append(out, o)
	}
	return out
}

func (m *Manager) AcceptSend(server, peer string) error {
	o, ok := m.get(server, peer)
	if !ok || o.Kind != Send {
		return errors.New("no pending DCC SEND from that peer")
	}
	if !m.acquire() {
		return errors.New("too many active DCC transfers")
	}
	go func() { defer m.release(); m.receiveFile(o) }()
	return nil
}

func (m *Manager) AcceptChat(server, peer string, onLine func(string)) error {
	o, ok := m.get(server, peer)
	if !ok || o.Kind != Chat {
		return errors.New("no pending DCC CHAT from that peer")
	}
	if !m.acquire() {
		return errors.New("too many active DCC transfers")
	}
	go func() {
		defer m.release()
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(o.Address, strconv.Itoa(o.Port)), 15*time.Second)
		if err != nil {
			m.say(o.Server, o.Peer, "CHAT connect failed: "+err.Error())
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(transferTimeout))
		m.say(o.Server, o.Peer, "CHAT connected")
		s := bufio.NewScanner(conn)
		for s.Scan() {
			onLine(s.Text())
		}
		if err := s.Err(); err != nil {
			m.say(o.Server, o.Peer, "CHAT error: "+err.Error())
		}
		m.say(o.Server, o.Peer, "CHAT closed")
	}()
	return nil
}

func (m *Manager) SendFile(server, peer, path string, sendCTCP func(payload string)) error {
	if !m.acquire() {
		return errors.New("too many active DCC transfers")
	}
	started := false
	defer func() {
		if !started {
			m.release()
		}
	}()
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		if !started {
			f.Close()
		}
	}()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || st.Size() > maxFileBytes {
		return errors.New("DCC SEND requires a regular file no larger than 1 GiB")
	}
	ln, err := net.Listen("tcp", m.listenAddr)
	if err != nil {
		return err
	}
	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		ln.Close()
		return errors.New("DCC listener is not TCP")
	}
	advertise, err := m.addressForAdvertisement(tcpAddr)
	if err != nil {
		ln.Close()
		return err
	}
	name := filepath.Base(path)
	payload := fmt.Sprintf("SEND %q %s %d %d", name, advertise, tcpAddr.Port, st.Size())
	sendCTCP(payload)
	m.say(server, peer, fmt.Sprintf("offered %s (%d bytes)", name, st.Size()))
	started = true
	go func() {
		defer m.release()
		defer f.Close()
		defer ln.Close()
		if tl, ok := ln.(*net.TCPListener); ok {
			_ = tl.SetDeadline(time.Now().Add(2 * time.Minute))
		}
		conn, err := ln.Accept()
		if err != nil {
			m.say(server, peer, "SEND accept failed: "+err.Error())
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(transferTimeout))
		n, err := io.CopyN(conn, f, st.Size())
		if err != nil {
			m.say(server, peer, fmt.Sprintf("SEND failed after %d bytes: %v", n, err))
			return
		}
		m.say(server, peer, fmt.Sprintf("SEND complete: %s (%d bytes)", name, n))
	}()
	return nil
}

func (m *Manager) receiveFile(o Offer) {
	if o.Port == 0 {
		m.say(o.Server, o.Peer, "passive/reverse DCC SEND is not implemented yet")
		return
	}
	if err := os.MkdirAll(m.downloadDir, 0o700); err != nil {
		m.say(o.Server, o.Peer, "cannot create download directory: "+err.Error())
		return
	}
	path := uniquePath(filepath.Join(m.downloadDir, filepath.Base(o.Filename)))
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(o.Address, strconv.Itoa(o.Port)), 15*time.Second)
	if err != nil {
		m.say(o.Server, o.Peer, "SEND connect failed: "+err.Error())
		return
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(transferTimeout))
	reader := io.LimitReader(conn, o.Size)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		m.say(o.Server, o.Peer, "SEND create failed: "+err.Error())
		return
	}
	defer f.Close()
	complete := false
	defer func() {
		if !complete {
			_ = os.Remove(path)
		}
	}()
	m.say(o.Server, o.Peer, "receiving "+filepath.Base(path))
	buf := make([]byte, 64*1024)
	var total uint64
	for {
		n, rerr := reader.Read(buf)
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				m.say(o.Server, o.Peer, "SEND write failed: "+err.Error())
				return
			}
			total += uint64(n)
			var ack [4]byte
			binary.BigEndian.PutUint32(ack[:], uint32(total))
			_, _ = conn.Write(ack[:])
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			m.say(o.Server, o.Peer, "SEND read failed: "+rerr.Error())
			return
		}
	}
	if int64(total) != o.Size {
		m.say(o.Server, o.Peer, "SEND failed: incomplete file")
		return
	}
	complete = true
	m.say(o.Server, o.Peer, fmt.Sprintf("received %s (%d bytes)", filepath.Base(path), total))
}

func (m *Manager) get(server, peer string) (Offer, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.offers[id(server, peer)]
	if ok {
		delete(m.offers, id(server, peer))
	}
	return o, ok
}

func (m *Manager) say(server, peer, text string) {
	if m.emit != nil {
		m.emit(Event{Server: server, Peer: peer, Text: text})
	}
}

func (m *Manager) addressForAdvertisement(addr *net.TCPAddr) (string, error) {
	ip := net.ParseIP(m.advertiseIP)
	if ip == nil {
		ip = addr.IP
	}
	if ip == nil || ip.IsUnspecified() {
		return "", errors.New("dcc.advertise_ip must be set when listening on a wildcard address")
	}
	if v4 := ip.To4(); v4 != nil {
		return strconv.FormatUint(uint64(binary.BigEndian.Uint32(v4)), 10), nil
	}
	return ip.String(), nil
}

func decodeAddress(s string) (string, error) {
	if ip := net.ParseIP(s); ip != nil {
		return ip.String(), nil
	}
	n, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return "", fmt.Errorf("invalid DCC address %q", s)
	}
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(n))
	return net.IP(b[:]).String(), nil
}

func uniquePath(path string) string {
	if _, err := os.Stat(path); err != nil {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(candidate); err != nil {
			return candidate
		}
	}
}

func id(server, peer string) string { return server + "\x00" + strings.ToLower(peer) }

func splitArgs(s string) ([]string, error) {
	var out []string
	var cur strings.Builder
	quoted := false
	escaped := false
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for _, r := range strings.TrimSpace(s) {
		if escaped {
			cur.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' && quoted {
			escaped = true
			continue
		}
		if r == '"' {
			quoted = !quoted
			continue
		}
		if (r == ' ' || r == '\t') && !quoted {
			flush()
			continue
		}
		cur.WriteRune(r)
	}
	if quoted {
		return nil, errors.New("unterminated quoted DCC argument")
	}
	flush()
	return out, nil
}

func (m *Manager) acquire() bool {
	select {
	case m.active <- struct{}{}:
		return true
	default:
		return false
	}
}
func (m *Manager) release() { <-m.active }
