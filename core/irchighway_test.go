package core

import "testing"

func TestNormalizeChannel(t *testing.T) {
	cases := map[string]string{
		"#bookz":     "bookz",
		"bookz":      "bookz",
		"  ##bookz ": "bookz",
		"#ebooks":    "ebooks",
	}
	for input, want := range cases {
		if got := NormalizeChannel(input); got != want {
			t.Errorf("NormalizeChannel(%q) = %q, want %q", input, got, want)
		}
	}
}
