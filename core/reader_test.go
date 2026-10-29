package core

import "testing"

func TestClassify_WelcomeOnlyOnNumeric001(t *testing.T) {
	cases := map[string]event{
		":Capelle.NL.EU.Undernet.Org 001 rdr1975 :Welcome to the UnderNet IRC Network, rdr1975": Welcome,
		":someone!u@h PRIVMSG #bookz :I got 001 problems":                                       noOp,
	}
	for line, want := range cases {
		if got := classify(line); got != want {
			t.Errorf("classify(%q) = %v, want %v", line, got, want)
		}
	}
}

func TestClassify_NumericsMatchCommandFieldOnly(t *testing.T) {
	cases := map[string]event{
		":dallas.tx.us.undernet.org 353 me = #bookz :a b":                           userListPart,
		":dallas.tx.us.undernet.org 366 me #bookz :End of /NAMES list.":             ServerList,
		":x!u@h PRIVMSG #bookz :!Horla Anthology 1366 - Tales.epub":                 noOp,
		":reader3661!u@h JOIN #bookz":                                               noOp,
		":dallas.tx.us.undernet.org 433 * rdrtest42 :Nickname is already in use.":   NickInUse,
		":irc.example.net 437 * rdrtest42 :Nick/channel is temporarily unavailable": NickInUse,
	}
	for line, want := range cases {
		if got := classify(line); got != want {
			t.Errorf("classify(%q) = %v, want %v", line, got, want)
		}
	}
}

func TestAlternateNick_DiffersAndFitsNickLength(t *testing.T) {
	for _, nick := range []string{"rdrtest42", "averyveryverylongnick"} {
		alt := AlternateNick(nick)
		if alt == nick || len(alt) > 12 || len(alt) == 0 {
			t.Errorf("AlternateNick(%q) = %q", nick, alt)
		}
	}
}

func TestPingToken(t *testing.T) {
	cases := map[string]string{
		"PING :1102911567":           ":1102911567",
		":irc.example.org PING :abc": ":abc",
		"PING irc.irchighway.net":    "irc.irchighway.net",
	}
	for line, want := range cases {
		if got := PingToken(line); got != want {
			t.Errorf("PingToken(%q) = %q, want %q", line, got, want)
		}
	}
}
