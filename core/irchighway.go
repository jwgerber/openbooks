package core

import (
	"fmt"
	"strings"

	"github.com/evan-buss/openbooks/irc"
)

// Book channel commands. Originally written for irc.irchighway.net #ebooks,
// they work on any network that uses the @search / !Server conventions.

// NormalizeChannel returns the bare channel name without leading '#' or whitespace.
func NormalizeChannel(name string) string {
	return strings.TrimLeft(strings.TrimSpace(name), "#")
}

// Join connects to the IRC server. The channel is joined later, when the
// server's 001 welcome arrives (see JoinOnWelcome): some networks, like
// Undernet, reject JOIN with "451 Register first" before registration ends.
func Join(irc *irc.Conn, address string, enableTLS bool) error {
	return irc.Connect(address, enableTLS)
}

// JoinOnWelcome returns a Welcome handler that joins the given channel.
func JoinOnWelcome(irc *irc.Conn, channel string) HandlerFunc {
	return func(_ string) {
		irc.JoinChannel(NormalizeChannel(channel))
	}
}

// SearchBook sends a search query to the search bot
func SearchBook(irc *irc.Conn, searchBot string, query string) {
	searchBot = strings.TrimPrefix(searchBot, "@")
	irc.SendMessage(fmt.Sprintf("@%s %s", searchBot, query))
}

// DownloadBook sends the book string to the download bot
func DownloadBook(irc *irc.Conn, book string) {
	irc.SendMessage(book)
}

// Send a CTCP Version response
func SendVersionInfo(irc *irc.Conn, line string, version string) {
	// Line format is like ":messager PRIVMSG #channel: message"
	// we just want the messager without the colon
	sender := strings.Split(line, " ")[0][1:]
	// TODO: Figure out if there's an automated way to adjust this...
	irc.SendNotice(sender, fmt.Sprintf("\x01%s\x01", version))
}
