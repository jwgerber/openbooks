package core

import (
	"reflect"
	"testing"
)

func TestCaseInsensitiveSort(t *testing.T) {
	cases := []struct {
		input string
		want  IrcServers
	}{
		{
			"+FWServer ~Oatmeal +LawdyServer +fwServer evan",
			IrcServers{
				ElevatedUsers: []string{"FWServer", "fwServer", "LawdyServer", "Oatmeal"},
				RegularUsers:  []string{"evan"},
			},
		},
		{"",
			IrcServers{
				ElevatedUsers: []string{},
				RegularUsers:  []string{},
			},
		},
	}

	for _, v := range cases {
		result := ParseServers(v.input)
		if !reflect.DeepEqual(result.ElevatedUsers, v.want.ElevatedUsers) {
			t.Errorf("got %#v, want %#v", result.ElevatedUsers, v.want.ElevatedUsers)
		}

		if !reflect.DeepEqual(result.RegularUsers, v.want.RegularUsers) {
			t.Errorf("got %#v, want %#v", result.RegularUsers, v.want.RegularUsers)
		}
	}
}

func TestOnlineSet_CaseInsensitive(t *testing.T) {
	online := ParseServers("@Oatmeal +Pondering regularguy").OnlineSet()
	for _, nick := range []string{"oatmeal", "pondering", "regularguy"} {
		if !online[nick] {
			t.Errorf("%q missing from online set %v", nick, online)
		}
	}
}
