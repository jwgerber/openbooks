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
	stopWords = map[string]bool{"the": true, "a": true, "an": true}
	// Release tags in titles that shouldn't count as extra words.
	releaseTags = map[string]bool{
		"retail": true, "epub": true, "mobi": true, "azw3": true, "azw": true,
		"pdf": true, "rtf": true, "txt": true, "ebook": true, "scan": true, "ocr": true,
	}
	versionToken = regexp.MustCompile(`^v\d+$`)
	scanPattern  = regexp.MustCompile(`(?i)\b(scan|ocr)\b`)
	oldVersion   = regexp.MustCompile(`(?i)\bv[1-4](\.\d+)?\b`)
	retailMarker = "retail"
)

// RankBooks sets Score on every book and returns them sorted by Score desc,
// ties kept in input order. Never drops a book.
//
// Score = match*10000 + format*1000 + closeness*100 + (quality+3)*10 + online,
// so each factor only breaks ties of the one before it: how many query words
// the book has, format preference, how few extra words it has, release
// quality, then whether the server is online.
func RankBooks(books []BookDetail, opts RankOptions) []BookDetail {
	queryTokens := tokenize(opts.Query)
	ranked := make([]BookDetail, len(books))
	for i, b := range books {
		match, closeness := matchScores(queryTokens, b)
		b.Score = match*10000 +
			formatScore(b.Format, opts.FormatOrder)*1000 +
			closeness*100 +
			(qualityScore(b.Full)+3)*10 +
			onlineScore(b.Server, opts.Online)
		ranked[i] = b
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].Score > ranked[j].Score })
	return ranked
}

// matchScores returns the share of query words found in the author and
// title (match, 0..10) and the share of the book's words that were asked for
// (closeness, 0..9). Closeness ranks "Jane Austen - Pride and Prejudice"
// above "(Jane Austen 01) - First Impressions (Pride and Prejudice)".
func matchScores(queryTokens []string, b BookDetail) (match, closeness int) {
	if len(queryTokens) == 0 {
		return 0, 0
	}
	have := map[string]bool{}
	for _, tok := range tokenize(b.Author + " " + b.Title) {
		if !releaseTags[tok] && !versionToken.MatchString(tok) {
			have[tok] = true
		}
	}
	wanted := map[string]bool{}
	for _, tok := range queryTokens {
		wanted[tok] = true
	}

	found := 0
	for tok := range wanted {
		if have[tok] {
			found++
		}
	}
	match = roundedShare(found, len(wanted), 10)
	if len(have) > 0 {
		closeness = roundedShare(found, len(have), 9)
	}
	return match, closeness
}

// roundedShare returns part/whole scaled to 0..scale, rounded to nearest.
func roundedShare(part, whole, scale int) int {
	return (part*scale*2 + whole) / (whole * 2)
}

// formatScore is 9 for the most preferred format, counting down; formats past
// the ninth, and unlisted ones, score 0 so they can't outweigh a match step.
func formatScore(format string, order []string) int {
	format = strings.ToLower(format)
	for i, f := range order {
		if f == format {
			if score := 9 - i; score > 0 {
				return score
			}
			return 0
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
