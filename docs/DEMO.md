# Record the Kitty graphics demo

The README video records the real application in a native Kitty window on
Wayland. Profile pictures and reel thumbnails use Kitty's graphics protocol.
A plain terminal emulator or text-only recorder cannot reproduce those pixels.

Every profile, counter, caption and image in `--demo` is synthetic. The original
SVG illustrations are in `examples/demo-art/`; their PNG versions are embedded
under `internal/snapshot/demo/assets/`. No capture, Instagram session, backend,
or real saved profile is needed to reproduce the demo.

## Open the demo

Install [Kitty](https://sw.kovidgoyal.net/kitty/binary/). In a Kitty window, from
the repository root:

```sh
go build -o bin/instagram-scraper ./cmd/instagram-scraper
./bin/instagram-scraper --demo
```

Use a window around 1440 × 900 pixels with a readable monospace font. If your
shell normally sets `NO_COLOR`, temporarily unset it for a color recording.
The app can still be used without color.

## Suggested sequence

1. Pause on the saved-profile catalogue.
2. Press `i` to show a profile picture; press **Enter** to return.
3. Press `/`, type `lumen`, then press Enter to search. Press Enter again to
   open that profile.
4. Press `t` to show its first saved reel thumbnail; press Enter to return.
5. Press `n`, then `t` to show the next reel thumbnail; press Enter to return.
6. Press Esc to return to the catalogue. Pause briefly, then press `c` to clear
   search and `f` twice to show an unmeasured profile.
7. Stop the recording before closing the demo with `q`.

Image previews close with **Enter or Esc**, not arbitrary keys. When automating
Kitty, use its `send-key` command for these keys so the enhanced keyboard
protocol is encoded correctly. Avoid sending a raw carriage return as text.

## Capture the actual window

On macOS, use the system screen recorder (Shift–Command–5) and select only the
Kitty region. On Wayland, use your desktop recorder or `wf-recorder` with a
selected Kitty-only region, for example:

```sh
wf-recorder -g "$(slurp)" -f demo-native.mp4
```

Select the Kitty window's rectangle when prompted and stop with Ctrl+C in the
recorder terminal. Use a dedicated workspace so notifications and unrelated
windows cannot enter the recording. Do not enable microphone or system audio.

The checked-in recording uses native Wayland capture, not a reconstructed
terminal animation. The README assets are:

- `docs/demo.mp4`: native video.
- `docs/demo.gif`: animated README preview.
- `docs/demo.png`: still preview.

For a new recording, encode an H.264 MP4 and a small GIF preview with FFmpeg:

```sh
ffmpeg -i demo-native.mp4 -an -c:v libx264 -crf 20 -pix_fmt yuv420p \
  -movflags +faststart docs/demo.mp4
ffmpeg -i docs/demo.mp4 \
  -filter_complex '[0:v]fps=10,scale=960:-1:flags=lanczos,split[a][b];[a]palettegen[p];[b][p]paletteuse' \
  -loop 0 docs/demo.gif
ffmpeg -ss 1 -i docs/demo.mp4 -frames:v 1 docs/demo.png
```

Inspect the profile, avatar and thumbnail frames before replacing the README
assets. Preserve the visible **SYNTHETIC DEMO** label, and never substitute real
session material or private results in a public recording.
