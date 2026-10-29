package server

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"path/filepath"

	"github.com/evan-buss/openbooks/irc"
)

func (server *server) currentSettings() Settings {
	server.settingsMu.RLock()
	defer server.settingsMu.RUnlock()
	return server.settings
}

// newIrcConn creates an IRC connection using the current nickname setting,
// or a random one when it is empty.
func (server *server) newIrcConn() *irc.Conn {
	name := server.currentSettings().UserName
	if name == "" {
		name = fmt.Sprintf("reader%04d", rand.Intn(10000))
	}
	return irc.New(name, server.config.UserAgent)
}

func (server *server) getSettingsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(server.currentSettings())
	}
}

func (server *server) putSettingsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requested Settings
		if err := json.NewDecoder(r.Body).Decode(&requested); err != nil {
			writeSettingsError(w, "Settings must be valid JSON.")
			return
		}
		settings, err := requested.Validate()
		if err != nil {
			writeSettingsError(w, err.Error())
			return
		}
		settings.DownloadDir = server.config.DownloadDir

		if server.config.ConfigDir != "" {
			if err := SaveSettings(filepath.Join(server.config.ConfigDir, settingsFileName), settings); err != nil {
				server.log.Printf("Unable to save settings: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Unable to save settings file."})
				return
			}
		}

		server.settingsMu.Lock()
		server.settings = settings
		server.settingsMu.Unlock()
		server.log.Printf("Settings updated: server=%s channel=#%s tls=%v", settings.Server, settings.Channel, settings.EnableTLS)

		for _, client := range server.clientList() {
			go client.reconnect(server)
		}

		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)
	}
}

func writeSettingsError(w http.ResponseWriter, message string) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// reconnect drops the client's IRC session and starts a new one with the
// current settings. Closing the old connection ends its reader goroutine.
func (c *Client) reconnect(server *server) {
	c.log.Println("Reconnecting to IRC with new settings.")
	c.irc.Disconnect()
	c.irc = server.newIrcConn()
	c.startIrcConnection(server)

	// The browser may have left while we were dialing; don't leave an
	// orphaned IRC session holding the nickname.
	if _, ok := server.lookupClient(c.uuid); !ok {
		c.log.Println("Client left during reconnect; closing IRC session.")
		c.irc.Disconnect()
	}
}
