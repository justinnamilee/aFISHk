package setup

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"afishk/internal/config"
	"afishk/internal/privatefs"
)

// Save writes a new configuration and a private random token; never overwrites.
func Save(c config.Config, path string) (string, error) {
	var err error
	if _, err = os.Lstat(path); !os.IsNotExist(err) {
		return "", fmt.Errorf("configuration already exists or cannot be accessed")
	}
	c.TokenFile = filepath.Join(filepath.Dir(path), "token")
	if err = c.Validate(); err != nil {
		return "", err
	}
	if err = privatefs.Dir(filepath.Dir(path)); err != nil {
		return "", err
	}
	// Always generate a token; never offer unauthenticated network access.
	token := make([]byte, 32)
	if _, err = rand.Read(token); err != nil {
		return "", err
	}
	tokenCreated := false
	if info, statErr := os.Lstat(c.TokenFile); statErr == nil {
		if !info.Mode().IsRegular() || privatefs.CheckFile(c.TokenFile) != nil {
			return "", fmt.Errorf("existing token must be a private regular file: %s", c.TokenFile)
		}
		data, readErr := os.ReadFile(c.TokenFile)
		if readErr != nil {
			return "", readErr
		}
		if len(strings.TrimSpace(string(data))) < 32 {
			return "", fmt.Errorf("existing token is too short: %s", c.TokenFile)
		}
	} else if os.IsNotExist(statErr) {
		if err = writePrivate(c.TokenFile, []byte(hex.EncodeToString(token)+"\n")); err != nil {
			return "", err
		}
		tokenCreated = true
	} else {
		return "", statErr
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err == nil {
		err = writePrivate(path, append(data, '\n'))
	}
	if err != nil {
		if tokenCreated {
			_ = os.Remove(c.TokenFile)
		}
		return "", err
	}
	data, err = os.ReadFile(c.TokenFile)
	return strings.TrimSpace(string(data)), err
}

func writePrivate(path string, data []byte) error { return privatefs.WriteNew(path, data) }
