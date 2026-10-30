# openbooks

## This fork

A fork of [evan-buss/openbooks](https://github.com/evan-buss/openbooks) for IRC book channels beyond IRCHighway. It defaults to Undernet `#bookz` and adds:

- **Ranked results.** Results are sorted by how well the author and title match your search, then your format order, then how closely the title matches (fewer extra words), then release quality (retail is boosted; scans, OCR, and v1–v4 are pushed down), then whether the server is in the channel. The five best appear as **Top picks** above the full list, and nothing is hidden.
- **A settings panel** (the gear icon in the sidebar) for server, channel, search trigger, TLS, nickname, and format order. Saving reconnects to IRC. Settings are stored in `/config/settings.json` and survive restarts.
- **Undernet fixes.** It joins the channel after the server's welcome, times out dead servers, and reads multi-line user lists.

| Setting | Default |
|---|---|
| Server | `irc.undernet.org:6667` |
| TLS | off |
| Channel | `#bookz` |
| Search trigger | `@search` |
| Nickname | random (`readerNNNN`) |
| Format order | epub, azw3, mobi, pdf |
| Download folder | `/books` (container path) |

Container flags (`--server`, `--channel`, `--searchbot`, `--tls`, `--name`) set only the starting defaults. Once `settings.json` exists, it wins.

### Run on a NAS

```bash
git clone https://github.com/jwgerber/openbooks.git && cd openbooks
mkdir -p books config
docker compose up -d --build
```

Then open `http://<nas>:8080`. Point `./books` at your NAS share in `docker-compose.yml`. Books are saved there and also download to your browser.

To use IRCHighway instead, set the server to `irc.irchighway.net:6697`, turn TLS on, and set the channel to `#ebooks` in settings.

---

> NOTE: Going forward only the latest release will be supported. If you encounter any issues, be sure you are using the latest version.

[![Docker Pulls](https://img.shields.io/docker/pulls/evanbuss/openbooks.svg)](https://hub.docker.com/r/evanbuss/openbooks/)

Openbooks allows you to download ebooks from irc.irchighway.net quickly and easily.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./.github/home_v3_dark.png">
  <img alt="openbooks screenshot" src="./.github/home_v3.png">
</picture>


## Getting Started

### Binary

1. Download the latest release for your platform from the [releases page](https://github.com/evan-buss/openbooks/releases).
2. Run the binary
   - Linux users may have to run `chmod +x [binary name]` to make it executable
3. `./openbooks --help`
   - This will display all possible configuration values and introduce the two modes; CLI or Server.

### Docker

- Basic config
  - `docker run -p 8080:80 evanbuss/openbooks`
- Config to persist all eBook files to disk
  - `docker run -p 8080:80 -v /home/evan/Downloads/openbooks:/books evanbuss/openbooks --persist`

### Setting the Base Path

OpenBooks server doesn't have to be hosted at the root of your webserver. The basepath value allows you to host it behind a reverse proxy. The base path value must have opening and closing forward slashes (default "/").

- Docker
  - `docker run -p 8080:80 -e BASE_PATH=/openbooks/ evanbuss/openbooks`
- Binary
  - `./openbooks server --basepath /openbooks/`

## Usage

For a complete list of features use the `--help` flags on all subcommands.
For example `openbooks cli --help or openbooks cli download --help`. There are
two modes; Server or CLI. In CLI mode you interact and download books through
a terminal interface. In server mode the application runs as a web application
that you can visit in your browser.

Double clicking the executable will open the UI in your browser. In the future it may use [webviews](https://developer.microsoft.com/en-us/microsoft-edge/webview2/) to provide a "native-like" desktop application. 

## Development

### Install the dependencies

- `go get`
- `cd server/app && npm install`
- `cd ../..`
- `go run main.go`

### Build the React SPA and compile binaries for multiple platforms.

- Run `./build.sh`
- This will install npm packages, build the React app, and compile the executable.

### Build the go binary (if you haven't changed the frontend)

- `go build`

### Mock Development Server

- The mock server allows you to debug responses and requests to simplified IRC / DCC
  servers that mimic the responses received from IRC Highway.
- ```bash
  cd cmd/mock_server
  go run .
  # Another Terminal
  cd cmd/openbooks
  go run . server --server localhost --log
  ```

### Desktop App
Compile OpenBooks with experimental webview support:

``` shell
cd cmd/openbooks
go build -tags webview
```


## Why / How

- I wrote this as an easier way to search and download books from irchighway.net. It handles all the extraction and data processing for you. You just have to click the book you want. Hopefully you find it much easier than the IRC interface.
- It was also interesting to learn how the [IRC](https://en.wikipedia.org/wiki/Internet_Relay_Chat) and [DCC](https://en.wikipedia.org/wiki/Direct_Client-to-Client) protocols work and write custom implementations.

## Technology

- Backend
  - Golang
  - Chi
  - gorilla/websocket
  - Archiver (extract files from various archive formats)
- Frontend
  - React.js
  - TypeScript
  - Redux / Redux Toolkit
  - Mantine UI / @emotion/react
  - Framer Motion
