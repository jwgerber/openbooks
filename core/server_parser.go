package core

import (
	"sort"
	"strings"
)

var prefixes = map[byte]struct{}{
	'~': {},
	'&': {},
	'@': {},
	'%': {},
	'+': {},
}

type IrcServers struct {
	ElevatedUsers []string `json:"elevatedUsers"`
	RegularUsers  []string `json:"regularUsers"`
}

// ParseServers parses the complete list of IRC users to get the elevated users which in
// this case are the download servers
func ParseServers(rawString string) IrcServers {
	allServers := strings.Split(rawString, " ")

	servers := IrcServers{
		ElevatedUsers: make([]string, 0),
		RegularUsers:  make([]string, 0),
	}

	for _, name := range allServers {
		if len(name) > 1 {
			if _, exists := prefixes[name[0]]; exists {
				servers.ElevatedUsers = append(servers.ElevatedUsers, name[1:])
			} else {
				servers.RegularUsers = append(servers.RegularUsers, name)
			}
		}
	}

	sort.Slice(servers.ElevatedUsers, ignoreCaseSort(servers.ElevatedUsers))
	sort.Slice(servers.RegularUsers, ignoreCaseSort(servers.RegularUsers))

	return servers
}

func ignoreCaseSort(items []string) func(i, j int) bool {
	return func(i, j int) bool {
		return strings.ToLower(items[i]) < strings.ToLower(items[j])
	}
}

// OnlineSet returns every nick in the channel, lowercased. On networks like
// Undernet many file servers are not opped, so any nick counts as online.
func (s IrcServers) OnlineSet() map[string]bool {
	online := make(map[string]bool, len(s.ElevatedUsers)+len(s.RegularUsers))
	for _, nick := range append(append([]string{}, s.ElevatedUsers...), s.RegularUsers...) {
		online[strings.ToLower(nick)] = true
	}
	return online
}

// UserList accumulates the nick lists of 353 NAMES lines. Large channels
// split NAMES across many lines, so only the part after " :" is kept.
type UserList struct {
	nicks strings.Builder
}

func (u *UserList) Add(namesLine string) {
	if i := strings.Index(namesLine, " :"); i >= 0 {
		u.nicks.WriteString(namesLine[i+2:])
		u.nicks.WriteString(" ")
	}
}

func (u *UserList) String() string { return u.nicks.String() }

func (u *UserList) Reset() { u.nicks.Reset() }
