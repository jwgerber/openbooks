package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) (*server, string) {
	dir := t.TempDir()
	srv := New(Config{
		ConfigDir:   dir,
		DownloadDir: "/books",
		Server:      "irc.undernet.org:6667",
		Channel:     "bookz",
		SearchBot:   "search",
	})
	return srv, dir
}

func TestGetSettings_ReturnsDefaults(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := httptest.NewRecorder()
	srv.getSettingsHandler()(rec, httptest.NewRequest(http.MethodGet, "/settings", nil))

	var got Settings
	json.NewDecoder(rec.Body).Decode(&got)
	if rec.Code != http.StatusOK || got.Channel != "bookz" || got.FormatOrder[0] != "epub" || got.DownloadDir != "/books" {
		t.Errorf("code %d, settings %+v", rec.Code, got)
	}
}

func TestPutSettings_InvalidReturns400(t *testing.T) {
	srv, _ := newTestServer(t)
	body := `{"server":"no-port","channel":"bookz","searchBot":"search","formatOrder":["epub"]}`
	rec := httptest.NewRecorder()
	srv.putSettingsHandler()(rec, httptest.NewRequest(http.MethodPut, "/settings", strings.NewReader(body)))

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if rec.Code != http.StatusBadRequest || !strings.Contains(resp["error"], "host:port") {
		t.Errorf("code %d, body %v", rec.Code, resp)
	}
}

func TestPutSettings_PersistsAndReturnsSaved(t *testing.T) {
	srv, dir := newTestServer(t)
	body := `{"server":"irc.irchighway.net:6697","enableTls":true,"channel":"#ebooks","searchBot":"@search","formatOrder":["PDF","epub"],"downloadDir":"/etc"}`
	rec := httptest.NewRecorder()
	srv.putSettingsHandler()(rec, httptest.NewRequest(http.MethodPut, "/settings", strings.NewReader(body)))

	var got Settings
	json.NewDecoder(rec.Body).Decode(&got)
	if rec.Code != http.StatusOK || got.Channel != "ebooks" || got.FormatOrder[0] != "pdf" || got.DownloadDir != "/books" {
		t.Fatalf("code %d, settings %+v", rec.Code, got)
	}

	if _, err := os.Stat(filepath.Join(dir, "settings.json")); err != nil {
		t.Errorf("settings file not written: %v", err)
	}
	if reloaded := New(Config{ConfigDir: dir, DownloadDir: "/books"}).currentSettings(); reloaded.Server != "irc.irchighway.net:6697" {
		t.Errorf("settings not persisted across restart: %+v", reloaded)
	}
}
