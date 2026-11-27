// Package setup guides a desktop user through creating a private configuration.
package setup

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"afishk/internal/config"
)

// DefaultPath is shared by setup and the normal server invocation.
func DefaultPath() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(base, "afishk", "config.json")
}

// Run guides the user through setup. EOF cancels setup before saving.
func Run(in io.Reader, out io.Writer, path string, overwrite bool) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	c, err := setupConfig(path, overwrite)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(in)
	ask := func(label, fallback string) (string, error) {
		if _, err := fmt.Fprintf(out, "%s [%s]: ", label, fallback); err != nil {
			return "", err
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("setup cancelled: %w", err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			line = fallback
		}
		return line, nil
	}
	if overwrite {
		if _, err := fmt.Fprintln(out, "Saving will replace", path); err != nil {
			return err
		}
	}
	if _, err = fmt.Fprintln(out, "aFISHk setup — Enter accepts defaults; Ctrl-D cancels.\nRun afishk inspect first for the exact WoW class and title.\nUse your trusted LAN/VPN address; setup does not change DNS or firewall rules."); err != nil {
		return err
	}
	if c.Listen, err = ask("Listen IP:port", c.Listen); err != nil {
		return err
	}
	host, port, err := net.SplitHostPort(c.Listen)
	if err != nil || net.ParseIP(host) == nil {
		return fmt.Errorf("listen must be a literal IP:port (IPv6: [address]:port)")
	}
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("listen port must be between 1 and 65535")
	}
	mode, err := ask("TLS: no, builtin (PEM files), or proxy (external HTTPS)", tlsMode(c))
	if err != nil {
		return err
	}
	scheme := "http"
	switch mode {
	case "no":
	case "builtin":
		scheme = "https"
		if c.TLSCert, err = ask("Absolute certificate PEM path", c.TLSCert); err != nil {
			return err
		}
		if c.TLSKey, err = ask("Absolute private key PEM path", c.TLSKey); err != nil {
			return err
		}
		for _, p := range []string{c.TLSCert, c.TLSKey} {
			if !filepath.IsAbs(p) {
				return fmt.Errorf("TLS paths must be absolute")
			}
		}
		if _, err = tls.LoadX509KeyPair(c.TLSCert, c.TLSKey); err != nil {
			return fmt.Errorf("TLS certificate/key: %w", err)
		}
	case "proxy":
		scheme = "https"
	default:
		return fmt.Errorf("TLS must be no, builtin, or proxy")
	}
	if mode != "builtin" {
		c.TLSCert, c.TLSKey = "", ""
	}
	originDefault := scheme + "://" + net.JoinHostPort(host, port)
	if net.ParseIP(host).IsUnspecified() || mode == "proxy" {
		originDefault = ""
	}
	if strings.HasPrefix(c.PublicOrigin, scheme+"://") {
		originDefault = c.PublicOrigin
	}
	if c.PublicOrigin, err = ask("Browser origin (DNS name or IP, with scheme and port)", originDefault); err != nil {
		return err
	}
	if !strings.HasPrefix(c.PublicOrigin, scheme+"://") {
		return fmt.Errorf("browser origin must use %s for the selected TLS mode", scheme)
	}
	origin, err := url.Parse(c.PublicOrigin)
	if err != nil || origin.Hostname() == "" {
		return fmt.Errorf("enter a browser origin with a DNS name or IP")
	}
	if ip := net.ParseIP(origin.Hostname()); ip != nil && ip.IsUnspecified() {
		return fmt.Errorf("browser origin needs a reachable IP or DNS name, not a wildcard address")
	}
	mediaIP := host
	if net.ParseIP(host).IsUnspecified() || net.ParseIP(host).To4() == nil {
		mediaIP = ""
	}
	if c.ICEAddress != "" {
		mediaIP = c.ICEAddress
	}
	if c.ICEAddress, err = ask("Desktop IPv4 reachable by phone for audio (required for wildcard/IPv6 listeners)", mediaIP); err != nil {
		return err
	}
	if c.ICEAddress == "" {
		return fmt.Errorf("set a reachable desktop IPv4 address for audio")
	}
	if ip := net.ParseIP(c.ICEAddress); ip != nil && ip.IsUnspecified() {
		return fmt.Errorf("audio IPv4 must be reachable, not a wildcard address")
	}
	if c.GameClass, err = ask("Exact game class from afishk inspect", c.GameClass); err != nil {
		return err
	}
	if c.GameTitle, err = ask("Exact game title from afishk inspect", c.GameTitle); err != nil {
		return err
	}
	if err = configurePlatform(&c, ask, overwrite); err != nil {
		return err
	}
	c.TokenFile = tokenPath(c.TokenFile, path)
	if _, err = Save(c, path, overwrite); err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "Saved %s\nToken: %s (use as the browser Basic auth password; any username).\nOpen %s. Allow TCP %s and UDP %d–%d from trusted clients.\nValidate: afishk --config %q --check\nThen follow the platform startup instructions in README\n", path, c.TokenFile, c.PublicOrigin, port, c.UDPMin, c.UDPMax, path)
	if err != nil {
		return err
	}
	return printQR(out, c.PublicOrigin)
}

func tlsMode(c config.Config) string {
	if c.TLSCert != "" {
		return "builtin"
	}
	if strings.HasPrefix(c.PublicOrigin, "https://") {
		return "proxy"
	}
	return "no"
}
