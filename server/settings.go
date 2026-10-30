package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/evan-buss/openbooks/core"
)

// Settings are the user-editable IRC and ranking options, persisted as JSON.
type Settings struct {
	Server      string   `json:"server"` // host:port
	EnableTLS   bool     `json:"enableTls"`
	Channel     string   `json:"channel"`     // bare, no '#'
	SearchBot   string   `json:"searchBot"`   // bare, no '@'
	UserName    string   `json:"userName"`    // "" = random
	FormatOrder []string `json:"formatOrder"` // lowercase, most preferred first
	DownloadDir string   `json:"downloadDir"` // read-only; ignored on PUT
}

const settingsFileName = "settings.json"

// LoadSettings reads settings from path, filling missing fields from
// defaults. A missing or unreadable file falls back to defaults.
func LoadSettings(path string, defaults Settings) Settings {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaults
	}
	if err != nil {
		log.Printf("SETTINGS: unable to read %s, using defaults: %v", path, err)
		return defaults
	}

	loaded := defaults
	loaded.FormatOrder = nil
	if err := json.Unmarshal(data, &loaded); err != nil {
		log.Printf("SETTINGS: %s is not valid JSON, using defaults: %v", path, err)
		return defaults
	}
	if loaded.FormatOrder == nil {
		loaded.FormatOrder = defaults.FormatOrder
	}
	loaded.DownloadDir = defaults.DownloadDir
	return loaded
}

// SaveSettings writes settings atomically (temp file, then rename).
func SaveSettings(path string, s Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), settingsFileName+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Validate returns a normalized copy of the settings, or an error naming
// the first invalid field.
func (s Settings) Validate() (Settings, error) {
	host, port, err := net.SplitHostPort(strings.TrimSpace(s.Server))
	if err != nil || host == "" {
		return s, fmt.Errorf("server must be host:port, e.g. irc.undernet.org:6667")
	}
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		return s, fmt.Errorf("server port must be between 1 and 65535")
	}
	s.Server = net.JoinHostPort(host, port)

	s.Channel = core.NormalizeChannel(s.Channel)
	if s.Channel == "" {
		return s, fmt.Errorf("channel is required")
	}

	s.SearchBot = strings.TrimLeft(strings.TrimSpace(s.SearchBot), "@")
	if s.SearchBot == "" {
		return s, fmt.Errorf("search trigger is required")
	}

	s.UserName = strings.TrimSpace(s.UserName)

	formats := make([]string, 0, len(s.FormatOrder))
	seen := map[string]bool{}
	for _, f := range s.FormatOrder {
		f = strings.ToLower(strings.TrimSpace(f))
		if f == "" {
			continue
		}
		if seen[f] {
			return s, fmt.Errorf("format %q is listed twice", f)
		}
		seen[f] = true
		formats = append(formats, f)
	}
	if len(formats) == 0 {
		return s, fmt.Errorf("format order needs at least one format")
	}
	s.FormatOrder = formats
	return s, nil
}
