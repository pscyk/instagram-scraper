# Instagram Scraper

Instagram profiles and reel metrics in your terminal.

[![Instagram Scraper](docs/demo.gif)](docs/demo.mp4?raw=1)

[Watch video](docs/demo.mp4?raw=1)

## Install

Requires Git, Go 1.26+, Python 3.9+, and curl. Use [Kitty](https://sw.kovidgoyal.net/kitty/binary/)
for profile pictures and reel thumbnails.

On macOS with [Homebrew](https://brew.sh/):

```sh
brew install git go python curl
```

On Debian or Ubuntu:

```sh
sudo apt update
sudo apt install git python3 curl
```

On Linux, install Go using the [Go installation guide](https://go.dev/doc/install).
Then clone and build:

```sh
git clone https://github.com/pscyk/instagram-scraper.git
cd instagram-scraper
go build -o bin/instagram-scraper ./cmd/instagram-scraper
./bin/instagram-scraper --demo
```

Press **Enter** to open a profile, **i** for its picture, **t** for a reel
thumbnail, and **q** to quit.

## Credentials

Use a working cURL capture from your authenticated Instagram mobile API session.
It must include `-H 'Authorization: Bearer …'`. Keep the request's user agent,
device headers, and any Cookie header together. See the
[capture format](examples/zebra_curls.redacted.sh).

Store your capture at `.secrets/capture.sh`, replacing the source path below
with your own file:

```sh
mkdir -p .secrets
chmod 700 .secrets
install -m 600 /path/to/captured-request.sh .secrets/capture.sh
```

The collector reads the file as text. Keep it private; it contains your session
credentials. `.secrets/` and `results/` are excluded from Git.

```text
instagram-scraper/
├── .secrets/
│   └── capture.sh
└── results/
    ├── creator.json
    └── creator.media/
```

## Collect a profile

Pass an Instagram handle and choose where to save its data:

```sh
python3 lookup.py circletoonsig \
  --capture .secrets/capture.sh \
  --max-pages 3 \
  --download-images \
  --output results/circletoonsig.json
```

This saves profile details, reel counters, an avatar, and up to 12 thumbnails.
`--thumbnail-limit` accepts a value from 0 to 100. Omit `--download-images`
to collect metrics only.

Requests are paced at least one second apart. Keep one collector running per
session; use `--delay 2` for a longer interval. Start with a small `--max-pages`
value: each feed page can lead to several reel-detail requests. The default is
100 pages.

When a session expires, check the account in Instagram and replace
`.secrets/capture.sh` with a fresh capture. Keep the file permissions at `600`.

## Browse profiles

```sh
./bin/instagram-scraper results/
./bin/instagram-scraper results/circletoonsig.json
```

To export the loaded data as JSON:

```sh
./bin/instagram-scraper --input results --json
```

| Key | Action |
| --- | --- |
| `↑` / `↓` or `k` / `j` | Select a profile or scroll its details |
| `Enter` / `Esc` | Open a profile / go back |
| `/` | Search |
| `f` / `s` / `c` | Change filter / sort / clear |
| `[` / `]` | Previous / next profile |
| `n` / `p` | Next / previous reel |
| `i` / `t` | Profile picture / reel thumbnail |
| `?` / `q` | Help / quit |

Image previews use Kitty's `kitten` helper. Press **Enter or Esc** to close a
picture. On macOS, install Kitty in `/Applications`; on Linux, check that
`kitten --version` works. Copy the `.media/` folder alongside a JSON file when
moving profiles between machines.

## Metrics

Profiles include followers, following, post count, biography, and verification
status. Reels include captions, publication dates, plays, likes, comments,
reshares, and reposts when Instagram provides them.

`—` means unavailable; zero is displayed as `0`. The capture date and feed
coverage appear with each profile. See [field definitions](METRICS.md).

## Troubleshooting

| Problem | Fix |
| --- | --- |
| Missing command | Install the required tool and reopen your terminal. Check `go version`, `python3 --version`, and `curl --version`. |
| Capture file not found | Run from the repository directory or pass an absolute path with `--capture`. |
| Missing bearer authorization | Use a mobile API capture containing `-H 'Authorization: Bearer …'`. |
| HTTP 401/403 | Check the account in Instagram and replace the session capture. |
| HTTP 429 | Pause collection, keep one collector per session, and increase `--delay`. |
| Image unavailable | Collect with `--download-images` and keep the `.media/` folder beside its JSON file. |
| Kitty helper unavailable | Install Kitty and make its `kitten` command available. |
| Interactive mode requires a terminal | Open the viewer in a terminal, or use `--json` for scripts. |
| No JSON files found | Point the viewer at a profile JSON file or its containing directory. |

## Development

```sh
python3 -m unittest discover -s tests -v
go test -race ./...
go vet ./...
```

Built with [Charm](https://charm.land/). [Recording guide](docs/DEMO.md).

[MIT license](LICENSE) · [Media credits](docs/ATTRIBUTION.md)
