package setup

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"afishk/internal/config"
	"afishk/internal/privatefs"
)

func setupConfig(path string, overwrite bool) (config.Config, error) {
	info, err := os.Lstat(path)
	switch {
	case os.IsNotExist(err) && !overwrite:
		return config.Defaults()
	case err != nil:
		return config.Config{}, err
	case !overwrite:
		return config.Config{}, fmt.Errorf("configuration already exists: %s; use --overwrite to rerun setup", path)
	case !info.Mode().IsRegular():
		return config.Config{}, fmt.Errorf("configuration must be a regular file: %s", path)
	}
	return config.Load(path)
}

// Save installs the new settings only after validation and keeps the existing token.
func Save(c config.Config, path string, overwrite bool) (string, error) {
	configInfo, err := os.Lstat(path)
	if overwrite {
		if err != nil {
			return "", err
		}
		if !configInfo.Mode().IsRegular() {
			return "", fmt.Errorf("configuration must be a regular file: %s", path)
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("configuration already exists or cannot be accessed: %s", path)
	}
	c.TokenFile = tokenPath(c.TokenFile, path)
	absToken, err := filepath.Abs(c.TokenFile)
	if err != nil {
		return "", err
	}
	if absToken == path {
		return "", errors.New("configuration and token must use different files")
	}
	c.AllowUnauthenticatedLAN = false
	if err := c.Validate(); err != nil {
		return "", err
	}
	if tokenInfo, err := os.Lstat(c.TokenFile); err == nil && overwrite && os.SameFile(configInfo, tokenInfo) {
		return "", errors.New("configuration and token must use different files")
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	token, err := readOrCreateToken(c.TokenFile)
	if err != nil {
		return "", err
	}
	write := privatefs.WriteNew
	if overwrite {
		write = privatefs.Replace
	}
	if err := write(path, append(data, '\n')); err != nil {
		return "", err
	}
	return token, nil
}

func tokenPath(configured, configPath string) string {
	if configured != "" {
		return configured
	}
	return filepath.Join(filepath.Dir(configPath), "token")
}

func readOrCreateToken(path string) (string, error) {
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		token, err := newToken()
		if err != nil {
			return "", err
		}
		if err := privatefs.WriteNew(path, []byte(token+"\n")); !os.IsExist(err) {
			return token, err
		}
	} else if err != nil {
		return "", err
	}
	if err := privatefs.CheckFile(path); err != nil {
		return "", fmt.Errorf("token must be a private regular file: %s: %w", path, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(data))
	if len(token) < 32 {
		return "", fmt.Errorf("existing token is too short: %s", path)
	}
	return token, nil
}

// RotateToken changes only the private token file referenced by the config.
func RotateToken(path string) (string, error) {
	c, err := config.Load(path)
	if err != nil {
		return "", err
	}
	if c.TokenFile == "" {
		return "", errors.New("configuration has no token_file")
	}
	configInfo, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	tokenInfo, err := os.Lstat(c.TokenFile)
	if err != nil {
		return "", err
	}
	if os.SameFile(configInfo, tokenInfo) {
		return "", errors.New("configuration and token must use different files")
	}
	if err := privatefs.CheckFile(c.TokenFile); err != nil {
		return "", err
	}
	token, err := newToken()
	if err != nil {
		return "", err
	}
	return c.TokenFile, privatefs.Replace(c.TokenFile, []byte(token+"\n"))
}

func newToken() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return hex.EncodeToString(token), nil
}
