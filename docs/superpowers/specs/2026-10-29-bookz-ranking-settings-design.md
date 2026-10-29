# Bookz fork: ranked results + settings panel

## Intent

Josh grabs ebooks from IRC by hand: connect to Undernet, join `#bookz`, type
`@search <title>`, unzip the result list a search bot DCC-sends back, scan
hundreds of near-duplicate lines, paste a `!Server ...` line, wait in a queue,
receive the file. The painful part is the scanning and choosing.

This fork of openbooks (`jwgerber/openbooks`, upstream `evan-buss/openbooks`)
makes that a web page on his NAS: type a title, see the best few matches ranked
for him, click one, and the book lands in a NAS folder.

## Goals

1. **Any network/channel.** Server, channel, search trigger, TLS, and nickname
   are settings. Defaults target Undernet `#bookz`.
2. **Ranked results.** Every result line gets a score; the UI shows the top 5
   prominently with the full list still available below.
3. **Settings panel in the web UI**, persisted across container restarts.
4. **Saves to the NAS.** Downloads always persist to the mounted folder.
5. **Reliable downloads.** Fold in upstream PR #206 (DCC stall/truncation/ACK
   hardening).

## Non-goals

- Auto-downloading without a click.
- Kindle / Calibre / Apple Books delivery or post-download hooks.
- Auth or user accounts (LAN-only tool).
- A server-side download history database (openbooks' existing browser-side
  history is fine to keep).
- Rewriting in Python or swapping Mantine for MUI.

## Anti-goals

- Hiding results: the full list must always be reachable, ranking only orders.
- Breaking the upstream IRCHighway use case — `#ebooks` must still work by
  changing settings, so upstream merges stay feasible.

## Constraints

- Keep the existing stack: Go backend, React + TypeScript + Mantine + Redux
  Toolkit frontend, existing Dockerfile.
- Runs as a single Docker container on a NAS. No inbound ports needed beyond
  the web UI (standard DCC SEND: the app connects out to the sender).
- Each browser client currently gets its own IRC connection
  (`server/client.go`); keep that model.

## Decided

### Settings

| Setting | Default | Notes |
|---|---|---|
| Server | `irc.undernet.org:6667` | host:port |
| TLS | off | Undernet is mostly plaintext |
| Channel | `#bookz` | currently hard-coded `ebooks` in `core/irchighway.go` |
| Search trigger | `@search` | currently `--searchbot` flag |
| Nickname | random, as today | |
| Format order | epub, azw3, mobi, pdf | drag to reorder in UI; unlisted formats rank after |
| Download folder | `/books` (mounted volume) | shown read-only-ish in UI; it's a container path |

- Container flags/env set **starting defaults only**.
- UI changes save to a JSON settings file in a mounted config folder
  (e.g. `/config/settings.json`). File wins over flags once it exists.
- Saving settings reconnects the current client's IRC session with the new
  values.
- Persist mode is always on in this fork (books stay in the folder).

### Ranking (server-side, in Go)

Each parsed `BookDetail` gets a score; the order of importance is:

1. **Match** — how closely author + title match the query (normalized:
   case, punctuation, diacritics, word order tolerant).
2. **Format** — position in the user's format order.
3. **Quality** — boost `retail`; penalize `scan`, `OCR`, and old version tags
   like `v1`–`v4` vs higher.
4. **Server online** — boost if the server nick is currently in the channel
   (openbooks already tracks the server list in `core/server_parser.go`).

Results are sent to the browser already sorted, with the score available.

### UI

- Search box as today (title, optionally author in the same box).
- "Top picks": the 5 highest-scoring lines, no deduplication (copies of the
  same book from different servers may appear) — title, author, format, size, server, online dot,
  download button.
- Full results table below (existing table), sorted by score.
- Settings panel (drawer or modal) with the fields above.
- Existing queue/download progress behaviour kept.

## Builder's calls (made during review — Josh to check)

- **#bookz support is a goal, not best-effort.** First task: connect to
  Undernet `#bookz`, capture a real search result file, bot notices
  (accepted / queued / sorry / no results), and NAMES output, and check them
  as test fixtures. Any wording or format that differs from IRCHighway
  becomes a setting or parser fallback, not a hard-coded fork.
- **Match is a token-bag score:** normalized query tokens vs tokens of
  `author + " " + title`. No author/title split in the search box.
- **"Online" = any nick currently in the channel**, not only `@`/`+`. The
  server list is refreshed (NAMES) on each search and reset on reconnect or
  channel change.
- **Single user is the real case.** Saving settings reconnects every open
  client. Save is disabled while a search or download is in flight.

## Left to the builder

- Exact scoring weights and fuzzy-match algorithm; covered by table-driven
  tests on real-looking result lines.
- How settings travel to the browser (REST endpoint vs existing websocket).
- How to merge PR #206 (cherry-pick vs re-apply) and resolving conflicts.
- Whether the channel name is stored with or without `#`.
- README section documenting the fork's settings and Docker volumes.
