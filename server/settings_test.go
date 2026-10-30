package server

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var testDefaults = Settings{
	Server:      "irc.undernet.org:6667",
	Channel:     "bookz",
	SearchBot:   "search",
	FormatOrder: []string{"epub", "azw3", "mobi", "pdf"},
	DownloadDir: "/books",
}

func TestLoadSettings_MissingFileUsesDefaults(t *testing.T) {
	got := LoadSettings(filepath.Join(t.TempDir(), "settings.json"), testDefaults)
	if !reflect.DeepEqual(got, testDefaults) {
		t.Errorf("got %+v, want defaults", got)
	}
}

func TestLoadSettings_CorruptFileFallsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte("{not json"), 0o644)

	got := LoadSettings(path, testDefaults)
	if !reflect.DeepEqual(got, testDefaults) {
		t.Errorf("got %+v, want defaults", got)
	}
}

func TestLoadSettings_PartialFileFillsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte(`{"channel":"ebooks"}`), 0o644)

	got := LoadSettings(path, testDefaults)
	want := testDefaults
	want.Channel = "ebooks"
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestSaveThenLoad_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	saved := testDefaults
	saved.EnableTLS = true
	saved.UserName = "josh"
	saved.FormatOrder = []string{"pdf", "epub"}

	if err := SaveSettings(path, saved); err != nil {
		t.Fatal(err)
	}
	if got := LoadSettings(path, testDefaults); !reflect.DeepEqual(got, saved) {
		t.Errorf("got %+v, want %+v", got, saved)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		edit    func(*Settings)
		wantErr bool
	}{
		{"valid defaults", func(s *Settings) {}, false},
		{"server without port", func(s *Settings) { s.Server = "irc.undernet.org" }, true},
		{"port out of range", func(s *Settings) { s.Server = "host:99999" }, true},
		{"channel only hash", func(s *Settings) { s.Channel = "#" }, true},
		{"empty search bot", func(s *Settings) { s.SearchBot = "@" }, true},
		{"empty format order", func(s *Settings) { s.FormatOrder = []string{} }, true},
		{"duplicate format", func(s *Settings) { s.FormatOrder = []string{"epub", "EPUB"} }, true},
	}
	for _, c := range cases {
		s := testDefaults
		s.FormatOrder = append([]string{}, testDefaults.FormatOrder...)
		c.edit(&s)
		if _, err := s.Validate(); (err != nil) != c.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", c.name, err, c.wantErr)
		}
	}
}

func TestValidate_Normalizes(t *testing.T) {
	s := testDefaults
	s.Channel = "#bookz"
	s.SearchBot = "@search"
	s.FormatOrder = []string{"EPUB", " pdf"}

	got, err := s.Validate()
	if err != nil {
		t.Fatal(err)
	}
	if got.Channel != "bookz" || got.SearchBot != "search" || !reflect.DeepEqual(got.FormatOrder, []string{"epub", "pdf"}) {
		t.Errorf("not normalized: %+v", got)
	}
}
