package core

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

func TestBookzFixture_ParsesResults(t *testing.T) {
	books, _, err := ParseSearchFile("testdata/bookz/search_results.txt")
	if err != nil {
		t.Fatal(err)
	}

	// 30 result lines, 13 of which are mp3 audiobook parts (not ebooks).
	const ebookLines = 17
	if len(books)*10 < ebookLines*9 {
		t.Errorf("parsed %d books, want at least 90%% of %d ebook lines", len(books), ebookLines)
	}
	for _, b := range books {
		if b.Server == "" || b.Title == "" || b.Format == "" || !strings.HasPrefix(b.Full, "!") {
			t.Errorf("incomplete book: %+v", b)
		}
	}
}

func TestBookzFixture_ReaderEvents(t *testing.T) {
	file, err := os.Open("testdata/bookz/irc_log.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	seen := map[event]bool{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		seen[classify(scanner.Text())] = true
	}

	for _, want := range []event{Welcome, SearchResult, BookResult, NoResults, SearchAccepted, MatchesFound, ServerList, Ping} {
		if !seen[want] {
			t.Errorf("event %d never classified from bookz log", want)
		}
	}
}

func TestUserList_MultiLineNamesKeepsEveryNick(t *testing.T) {
	var users UserList
	users.Add(":dallas.tx.us.undernet.org 353 me = #bookz :me reader1 @Bsk +Horla")
	users.Add(":dallas.tx.us.undernet.org 353 me = #bookz :reader2 +Oatmeal")

	online := ParseServers(users.String()).OnlineSet()
	for _, nick := range []string{"horla", "reader2", "bsk", "oatmeal", "reader1"} {
		if !online[nick] {
			t.Errorf("%q missing from online set %v", nick, online)
		}
	}
	if online["dallas.tx.us.undernet.org"] || online[":dallas.tx.us.undernet.org"] {
		t.Errorf("server prefix leaked into nick list: %v", online)
	}
}
