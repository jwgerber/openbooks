# Bookz Ranking + Settings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the openbooks fork work against Undernet `#bookz` (or any network), rank search results for the user, and add a persisted settings panel.

**Architecture:** The Go backend gains three things: a configurable channel, a pure ranking function applied in `searchResultHandler` before results go over the websocket, and a JSON settings store with `GET/PUT /settings` that reconnects the IRC client on save. The React/Mantine frontend gains a settings drawer and a "Top picks" block above the existing `BookTable`.

**Tech Stack:** Go 1.19+ (chi, gorilla/websocket), React 18 + TypeScript + Mantine 5 + Redux Toolkit / RTK Query, Docker.

**Spec:** `docs/superpowers/specs/2026-10-29-bookz-ranking-settings-design.md`

## Global Constraints

- Keep the existing stack: Go backend, Mantine/Redux frontend, existing Dockerfile. No MUI, no rewrite.
- Default settings: server `irc.undernet.org:6667`, TLS off, channel `#bookz`, search trigger `@search`, nickname random (as today), format order `epub, azw3, mobi, pdf`, download folder `/books`.
- Container flags/env set **starting defaults only**; once `settings.json` exists, it wins.
- Persist is always on; browser downloads stay on by default (existing openbooks behaviour unchanged).
- Top picks = the 5 highest-scoring lines, **no deduplication**.
- The full results list is always reachable; ranking only reorders.
- IRCHighway `#ebooks` must still work by changing settings only.
- Ranking runs server-side; browser displays the order it receives.
- `go` is not installed on this Mac: run `brew install go` once before Task 1. Frontend has no test runner, so frontend tasks verify with `npm run build` (runs `tsc`) plus a manual check in `npm run dev`.

## Review Focus

1. Query vs. title punctuation/case/diacritics (`enders game` vs `Ender's Game`, `Brontë` vs `bronte`) should still score as a full match → Task 4 test `TestRank_NormalizesPunctuationAndDiacritics`.
2. Results with an unlisted or archive format (`rar`, `zip`, `txt`, empty) should rank after listed formats, never be dropped → Task 4 test `TestRank_UnlistedFormatsLastNotDropped`.
3. A missing, empty, or corrupt `settings.json` should log a warning and fall back to defaults, not crash on boot → Task 5 test `TestLoadSettings_CorruptFileFallsBack`.
4. A channel typed with or without `#` (`#bookz` / `bookz`) should join the same channel → Task 1 test `TestNormalizeChannel`.
5. A server nick that differs only in case from the NAMES list (`Oatmeal` vs `oatmeal`) should still count as online → Task 3 test `TestOnlineSet_CaseInsensitive`.

---

### Task 1: Configurable channel + Undernet defaults

**Files:**
- Modify: `core/irchighway.go` (`Join` gains `channel string`)
- Modify: `irc/irc.go` (`JoinChannel`, `SendMessage`, `GetUsers` take a channel name without `#`, so stop hard-coding the prefix in callers)
- Modify: `cmd/openbooks/main.go:40-43` (defaults + new `--channel` flag), `cmd/openbooks/util.go` (`bindGlobalServerFlags`), `server/server.go` (`Config.Channel`), `server/websocket_requests.go` (`startIrcConnection` passes the channel), `cli/` (wherever `core.Join` is called)
- Test: `core/irchighway_test.go`

**Interfaces:**
- Produces: `func NormalizeChannel(name string) string` in `core/irchighway.go`: trims whitespace and leading `#`s and returns the bare name. `func Join(irc *irc.Conn, address string, enableTLS bool, channel string) error`. `server.Config.Channel string`, stored bare (no `#`).

- [ ] **Step 1:** `brew install go`, then `go test ./...` and confirm the baseline passes. Note any pre-existing failures in the commit message rather than fixing them.
- [ ] **Step 2: Write failing test** `TestNormalizeChannel`: table cases `"#bookz"→"bookz"`, `"bookz"→"bookz"`, `"  ##bookz "→"bookz"`, `"#ebooks"→"ebooks"`.
- [ ] **Step 3:** `go test ./core -run TestNormalizeChannel`. Expected: FAIL (undefined).
- [ ] **Step 4:** Implement `NormalizeChannel` and thread `channel` through `Join` and all callers. Flag defaults: `--server irc.undernet.org:6667`, `--tls=false`, `--searchbot search`, `--channel bookz`. Update the `Join` doc comment and the cobra `Short` text, which says "IRCHighway".
- [ ] **Step 5:** `go test ./... && go build ./...`. Expected: PASS.
- [ ] **Step 6: Commit** `feat: configurable IRC channel, default to Undernet #bookz`

