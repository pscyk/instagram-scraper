# Record the terminal walkthrough

Open a Kitty window around 1440 × 900 pixels. From the repository root:

```sh
go build -o bin/instagram-scraper ./cmd/instagram-scraper
./bin/instagram-scraper --demo
```

Use this sequence:

1. Pause on the profile list.
2. Press `i` for the profile picture, then **Enter** to return.
3. Press **Enter** to open the profile and its reel counters.
4. Press `t` for the thumbnail, then **Enter** to return.
5. Pause on the counters and stop recording before pressing `q`.

Use **Enter or Esc** to close image previews. For automation, Kitty's
`send-key` command handles the terminal's keyboard protocol.

## Capture

On macOS, press Shift–Command–5 and select the Kitty window region. On Wayland,
use your desktop recorder or `wf-recorder`:

```sh
wf-recorder -g "$(slurp)" -f recording.mp4
```

Select the Kitty rectangle and stop with Ctrl+C in the recorder terminal.
Capture only the application window and leave audio disabled.

## README assets

```sh
ffmpeg -i recording.mp4 -an -c:v libx264 -crf 20 -pix_fmt yuv420p \
  -movflags +faststart docs/demo.mp4
ffmpeg -i docs/demo.mp4 \
  -filter_complex '[0:v]fps=10,scale=960:-1:flags=lanczos,split[a][b];[a]palettegen[p];[b][p]paletteuse' \
  -loop 0 docs/demo.gif
ffmpeg -ss 10 -i docs/demo.mp4 -frames:v 1 docs/demo.png
```

Check the profile, picture, thumbnail, and counters in the finished video.
[Media credits and source data](ATTRIBUTION.md).
