package server

import (
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/evan-buss/openbooks/irc"
	"github.com/google/uuid"
)

func newTestClient() *Client {
	return &Client{
		uuid: uuid.New(),
		send: make(chan interface{}, 16),
		irc:  irc.New("tester", "test"),
		log:  log.New(os.Stdout, "TEST: ", 0),
	}
}

func TestNoResults_CompletesSearchWithEmptyResults(t *testing.T) {
	c := newTestClient()
	c.noResultsHandler("")

	resp, ok := (<-c.send).(SearchResponse)
	if !ok || len(resp.Books) != 0 || resp.MessageType != SEARCH {
		t.Errorf("want empty SearchResponse, got %#v", resp)
	}
}

func TestBadServer_EndsInFlightDownload(t *testing.T) {
	c := newTestClient()
	c.badServerHandler("")

	resp, ok := (<-c.send).(DownloadResponse)
	if !ok || resp.MessageType != DOWNLOAD || resp.DownloadPath != "" || resp.NotificationType != DANGER {
		t.Errorf("want failed DownloadResponse, got %#v", resp)
	}
}

func TestSearchBeforeJoin_IsRejectedNotLost(t *testing.T) {
	srv, _ := newTestServer(t)
	c := newTestClient()
	c.sendSearchRequest(&SearchRequest{Query: "dune"}, srv)

	resp, ok := (<-c.send).(StatusResponse)
	if !ok || resp.MessageType != RATELIMIT {
		t.Errorf("want RATELIMIT-type rejection (UI drops the history item), got %#v", resp)
	}
}

func TestReconnect_ClosedClientDoesNotPanic(t *testing.T) {
	srv, _ := newTestServer(t)
	srv.settings.Server = "127.0.0.1:1" // connection refused → error response
	c := newTestClient()
	c.markClosed()
	close(c.send)

	c.reconnect(srv) // must not panic with "send on closed channel"
}

func TestReconnect_DisconnectsIfClientLeftMeanwhile(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	srv, _ := newTestServer(t)
	srv.settings.Server = listener.Addr().String()
	c := newTestClient() // never registered with srv: it already left

	go c.reconnect(srv)
	conn, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 512)
	for {
		if _, err := conn.Read(buf); err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				t.Fatal("orphaned IRC connection was left open")
			}
			return // closed by the server side: good
		}
	}
}

func TestReconnect_DisconnectsIfReplacedBySameBrowser(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	srv, _ := newTestServer(t)
	srv.settings.Server = listener.Addr().String()
	c := newTestClient()
	// A reload registers a new *Client under the same cookie UUID.
	replacement := newTestClient()
	replacement.uuid = c.uuid
	srv.clients[c.uuid] = replacement

	go c.reconnect(srv)
	conn, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 512)
	for {
		if _, err := conn.Read(buf); err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				t.Fatal("old client's IRC session left open after a reload replaced it")
			}
			return
		}
	}
}

func TestDownloadBeforeJoin_EndsInFlightDownload(t *testing.T) {
	c := newTestClient()
	c.sendDownloadRequest(&DownloadRequest{Book: "!Bsk Some Book.epub"})

	resp, ok := (<-c.send).(DownloadResponse)
	if !ok || resp.MessageType != DOWNLOAD || resp.DownloadPath != "" {
		t.Errorf("want failed DownloadResponse, got %#v", resp)
	}
}
