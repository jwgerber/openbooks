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
