package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/evan-buss/openbooks/core"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rs/cors"
)

type server struct {
	// Shared app configuration
	config *Config

	// Shared data
	repository *Repository

	// Registered clients.
	clients map[uuid.UUID]*Client

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	log *log.Logger

	// Mutex to guard the lastSearch timestamp
	lastSearchMutex sync.Mutex

	// The time the last search was performed. Used to rate limit searches.
	lastSearch time.Time

	// Guards settings, which the settings panel can change at runtime.
	settingsMu sync.RWMutex
	settings   Settings
}

// Config contains settings for server
type Config struct {
	Log                     bool
	Port                    string
	UserName                string
	Persist                 bool
	DownloadDir             string
	Basepath                string
	Server                  string
	Channel                 string
	EnableTLS               bool
	SearchTimeout           time.Duration
	SearchBot               string
	DisableBrowserDownloads bool
	UserAgent               string
	// Folder holding settings.json. Empty keeps settings in memory only.
	ConfigDir string
}

func New(config Config) *server {
	defaults := Settings{
		Server:      config.Server,
		EnableTLS:   config.EnableTLS,
		Channel:     core.NormalizeChannel(config.Channel),
		SearchBot:   config.SearchBot,
		UserName:    config.UserName,
		FormatOrder: core.DefaultFormatOrder,
		DownloadDir: config.DownloadDir,
	}
	settings := defaults
	if config.ConfigDir == "" {
		log.Println("SERVER: no config dir set; settings changes will not survive a restart.")
	} else {
		settings = LoadSettings(filepath.Join(config.ConfigDir, settingsFileName), defaults)
	}

	return &server{
		settings:   settings,
		repository: NewRepository(),
		config:     &config,
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[uuid.UUID]*Client),
		log:        log.New(os.Stdout, "SERVER: ", log.LstdFlags|log.Lmsgprefix),
	}
}

// Start instantiates the web server and opens the browser
func Start(config Config) {
	createBooksDirectory(config)
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)

	corsConfig := cors.Options{
		AllowCredentials: true,
		AllowedOrigins:   []string{"http://127.0.0.1:5173"},
		AllowedHeaders:   []string{"*"},
		AllowedMethods:   []string{"GET", "PUT", "DELETE"},
	}
	router.Use(cors.New(corsConfig).Handler)

	server := New(config)
	routes := server.registerRoutes()

	ctx, cancel := context.WithCancel(context.Background())
	go server.startClientHub(ctx)
	server.registerGracefulShutdown(cancel)
	router.Mount(config.Basepath, routes)

	server.log.Printf("Base Path: %s\n", config.Basepath)
	server.log.Printf("OpenBooks is listening on port %v", config.Port)
	server.log.Printf("Download Directory: %s\n", config.DownloadDir)
	server.log.Printf("Open http://localhost:%v%s in your browser.", config.Port, config.Basepath)
	server.log.Fatal(http.ListenAndServe(":"+config.Port, router))
}

// The client hub is to be run in a goroutine and handles management of
// websocket client registrations.
func (server *server) startClientHub(ctx context.Context) {
	for {
		select {
		case client := <-server.register:
			server.clients[client.uuid] = client
		case client := <-server.unregister:
			if _, ok := server.clients[client.uuid]; ok {
				_, cancel := context.WithCancel(client.ctx)
				close(client.send)
				cancel()
				delete(server.clients, client.uuid)
			}
		case <-ctx.Done():
			for _, client := range server.clients {
				_, cancel := context.WithCancel(client.ctx)
				close(client.send)
				cancel()
				delete(server.clients, client.uuid)
			}
			return
		}
	}
}

func (server *server) registerGracefulShutdown(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		server.log.Println("Graceful shutdown.")
		// Close the shutdown channel. Triggering all reader/writer WS handlers to close.
		cancel()
		time.Sleep(time.Second)
		os.Exit(0)
	}()
}

func createBooksDirectory(config Config) {
	err := os.MkdirAll(filepath.Join(config.DownloadDir, "books"), os.FileMode(0755))
	if err != nil {
		panic(err)
	}
}
