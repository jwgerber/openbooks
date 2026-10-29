package core

import "testing"

func book(server, author, title, format, full string) BookDetail {
	if full == "" {
		full = "!" + server + " " + author + " - " + title + "." + format
	}
	return BookDetail{Server: server, Author: author, Title: title, Format: format, Full: full}
}

func defaultOpts(query string) RankOptions {
	return RankOptions{Query: query, FormatOrder: DefaultFormatOrder}
}

func titles(books []BookDetail) []string {
	out := make([]string, len(books))
	for i, b := range books {
		out[i] = b.Title + "." + b.Format
	}
	return out
}

func TestRank_MatchBeatsFormat(t *testing.T) {
	ranked := RankBooks([]BookDetail{
		book("A", "P. L. Travers", "Mary Poppins", "epub", ""),
		book("B", "Andy Weir", "Project Hail Mary", "pdf", ""),
	}, defaultOpts("project hail mary"))

	if ranked[0].Title != "Project Hail Mary" {
		t.Errorf("got order %v", titles(ranked))
	}
}

func TestRank_FormatOrderWithinSameMatch(t *testing.T) {
	ranked := RankBooks([]BookDetail{
		book("A", "Andy Weir", "Project Hail Mary", "pdf", ""),
		book("A", "Andy Weir", "Project Hail Mary", "mobi", ""),
		book("A", "Andy Weir", "Project Hail Mary", "epub", ""),
		book("A", "Andy Weir", "Project Hail Mary", "azw3", ""),
	}, defaultOpts("project hail mary"))

	want := []string{"epub", "azw3", "mobi", "pdf"}
	for i, f := range want {
		if ranked[i].Format != f {
			t.Fatalf("got order %v, want formats %v", titles(ranked), want)
		}
	}
}

func TestRank_CustomFormatOrder(t *testing.T) {
	ranked := RankBooks([]BookDetail{
		book("A", "Andy Weir", "Project Hail Mary", "epub", ""),
		book("A", "Andy Weir", "Project Hail Mary", "pdf", ""),
	}, RankOptions{Query: "project hail mary", FormatOrder: []string{"pdf", "epub"}})

	if ranked[0].Format != "pdf" {
		t.Errorf("got order %v", titles(ranked))
	}
}

func TestRank_RetailBoostScanPenalty(t *testing.T) {
	ranked := RankBooks([]BookDetail{
		book("A", "Andy Weir", "Project Hail Mary", "epub", "!A Andy Weir - Project Hail Mary [scan].epub"),
		book("B", "Andy Weir", "Project Hail Mary", "epub", "!B Andy Weir - Project Hail Mary OCR v2.epub"),
		book("C", "Andy Weir", "Project Hail Mary", "epub", "!C Andy Weir - Project Hail Mary.epub"),
		book("D", "Andy Weir", "Project Hail Mary", "epub", "!D Andy Weir - Project Hail Mary (Retail).epub"),
	}, defaultOpts("project hail mary"))

	want := []string{"D", "C", "A", "B"}
	for i, s := range want {
		if ranked[i].Server != s {
			t.Fatalf("got servers %v, want %v", []string{ranked[0].Server, ranked[1].Server, ranked[2].Server, ranked[3].Server}, want)
		}
	}
}

func TestRank_OnlineTiebreak(t *testing.T) {
	opts := defaultOpts("project hail mary")
	opts.Online = map[string]bool{"oatmeal": true}
	ranked := RankBooks([]BookDetail{
		book("Wench", "Andy Weir", "Project Hail Mary", "epub", ""),
		book("Oatmeal", "Andy Weir", "Project Hail Mary", "epub", ""),
	}, opts)

	if ranked[0].Server != "Oatmeal" {
		t.Errorf("online server not first: %s", ranked[0].Server)
	}
}

func TestRank_NormalizesPunctuationAndDiacritics(t *testing.T) {
	ranked := RankBooks([]BookDetail{book("A", "Orson Scott Card", "Ender's Game", "epub", "")}, defaultOpts("enders game"))
	if got := ranked[0].Score / 1000; got != 10 {
		t.Errorf("ender's game match = %d, want 10", got)
	}

	ranked = RankBooks([]BookDetail{book("A", "Charlotte Brontë", "Jane Eyre", "epub", "")}, defaultOpts("jane eyre bronte"))
	if got := ranked[0].Score / 1000; got != 10 {
		t.Errorf("bronte match = %d, want 10", got)
	}
}

func TestRank_UnlistedFormatsLastNotDropped(t *testing.T) {
	ranked := RankBooks([]BookDetail{
		book("A", "Andy Weir", "Project Hail Mary", "rar", ""),
		book("A", "Andy Weir", "Project Hail Mary", "", ""),
		book("A", "Andy Weir", "Project Hail Mary", "txt", ""),
		book("A", "Andy Weir", "Project Hail Mary", "epub", ""),
	}, defaultOpts("project hail mary"))

	if len(ranked) != 4 || ranked[0].Format != "epub" {
		t.Errorf("got %v", titles(ranked))
	}
}

func TestRank_StableTies(t *testing.T) {
	ranked := RankBooks([]BookDetail{
		book("First", "Andy Weir", "Project Hail Mary", "epub", ""),
		book("Second", "Andy Weir", "Project Hail Mary", "epub", ""),
	}, defaultOpts("project hail mary"))

	if ranked[0].Server != "First" || ranked[1].Server != "Second" {
		t.Errorf("tie order changed: %s, %s", ranked[0].Server, ranked[1].Server)
	}
}

func TestRank_BookzFixture(t *testing.T) {
	books, _, err := ParseSearchFile("testdata/bookz/search_results.txt")
	if err != nil {
		t.Fatal(err)
	}
	ranked := RankBooks(books, defaultOpts("project hail mary"))

	if ranked[0].Format != "epub" {
		t.Errorf("top pick is %s (%s), want an epub", ranked[0].Format, ranked[0].Full)
	}
}
