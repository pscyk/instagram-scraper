# Demo media attribution

The bundled public demo shows [@circletoonsig](https://www.instagram.com/circletoonsig/)
and the selected [Instagram post DdwMTTuMNnE](https://www.instagram.com/p/DdwMTTuMNnE/).
The saved avatar, post thumbnail and displayed post content are attributed to
their source creator and respective owners.

The repository's [MIT license](../LICENSE) covers its source code. Instagram
images and post content remain their owners' material; including them in the
demo does not relicense them under MIT or imply endorsement of this project.

## Source and observation time

The [bundled snapshot](../internal/snapshot/demo/circletoonsig.json) was observed
on **30 September 2026 at 11:54:31 UTC** (`2026-09-30T11:54:31.611Z`). Its source
is embedded JSON from the public Instagram profile and post pages, accessed
without a login. This acquisition used the public pages, not an authenticated
mobile API session.

It contains one selected reel, with `feed_complete: false`; it does not represent
a complete profile feed or a ranking of the creator's posts. Reported followers,
following, likes and comments are saved observations, not current totals. Plays,
video duration, profile post count and other unavailable metrics remain `null`.

The avatar and post thumbnail were saved as local PNGs after decoding and
re-encoding their pixels to remove metadata. The published snapshot omits CDN
image URLs and their query parameters. The viewer reads these local files
offline and does not refresh their data or images. Inspect the saved values with:

```sh
./bin/instagram-scraper --demo --json
```
