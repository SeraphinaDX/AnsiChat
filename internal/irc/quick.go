package irc

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"unicode"

	"ansichat/internal/config"
	"github.com/lrstanley/girc"
)

const QuickUsage = "usage: /quick [-tls|-plain] host[:port] [port] [nickname]"

// quickConfig creates an isolated identity without inheriting network credentials.
func quickConfig(args string, general config.GeneralConfig) (config.ServerConfig, error) {
	sc := config.ServerConfig{TLS: true, Nick: general.Nick, User: general.User, RealName: general.RealName}
	var positional []string
	mode := ""
	for _, arg := range strings.Fields(args) {
		if arg == "-tls" || arg == "-plain" {
			if mode != "" {
				return sc, fmt.Errorf(QuickUsage)
			}
			mode = arg
			sc.TLS = arg == "-tls"
		} else {
			positional = append(positional, arg)
		}
	}
	if len(positional) < 1 || len(positional) > 3 || strings.ContainsAny(args, "\r\n\x00") {
		return sc, fmt.Errorf(QuickUsage)
	}
	host := positional[0]
	port := ""
	if strings.Contains(host, ":") {
		if net.ParseIP(host) == nil {
			var err error
			host, port, err = net.SplitHostPort(host)
			if err != nil {
				return sc, fmt.Errorf("invalid server address: %w", err)
			}
			if port == "" {
				return sc, fmt.Errorf("server address requires a port")
			}
		}
	}
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = strings.Trim(host, "[]")
	}
	if host == "" || strings.ContainsAny(host, "/\\@?#") || strings.HasPrefix(host, "-") || strings.IndexFunc(host, unicode.IsControl) >= 0 {
		return sc, fmt.Errorf("invalid server hostname")
	}
	remaining := positional[1:]
	if len(remaining) > 0 && port == "" {
		port = remaining[0]
		remaining = remaining[1:]
	}
	sc.Port = 6697
	if !sc.TLS {
		sc.Port = 6667
	}
	if port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return sc, fmt.Errorf("port must be between 1 and 65535")
		}
		sc.Port = n
	}
	if len(remaining) > 1 {
		return sc, fmt.Errorf(QuickUsage)
	}
	if len(remaining) == 1 {
		sc.Nick = remaining[0]
	}
	if sc.Nick == "" {
		sc.Nick = "ansichat"
	}
	if !girc.IsValidNick(sc.Nick) {
		return sc, fmt.Errorf("invalid IRC nickname")
	}
	if sc.User == "" {
		sc.User = sc.Nick
	}
	if sc.RealName == "" {
		sc.RealName = "AnsiChat IRC Client"
	}
	sc.Host = strings.ToLower(host)
	sc.Name = net.JoinHostPort(sc.Host, strconv.Itoa(sc.Port))
	if sc.TLS {
		sc.Name += "/tls"
	} else {
		sc.Name += "/plain"
	}
	return sc, nil
}

// QuickConnect starts a temporary session; it never edits the configuration.
func (m *Manager) QuickConnect(args string) (string, error) {
	sc, err := quickConfig(args, m.cfg.General)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	if existing := m.sessions[sc.Name]; existing != nil {
		if !existing.quick || existing.cfg.Host != sc.Host || existing.cfg.Port != sc.Port || existing.cfg.TLS != sc.TLS || existing.cfg.Nick != sc.Nick {
			m.mu.Unlock()
			return "", fmt.Errorf("server name %q is already in use", sc.Name)
		}
	} else {
		s := &Session{cfg: sc, quick: true}
		s.client, s.setupErr = m.newClient(sc)
		if s.setupErr != nil {
			m.mu.Unlock()
			return "", s.setupErr
		}
		m.sessions[sc.Name] = s
	}
	m.mu.Unlock()
	return sc.Name, m.ConnectServer(sc.Name)
}
