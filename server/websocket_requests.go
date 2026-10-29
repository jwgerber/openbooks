package server

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/evan-buss/openbooks/core"
	"github.com/evan-buss/openbooks/util"
)

// RequestHandler defines a generic handle() method that is called when a specific request type is made
type RequestHandler interface {
	handle(c *Client)
}

// messageRouter is used to parse the incoming request and respond appropriately
func (server *server) routeMessage(message Request, c *Client) {
	var obj interface{}

	switch message.MessageType {
	case SEARCH:
		obj = new(SearchRequest)
	case DOWNLOAD:
		obj = new(DownloadRequest)
	}

	err := json.Unmarshal(message.Payload, &obj)
	if err != nil {
		server.log.Printf("Invalid request payload. %s.\n", err.Error())
		c.trySend(StatusResponse{
			MessageType:      STATUS,
			NotificationType: DANGER,
			Title:            "Unknown request payload.",
		})
	}

	switch message.MessageType {
	case CONNECT:
		c.startIrcConnection(server)
	case SEARCH:
		c.sendSearchRequest(obj.(*SearchRequest), server)
	case DOWNLOAD:
		c.sendDownloadRequest(obj.(*DownloadRequest))
	default:
		server.log.Println("Unknown request type received.")
	}
}

// handle ConnectionRequests and either connect to the server or do nothing
func (c *Client) startIrcConnection(server *server) {
	server.repository.Reset()
	c.joined.Store(false)
	settings := server.currentSettings()
	err := core.Join(c.irc, settings.Server, settings.EnableTLS)
	if err != nil {
		c.log.Println(err)
		c.trySend(newErrorResponse("Unable to connect to IRC server."))
		return
	}

	handler := server.NewIrcEventHandler(c)

	if server.config.Log {
		logger, _, err := util.CreateLogFile(c.irc.Username, server.config.DownloadDir)
		if err != nil {
			server.log.Println(err)
		}
		handler[core.Message] = func(text string) { logger.Println(text) }
	}

	go core.StartReader(c.ctx, c.irc, handler)

	c.sendConnected("Welcome, connection established.")
}

// handle SearchRequests and send the query to the book server
func (c *Client) sendSearchRequest(s *SearchRequest, server *server) {
	// Until the 001 welcome arrives the channel isn't joined and the search
	// would be silently dropped. RATELIMIT makes the UI discard the attempt.
	if !c.joined.Load() {
		c.trySend(StatusResponse{
			MessageType:      RATELIMIT,
			NotificationType: WARNING,
			Title:            "Still connecting to the channel.",
			Detail:           "Try your search again in a few seconds.",
		})
		return
	}

	server.lastSearchMutex.Lock()
	defer server.lastSearchMutex.Unlock()

	nextAvailableSearch := server.lastSearch.Add(server.config.SearchTimeout)

	if time.Now().Before(nextAvailableSearch) {
		remainingSeconds := time.Until(nextAvailableSearch).Seconds()
		c.trySend(newRateLimitResponse(remainingSeconds))

		return
	}

	// Refresh NAMES so the online list used for ranking is current.
	settings := server.currentSettings()
	c.irc.GetUsers(settings.Channel)
	c.lastQuery = s.Query
	core.SearchBook(c.irc, settings.SearchBot, s.Query)
	server.lastSearch = time.Now()

	c.trySend(newStatusResponse(NOTIFY, "Search request sent."))
}

// handle DownloadRequests by sending the request to the book server
func (c *Client) sendDownloadRequest(d *DownloadRequest) {
	// Same as searches: before the JOIN the request would be silently dropped.
	if !c.joined.Load() {
		c.trySend(newFailedDownloadResponse("Still connecting to the channel. Try again in a few seconds."))
		return
	}

	core.DownloadBook(c.irc, d.Book)
	c.trySend(newStatusResponse(NOTIFY, "Download request received."))
}

// sendConnected tells the browser the current IRC nickname.
func (c *Client) sendConnected(title string) {
	c.trySend(ConnectionResponse{
		StatusResponse: StatusResponse{
			MessageType:      CONNECT,
			NotificationType: SUCCESS,
			Title:            title,
			Detail:           fmt.Sprintf("IRC username %s", c.irc.Username),
		},
		Name: c.irc.Username,
	})
}
