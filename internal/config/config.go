// Package config reads the operator-owned JSON configuration.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
)

type Config struct {
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

func Load(path string) (Config, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return Config{}, err
	}
	c := Config{Listen: "127.0.0.1:8787", CacheDir: filepath.Join(cache, "afishk"),
		GameTitle: "World of Warcraft", Ydotool: "ydotool", FFmpeg: "ffmpeg", PulseSource: "@DEFAULT_MONITOR@", UDPMin: 49160, UDPMax: 49180}
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
	host, _, err := net.SplitHostPort(c.Listen)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
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
	if c.FFmpeg == "" || c.Ydotool == "" || c.PulseSource == "" {
		return errors.New("ffmpeg, ydotool and pulse_source must be nonempty")
	}
	return nil
}