### Task 2: Merge upstream DCC hardening (PR #206)

**Files:** `core/file.go`, `dcc/dcc.go`, `dcc/dcc_test.go`, `mock/dcc_server_ack.go` (from the PR)

- [ ] **Step 1:** `git fetch upstream pull/206/head:pr-206 && git merge --no-ff pr-206`. Resolve conflicts in favour of the PR, but keep any change from Task 1.
- [ ] **Step 2:** `go test ./dcc ./core`. Expected: PASS, including the PR's new tests.
- [ ] **Step 3:** The merge commit is the commit. Message: `merge: upstream PR #206 DCC download hardening`.

### Task 3: Capture real #bookz traffic; fix reader/parser/online detection

This task is the reality check for the spec's "bookz support is a goal" call. The fixtures it captures become test data for Task 4.

**Files:**
- Create: `core/testdata/bookz/search_results.txt` (one real unzipped result file, trimmed to ~200 lines), `core/testdata/bookz/irc_log.txt` (raw lines: welcome, JOIN, NAMES 353/366, search accepted, matches found, no results, DCC SEND for results and for a book)
- Modify: `core/reader.go`, `core/search_parser.go`, `core/server_parser.go` (only where the fixtures show differences)
- Modify: `server/irc_events.go` (`userListHandler`), `server/websocket_requests.go` (`sendSearchRequest`), `server/repository.go`, `server/app/src/state/api.ts` (`getServers` transform)
- Test: `core/bookz_fixture_test.go`, `core/server_parser_test.go`

**Interfaces:**
- Produces: `func (s IrcServers) OnlineSet() map[string]bool` in `core/server_parser.go`. It returns lowercase nicks from `ElevatedUsers` and `RegularUsers` together. Also `func (r *Repository) Reset()`.

- [ ] **Step 1:** Run `go run ./cmd/openbooks server --log --dir ./tmp-capture` (Task 1 defaults point at Undernet `#bookz`). In the UI, search `project hail mary`, download one small result, then search `zzqxnonexistent`. Copy the raw log lines from the log file in `./tmp-capture` and the unzipped results file into `core/testdata/bookz/`. Strip other users' hostmasks to nicks only. Do not commit `tmp-capture/`; add it to `.gitignore`.
- [ ] **Step 2: Write failing tests** in `core/bookz_fixture_test.go`:
  - `TestBookzFixture_ParsesResults`: `ParseSearchFile("testdata/bookz/search_results.txt")` returns ≥90% of non-header lines as books, each with non-empty `Server`, `Title`, `Format`, and `Full` starting with `!`.
  - `TestBookzFixture_ReaderEvents`: feed `irc_log.txt` through the reader's classification and assert each of `SearchResult`, `BookResult`, `NoResults`, `SearchAccepted`, `MatchesFound`, `ServerList` fires at least once. To make this testable, extract the classification into `func classify(text string) event`, and keep the 353/366 accumulation in `StartReader`.
  - `TestOnlineSet_CaseInsensitive`: `ParseServers("@Oatmeal +Pondering regularguy").OnlineSet()` contains `oatmeal`, `pondering`, `regularguy`.
- [ ] **Step 3:** `go test ./core`. Expected: FAIL. Any failure beyond the missing `classify`/`OnlineSet` is a real bookz difference.
- [ ] **Step 4:** Fix each difference the fixtures reveal in `reader.go`/`search_parser.go` by adding alternatives, never replacing the IRCHighway strings (Global Constraint: `#ebooks` must keep working). Implement `classify`, `OnlineSet`, and `Repository.Reset`. In `sendSearchRequest`, call `c.irc.GetUsers(channel)` right before `core.SearchBook` so the online list is fresh. Call `repository.Reset()` in `startIrcConnection` before joining. Change the `api.ts` `getServers` transform to return elevated and regular users together, so the UI's online dot matches the server-side definition.
- [ ] **Step 5:** `go test ./...` and `cd server/app && npm run build`. Expected: PASS.
- [ ] **Step 6: Commit** `feat: support #bookz traffic, treat any channel nick as online`. Commit the fixture files with it.

### Task 4: Ranking

