# Record the real-profile Kitty demo

The README demo shows a saved public snapshot of
[@circletoonsig](https://www.instagram.com/circletoonsig/) and the selected
[Instagram post DdwMTTuMNnE](https://www.instagram.com/p/DdwMTTuMNnE/).
The avatar and post thumbnail are real saved images. Kitty displays them with
its graphics protocol; a text-only terminal recording cannot show those pixels.

The demo contains **one creator and one selected post**, not a complete feed.
Its counts are static observations from **30 September 2026 at 11:54:31 UTC**.
See [source and attribution](ATTRIBUTION.md#source-and-observation-time) for the
exact timestamp, acquisition method and bundled JSON.
Missing metrics remain unknown, and opening the demo does not collect new data.
No Instagram login, capture, backend, or API key is needed to view it. Live
collection separately requires a working mobile API session as explained in
[the README](../README.md#2-prepare-your-own-session-capture).

## Open the saved snapshot

Install [Kitty](https://sw.kovidgoyal.net/kitty/binary/). In a Kitty window, from
the repository root:

```sh
go build -o bin/instagram-scraper ./cmd/instagram-scraper
./bin/instagram-scraper --demo
```

To inspect the saved observation timestamp, counts and source post URL:

```sh
./bin/instagram-scraper --demo --json
```

Use a window around 1440 × 900 pixels with a readable monospace font. If your
shell normally sets `NO_COLOR`, temporarily unset it for a color recording.
The app can still be used without color.

## Suggested sequence

1. Pause on the saved `@circletoonsig` profile so its identity and observation
   time are readable.
2. Press `i` to show the real avatar. Press **Enter or Esc** to return.
3. Press **Enter** to open the profile and inspect the selected post's saved
   counters. Keep unavailable counts visible as unknown.
4. Press `t` to show the real thumbnail for `DdwMTTuMNnE`. Press **Enter or Esc**
   to return to the post details.
5. Press **Esc** to return to the profile list. Pause briefly, then stop the
   recording before closing the viewer with `q`.

There is no second creator or post in this bundled walkthrough. Image previews
close with **Enter or Esc**, not arbitrary keys. When automating Kitty, use its
`send-key` command for these keys so the enhanced keyboard protocol is encoded
correctly. Avoid sending a raw carriage return as text.

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
Record the actual application window so the avatar and thumbnail are captured
as Kitty rendered them.

The README assets are:

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
assets. Keep the saved-snapshot label and observation time readable. Do not
include session headers, cookies, capture files or private profiles in a public
recording. Credit the source in [the attribution note](ATTRIBUTION.md).

Synthetic fixtures remain useful for automated tests; the public showcase uses
the saved real profile and selected post described above.
