package core

import (
	"bufio"
	"context"
	"log"
	"strings"

	"github.com/evan-buss/openbooks/irc"
)

type event int

const (
	noOp           = event(0)
	Message        = event(1)
	SearchResult   = event(2)
	BookResult     = event(3)
	NoResults      = event(4)
	BadServer      = event(5)
	SearchAccepted = event(6)
	MatchesFound   = event(7)
	ServerList     = event(8)
	Ping           = event(9)
	Version        = event(10)
	Welcome        = event(11)
	NickInUse      = event(12)
)

// Unique identifiers found in the message for various different events.
const (
	pingMessage            = "PING"
	sendMessage            = "DCC SEND"
	noticeMessage          = "NOTICE"
	noResults              = "Sorry"
	serverUnavailable      = "try another server"
	searchAccepted         = "has been accepted"
	searchResultIdentifier = "_results_for"
	numMatches             = "matches"
	beginUserList          = "353"
	endUserList            = "366"
	versionInquiry         = "\x01VERSION\x01"
	welcomeNumeric         = "001"
	nickInUseNumeric       = "433"
	nickUnavailableNumeric = "437"
)

// userListPart is an internal event for a 353 NAMES line; StartReader
// accumulates these and emits ServerList on the closing 366.
const userListPart = event(-1)

// classify determines which event a raw IRC line represents.
func classify(text string) event {
	// Numerics are matched on the command field only: chat lines often
	// contain numbers like "1366" that must not end a NAMES list.
	if fields := strings.Fields(text); len(fields) > 1 {
		switch fields[1] {
		case welcomeNumeric:
			return Welcome
		case nickInUseNumeric, nickUnavailableNumeric:
			return NickInUse
		case beginUserList:
			return userListPart
		case endUserList:
			return ServerList
		}
	}

	switch {
	case strings.Contains(text, sendMessage):
		if strings.Contains(text, searchResultIdentifier) {
			return SearchResult
		}
		return BookResult
	case strings.Contains(text, noticeMessage):
		switch {
		case strings.Contains(text, noResults):
			return NoResults
		case strings.Contains(text, serverUnavailable):
			return BadServer
		case strings.Contains(text, searchAccepted):
			return SearchAccepted
		case strings.Contains(text, numMatches):
			return MatchesFound
		}
	case strings.Contains(text, pingMessage):
		return Ping
	case strings.Contains(text, versionInquiry):
		return Version
	}
	return noOp
}

type HandlerFunc func(text string)
type EventHandler map[event]HandlerFunc

func StartReader(ctx context.Context, irc *irc.Conn, handler EventHandler) {
	var users UserList
	scanner := bufio.NewScanner(irc)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			text := scanner.Text()
			if err := scanner.Err(); err != nil {
				log.Println(err)
			}

			// Send raw message if they want to recieve it (logging purposes)
			if invoke, ok := handler[Message]; ok {
				invoke(text)
			}

			event := classify(text)
			switch event {
			case MatchesFound:
				start := strings.LastIndex(text, "returned") + 9
				end := strings.LastIndex(text, "matches") - 1
				text = text[start:end]
			case userListPart:
				users.Add(text)
				event = noOp
			case ServerList:
				text = users.String()
				users.Reset()
			}

			if invoke, ok := handler[event]; ok {
				go invoke(text)
			}
		}
	}
}
