// Package config reads the operator-owned JSON configuration.
package config

import (
	"afishk/internal/keybind"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	LureKey                 string `json:"lure_key,omitempty"`
	FishKey                 string `json:"fish_key"`
	GameExecutable          string `json:"game_executable,omitempty"`
	Listen                  string `json:"listen"`
	PublicOrigin            string `json:"public_origin"`
	AllowUnauthenticatedLAN bool   `json:"allow_unauthenticated_lan"`
	TokenFile               string `json:"token_file"`
	TLSCert                 string `json:"tls_cert"`
	TLSKey                  string `json:"tls_key"`
	CacheDir                string `json:"cache_dir"`
	GameClass               string `json:"game_class"`
	GameTitle               string `json:"game_title"`
	Ydotool                 string `json:"ydotool"`
	FFmpeg                  string `json:"ffmpeg"`
	PulseSource             string `json:"pulse_source"`
	ICEAddress              string `json:"ice_address"`
	UDPMin                  uint16 `json:"udp_min"`
	UDPMax                  uint16 `json:"udp_max"`
}

// Defaults returns portable defaults plus the selected platform settings.
func Defaults() (Config, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return Config{}, err
	}
	c := Config{FishKey: "F", Listen: "127.0.0.1:8787", CacheDir: filepath.Join(cache, "afishk"),
		GameTitle: "World of Warcraft", UDPMin: 49160, UDPMax: 49180}
	err = platformDefaults(&c)
	return c, err
}

func Load(path string) (Config, error) {
	c, err := Defaults()
	if err != nil {
		return c, err
	}
	f, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer func() { _ = f.Close() }()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err = dec.Decode(&c); err != nil {
		return c, err
	}
	if err = dec.Decode(new(any)); err != io.EOF {
		return c, errors.New("config must contain one JSON object")
	}
	return c, c.Validate()
}

func (c Config) Validate() error {
	host, port, err := net.SplitHostPort(c.Listen)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	n, portErr := strconv.Atoi(port)
	if portErr != nil || n < 1 || n > 65535 {
		return errors.New("listen port must be between 1 and 65535")
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return errors.New("listen must use a literal IP address")
	}
	if !ip.IsLoopback() && c.TokenFile == "" && !c.AllowUnauthenticatedLAN {
		return errors.New("non-loopback listener requires token_file or explicit allow_unauthenticated_lan")
	}
	if (c.TLSCert == "") != (c.TLSKey == "") {
		return errors.New("tls_cert and tls_key must be supplied together")
	}
	if c.PublicOrigin != "" {
		u, e := url.Parse(c.PublicOrigin)
		if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
			return errors.New("public_origin must be an http(s) origin without a path")
		}
	}
	if c.GameClass == "" || c.GameTitle == "" {
		return errors.New("set exact game_class and game_title from `afishk inspect` before enabling control")
	}
	if !filepath.IsAbs(c.CacheDir) {
		return errors.New("cache_dir must be absolute")
	}
	if c.UDPMin == 0 || c.UDPMax < c.UDPMin {
		return errors.New("invalid UDP port range")
	}
	if c.ICEAddress != "" && net.ParseIP(c.ICEAddress).To4() == nil {
		return errors.New("ice_address must be a literal IPv4 VPN address")
	}
	if _, err := keybind.Parse(c.FishKey); err != nil {
		return fmt.Errorf("fish_key: %w", err)
	}
	if c.LureKey != "" {
		if _, err := keybind.Parse(c.LureKey); err != nil {
			return fmt.Errorf("lure_key: %w", err)
		}
	}
	return validatePlatform(c)
}