**Files:**
- Create: `core/rank.go`, `core/rank_test.go`
- Modify: `core/search_parser.go` (`BookDetail` gains `Score int \`json:"score"\``; remove the `sort.Slice` by server in both `ParseSearch` and `ParseSearchV2`, because ranking now owns order)
- Modify: `server/irc_events.go` (`searchResultHandler` ranks before `newSearchResponse`)

**Interfaces:**
- Consumes: `IrcServers.OnlineSet()` (Task 3). The current format order comes from settings (Task 5). Until Task 5 lands, use `core.DefaultFormatOrder`.
- Produces:
  ```go
  var DefaultFormatOrder = []string{"epub", "azw3", "mobi", "pdf"}
  type RankOptions struct {
      Query       string
      FormatOrder []string        // lowercase, most preferred first
      Online      map[string]bool // lowercase nicks
  }
  // RankBooks sets Score on every book and returns them sorted by Score desc,
  // ties kept in input order (sort.SliceStable). Never drops a book.
  func RankBooks(books []BookDetail, opts RankOptions) []BookDetail
  ```
  The search handler needs the query that produced the results. Add `lastQuery string` to `Client`, set it in `sendSearchRequest`, and read it in `searchResultHandler`.

**Scoring (the algorithm the tests pin; the spec's importance order made exact):**
```
normalize(s): lowercase, strip diacritics (golang.org/x/text/unicode/norm NFD + drop Mn),
              replace every non-alphanumeric rune with space, drop apostrophes first
              so "ender's" -> "enders", split on whitespace, drop tokens "the","a","an".
match   = round(10 * |queryTokens ∩ tokens(author+" "+title)| / |queryTokens|)   // 0..10
format  = len(order) - indexOf(format)  if listed, else 0                         // 0..len
quality = +2 if "retail" in Full (case-insens)
          -2 if any of "scan","ocr" as whole words in Full
          -1 if a version tag v1..v4 present (regex \bv[1-4](\.\d+)?\b)
online  = 1 if lowercase(Server) in Online else 0
Score   = match*1000 + format*100 + (quality+3)*10 + online
```
If the query is empty after normalizing, match = 0 for all books.

- [ ] **Step 1: Write failing tests** in `core/rank_test.go`. Build each test from small hand-made `BookDetail` values, plus one test against the Task 3 fixture:
  - `TestRank_MatchBeatsFormat`: query `"project hail mary"`. Title "Project Hail Mary" in pdf ranks above title "Mary Poppins" in epub.
  - `TestRank_FormatOrderWithinSameMatch`: same title in pdf, mobi, epub, and azw3 ranks as epub, azw3, mobi, pdf.
  - `TestRank_CustomFormatOrder`: with `FormatOrder: ["pdf","epub"]`, pdf ranks before epub.
  - `TestRank_RetailBoostScanPenalty`: same title and format; `(retail)` > plain > `[scan]` > `OCR v2`.
  - `TestRank_OnlineTiebreak`: two identical lines from different servers. The one whose server is in `Online` (under a different case) ranks first.
  - `TestRank_NormalizesPunctuationAndDiacritics`: query `"enders game"` gives title `"Ender's Game"` match = 10, and query `"jane eyre bronte"` against author `"Charlotte Brontë"` with title `"Jane Eyre"` gives match = 10. Assert via `Score / 1000 == 10`.
  - `TestRank_UnlistedFormatsLastNotDropped`: an input with `rar`, `""`, `txt`, and `epub` returns length 4 with epub first.
  - `TestRank_StableTies`: two books with identical scores keep input order.
  - `TestRank_BookzFixture`: ranking the fixture with query `"project hail mary"` puts an epub whose title contains "hail mary" at index 0.
- [ ] **Step 2:** `go test ./core -run TestRank`. Expected: FAIL (undefined).
- [ ] **Step 3:** Implement `RankBooks` per the scoring block. Add `golang.org/x/text` to `go.mod` and pin it with `go get golang.org/x/text@latest` (it's already an indirect dependency in many Go modules; check `go.sum`). Wire it into `searchResultHandler` with `server.repository.servers.OnlineSet()`.
- [ ] **Step 4:** `go test ./...`. Expected: PASS. Existing parser tests that asserted server-sorted order need updating to compare order-independently. Note this in the commit message.
- [ ] **Step 5: Commit** `feat: rank search results by match, format, quality, online`

### Task 5: Settings store + `GET/PUT /settings` + reconnect

**Files:**
- Create: `server/settings.go`, `server/settings_test.go`
- Modify: `server/server.go` (`Config` gains `Channel`, `ConfigDir`, `FormatOrder []string`, plus a `sync.RWMutex` guarding the mutable IRC fields; `Start` loads settings), `server/routes.go` (two routes), `server/client.go` / `server/websocket_requests.go` (reconnect), `cmd/openbooks/server.go` (`--config-dir` flag, env `CONFIG_DIR` fallback following the existing `BASE_PATH` pattern; force `Persist = true`)
- Test: `server/settings_test.go`, `server/settings_routes_test.go`

**Interfaces:**
- Consumes: `core.NormalizeChannel`, `core.DefaultFormatOrder`.
- Produces (API contract the frontend uses):
  ```go
  type Settings struct {
      Server      string   `json:"server"`      // host:port
      EnableTLS   bool     `json:"enableTls"`
      Channel     string   `json:"channel"`     // bare, no '#'
      SearchBot   string   `json:"searchBot"`   // bare, no '@'
      UserName    string   `json:"userName"`    // "" = random
      FormatOrder []string `json:"formatOrder"` // lowercase
      DownloadDir string   `json:"downloadDir"` // read-only; ignored on PUT
  }
  func LoadSettings(path string, defaults Settings) Settings // never errors; logs & falls back
  func SaveSettings(path string, s Settings) error           // atomic: write temp file, then rename
  func (s Settings) Validate() error
  ```
  `GET /settings` → 200 with `Settings` JSON. `PUT /settings` → validate, then 400 `{"error": "<message>"}` on failure. On success: save to `<ConfigDir>/settings.json`, apply to `server.config`, reconnect every connected client, return 200 with the saved `Settings`. If `ConfigDir` is `""`, keep the settings in memory only and log a warning once at startup.
  Validation rules: server must be `host:port` with a numeric port 1–65535 (`net.SplitHostPort`); channel must be non-empty after `NormalizeChannel`; searchBot must be non-empty after trimming `@`; formatOrder must be non-empty, with every entry lowercased and trimmed and no duplicates.

- [ ] **Step 1: Write failing tests:**
  - `TestLoadSettings_MissingFileUsesDefaults`
  - `TestLoadSettings_CorruptFileFallsBack`: a file containing `{not json` returns the defaults.
  - `TestLoadSettings_PartialFileFillsDefaults`: a file with only `{"channel":"ebooks"}` gives channel `ebooks` with every other field at its default.
  - `TestSaveThenLoad_RoundTrip`
  - `TestValidate`: table cases for `"irc.undernet.org"` (no port → error), `"host:99999"` (error), channel `"#"` (error), formatOrder `[]` (error), formatOrder `["EPUB"," pdf"]` (normalized to `["epub","pdf"]`), and a duplicate format (error).
  - `TestPutSettings_InvalidReturns400`
  - `TestPutSettings_PersistsAndReturnsSaved`: uses `httptest` and `t.TempDir()`. Checks that the file exists afterwards and that `downloadDir` in the request is ignored.
- [ ] **Step 2:** `go test ./server -run 'Settings|Validate'`. Expected: FAIL.
- [ ] **Step 3:** Implement. Reconnect for a client means:
  1. Cancel its reader context and call `irc.Disconnect()`.
  2. Create a fresh `irc.New(name, ...)`, using a random name when `UserName` is `""`, as `serveWs` does today.
  3. Call `startIrcConnection`, which already sends `ConnectionResponse` with the new username to the browser.

  This requires `Client.ctx` to become replaceable: store a `cancel func()` next to it. Read every config field used in `startIrcConnection`, `sendSearchRequest`, and `searchResultHandler` (for format order) under the RLock.
- [ ] **Step 4:** `go test ./... && go build ./...`. Expected: PASS.
- [ ] **Step 5: Commit** `feat: persisted settings with GET/PUT /settings and IRC reconnect`

### Task 6: Settings drawer (frontend)

**Files:**
- Create: `server/app/src/components/settings/SettingsDrawer.tsx`
- Modify: `server/app/src/state/api.ts` (add `getSettings` query and `updateSettings` mutation with tag `"settings"`), `server/app/src/components/sidebar/Sidebar.tsx` (gear button that opens the drawer), `server/app/src/state/stateSlice.ts` (expose `isSearching` if it doesn't exist; it is true from `sendSearch` until results or an error arrive)

**Interfaces:**
- Consumes: the `GET/PUT /settings` contract from Task 5. Use the same TS field names as the JSON tags.

Behaviour:
- Mantine `Drawer` with these fields:
  - server: `TextInput`
  - TLS: `Switch`
  - channel: `TextInput` showing a `#` prefix
  - search trigger: `TextInput` showing an `@` prefix
  - nickname: `TextInput`, placeholder "random"
  - format order: a list reordered with native HTML5 drag-and-drop (no new dependency), plus a text field to add a format and an × to remove one
  - download folder: read-only `TextInput`
- The Save button is disabled while `inFlightDownloads.length > 0` or `isSearching`, with the tooltip "Wait for the current search/download to finish".
- A 400 response shows the server's `error` text under the form. Success shows a Mantine notification "Settings saved — reconnecting".

- [ ] **Step 1:** Implement the components and wire them up.
- [ ] **Step 2:** `cd server/app && npm run build`. Expected: no TypeScript errors.
- [ ] **Step 3: Manual check** with `go run ./cmd/openbooks server --config-dir ./tmp-config` and `npm run dev`:
  1. Change the channel to `ebooks` and the server to `irc.irchighway.net:6697` with TLS on, then Save.
  2. The notification shows, the username in the sidebar changes, and `tmp-config/settings.json` reflects the change.
  3. Restart the server; the drawer still shows the saved values.
  4. Change the settings back.
- [ ] **Step 4: Commit** `feat: settings drawer`

### Task 7: Top picks (frontend)

**Files:**
- Create: `server/app/src/components/tables/TopPicks.tsx`
- Modify: `server/app/src/pages/SearchPage.tsx` (render `TopPicks` above `BookTable`), `server/app/src/state/messages.ts` (add `score: number` to the `BookDetail` type), `server/app/src/components/tables/BookTable.tsx` (default to the server order: no initial column sort, or an initial sort by `score` desc)

Behaviour:
- `TopPicks` takes `books: BookDetail[]` and renders the first 5 with no dedup. Each row shows title, author, a format badge, size, server, an online dot (same source as `BookTable`), and a Download button that dispatches the existing `sendDownload(book.full)`.
- It renders nothing when there are no books.
- The full `BookTable` stays below, unchanged except for its default order.

- [ ] **Step 1:** Implement.
- [ ] **Step 2:** `cd server/app && npm run build`. Expected: no TypeScript errors.
- [ ] **Step 3: Manual check** against live #bookz: search `project hail mary`. Top picks shows 5 rows with an epub first. Clicking Download shows queue progress, the file appears in the `--dir` folder, and the browser downloads it.
- [ ] **Step 4: Commit** `feat: top picks above search results`

### Task 8: Docker + README

**Files:**
- Modify: `Dockerfile`: add `VOLUME ["/config"]` and `ENV CONFIG_DIR=/config`. The entrypoint adds `--persist --config-dir /config`. Bump `FROM node:16` to `node:20` only if the build fails on 16.
- Create: `docker-compose.yml` (NAS example: ports `8080:80`, volumes `./books:/books` and `./config:/config`, `restart: unless-stopped`)
- Modify: `README.md`: add a "This fork" section at the top covering what it adds, the settings table (copied from the spec), the volumes, `docker compose up -d --build`, and how to point it back at IRCHighway.

- [ ] **Step 1:** Make the edits.
- [ ] **Step 2:** If Docker is available (it isn't on this Mac), run `docker build .`. Otherwise run `go build ./cmd/openbooks && (cd server/app && npm run build)` as the proxy, and leave the real image build to the Ship Checklist.
- [ ] **Step 3: Commit** `chore: docker config volume, compose example, fork README`

---

## Ship Checklist

- [ ] Migration: None
- [ ] Push: `git push -u origin feat/bookz-ranking-settings`, then merge to the fork's `master`. This needs Josh's go-ahead.
- [ ] Env vars: none required. `CONFIG_DIR=/config` is baked into the image.
- [ ] NAS: clone or pull the fork, create the `books/` and `config/` folders next to `docker-compose.yml`, run `docker compose up -d --build`, and map `books/` to the NAS share.
- [ ] Smoke:
  - Open `http://<nas>:8080`, search `project hail mary`. Top picks shows an epub first; click it. The file appears in the NAS `books/` share and also downloads in the browser.
  - Open settings, change the channel, and Save. The username changes, which confirms the reconnect.
  - Restart the container. The settings persist.
- [ ] Docs: the README "This fork" section (Task 8).
