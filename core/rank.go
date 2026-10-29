package core

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// DefaultFormatOrder is the format preference used when none is configured.
var DefaultFormatOrder = []string{"epub", "azw3", "mobi", "pdf"}

// RankOptions controls how RankBooks scores search results.
type RankOptions struct {
	Query       string
	FormatOrder []string        // lowercase, most preferred first
	Online      map[string]bool // lowercase nicks
}

var (
	stopWords    = map[string]bool{"the": true, "a": true, "an": true}
	scanPattern  = regexp.MustCompile(`(?i)\b(scan|ocr)\b`)
	oldVersion   = regexp.MustCompile(`(?i)\bv[1-4](\.\d+)?\b`)
	retailMarker = "retail"
)

// RankBooks sets Score on every book and returns them sorted by Score desc,
// ties kept in input order. Never drops a book.
//
// Score = match*1000 + format*100 + (quality+3)*10 + online, so each factor
// only breaks ties of the one before it: title match, then format
// preference, then release quality, then whether the server is online.
func RankBooks(books []BookDetail, opts RankOptions) []BookDetail {
	queryTokens := tokenize(opts.Query)
	ranked := make([]BookDetail, len(books))
	for i, b := range books {
		b.Score = matchScore(queryTokens, b)*1000 +
			formatScore(b.Format, opts.FormatOrder)*100 +
			(qualityScore(b.Full)+3)*10 +
			onlineScore(b.Server, opts.Online)
		ranked[i] = b
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].Score > ranked[j].Score })
	return ranked
}

// matchScore is the share of query tokens found in the author and title, 0..10.
func matchScore(queryTokens []string, b BookDetail) int {
	if len(queryTokens) == 0 {
		return 0
	}
	have := map[string]bool{}
	for _, tok := range tokenize(b.Author + " " + b.Title) {
		have[tok] = true
	}
	found := 0
	for _, tok := range queryTokens {
		if have[tok] {
			found++
		}
	}
	return (found*10 + len(queryTokens)/2) / len(queryTokens)
}

func formatScore(format string, order []string) int {
	format = strings.ToLower(format)
	for i, f := range order {
		if f == format {
			return len(order) - i
		}
	}
	return 0
}

// qualityScore rewards retail releases and penalizes scans, OCR and early versions.
func qualityScore(full string) int {
	score := 0
	if strings.Contains(strings.ToLower(full), retailMarker) {
		score += 2
	}
	if scanPattern.MatchString(full) {
		score -= 2
	}
	if oldVersion.MatchString(full) {
		score--
	}
	return score
}

func onlineScore(server string, online map[string]bool) int {
	if online[strings.ToLower(server)] {
		return 1
	}
	return 0
}

// tokenize lowercases, strips diacritics and punctuation, and drops stop words.
func tokenize(s string) []string {
	s = strings.ReplaceAll(strings.ToLower(s), "'", "")
	s = strings.ReplaceAll(s, "’", "")
	var b strings.Builder
	for _, r := range norm.NFD.String(s) {
		switch {
		case unicode.Is(unicode.Mn, r):
			continue
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		default:
			b.WriteRune(' ')
		}
	}
	var tokens []string
	for _, tok := range strings.Fields(b.String()) {
		if !stopWords[tok] {
			tokens = append(tokens, tok)
		}
	}
	return tokens
}
