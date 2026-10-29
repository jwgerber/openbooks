package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/evan-buss/openbooks/core"
	"github.com/evan-buss/openbooks/irc"
	"github.com/evan-buss/openbooks/util"
)

var servers []string

const clearLine = "\r\033[2K"

func registerShutdown(conn *irc.Conn, cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		conn.Disconnect()
		cancel()
		os.Exit(0)
	}()
}

// Connect to IRC server and save connection to Config
func instantiate(config *Config) {
	fmt.Printf("Connecting to %s.", config.Server)
	conn := irc.New(config.UserName, config.Version)
	config.irc = conn
	config.joined = make(chan struct{})
	err := core.Join(conn, config.Server, config.EnableTLS)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%sConnected to %s.\n", clearLine, config.Server)
}

// Required handlers are used regardless of what CLI mode is selected.
// Keep alive pings and other core IRC client features
func addEssentialHandlers(handler core.EventHandler, config *Config) {
	handler[core.Ping] = config.pingHandler
	handler[core.Version] = config.versionHandler
	joinChannel := core.JoinOnWelcome(config.irc, config.Channel)
	var once sync.Once
	handler[core.Welcome] = func(text string) {
		joinChannel(text)
		once.Do(func() { close(config.joined) })
	}
	handler[core.NickInUse] = func(_ string) {
		config.irc.ChangeNick(core.AlternateNick(config.irc.Username))
	}
	handler[core.ServerList] = func(text string) {
		servers = core.ParseServers(text).ElevatedUsers
	}
}

func (config *Config) setupLogger(handler core.EventHandler) io.Closer {
	logger, file, err := util.CreateLogFile(config.UserName, config.Dir)
	if err != nil {
		log.Fatalf("Error setting up logger: %s\n", err)
	}
	handler[core.Message] = func(text string) {
		logger.Println(text)
	}

	return file
}

// Show warning message if the server they are downloading from is not online.
func warnIfServerOffline(bookLine string) {
	for _, server := range servers {
		if strings.HasPrefix(bookLine[1:], server) {
			return
		}
	}

	fmt.Println("WARNING: That server is not online. Your request will never complete.")
}

func getLastSearchTime() time.Time {
	timestampFilePath := filepath.Join(os.TempDir(), ".openbooks")
	fileInfo, err := os.Stat(timestampFilePath)

	if errors.Is(err, os.ErrNotExist) {
		return time.Now()
	}

	return fileInfo.ModTime()
}

func setLastSearchTime() {
	timestampFilePath := filepath.Join(os.TempDir(), ".openbooks")
	_, err := os.Stat(timestampFilePath)

	if errors.Is(err, os.ErrNotExist) {
		os.Create(timestampFilePath)
	}

	os.Chtimes(timestampFilePath, time.Now(), time.Now())
}

// joinTimeout bounds how long one-shot commands wait for registration.
const joinTimeout = 60 * time.Second

// waitForJoin blocks until the channel has been joined. Messages sent before
// the server's 001 welcome are rejected ("451 Register first").
func (config *Config) waitForJoin() {
	select {
	case <-config.joined:
		// Give the JOIN a moment to complete before talking in the channel.
		time.Sleep(time.Second)
	case <-time.After(joinTimeout):
		log.Fatalf("Timed out joining #%s on %s.\n", core.NormalizeChannel(config.Channel), config.Server)
	}
}
