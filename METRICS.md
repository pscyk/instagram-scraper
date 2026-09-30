# Metrics observed on 11 September 2026 IST

These are observed response fields, not a guaranteed Instagram API contract.

| Request | Confirmed counters and metadata |
| --- | --- |
| `username_info` | `follower_count`, `following_count`, `media_count`; username, bio, website, public/private and verified flags |
| `user_feed` | `like_count`, `comment_count`, `reshare_count`, `media_repost_count`, `premium_reaction_count`, `video_duration`; IDs, timestamps and captions |
| `media_info` | The feed metrics plus `play_count` and `ig_play_count` |

`play_count` is plays, not unique viewers. Do not add it to `ig_play_count`:
both had the same value in the observed responses. Reshares and reposts are
distinct returned fields; their populations and overlap are not established here.
The meaning of `premium_reaction_count` is unverified.

The observed feed responses omitted plays. Ranking therefore fetches details
for every reel, rather than assuming the most-liked reel has the most plays.
Unavailable metrics remain JSON `null`; they are not converted to zero.
`fb_play_count` and `view_count` were absent in the observed detail responses.

The profile returned `total_clips_count: 1` while the complete profile feed
contained 25 reels. Do not treat that profile field as a reliable reel total.

Saves, unique reach, impressions, watch time, retention, completion rate and
attributed follows were not returned. The supplied curl inventory has no
dedicated Insights request. Other supplied requests (comments/replies, web
profile, profile stream, GraphQL, music and audio pivots) have not had their
response fields verified by this tool.

Coverage means the accessible profile feed reached `more_available: false`.
It does not establish coverage of deleted, archived or otherwise inaccessible
media, or reels not included in that feed. Snapshot counts change over time.
