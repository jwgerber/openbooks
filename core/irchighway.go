package core

import (
	"fmt"
	"strings"
	"time"

	"github.com/evan-buss/openbooks/irc"
)

// Book channel commands. Originally written for irc.irchighway.net #ebooks,
// they work on any network that uses the @search / !Server conventions.

// NormalizeChannel returns the bare channel name without leading '#' or whitespace.
func NormalizeChannel(name string) string {
	return strings.TrimLeft(strings.TrimSpace(name), "#")
}

// Join connects to the IRC server and joins the given channel
func Join(irc *irc.Conn, address string, enableTLS bool, channel string) error {
	err := irc.Connect(address, enableTLS)
	if err != nil {
		return err
	}
	// Wait before joining the channel
	// Often you recieve a private message from the server
	time.Sleep(time.Second * 2)
	irc.JoinChannel(NormalizeChannel(channel))
	return nil
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
