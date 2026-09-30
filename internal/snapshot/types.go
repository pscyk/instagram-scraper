// Package snapshot reads bounded, local lookup.py results. It never contacts Instagram.
package snapshot

import "time"

// Profile contains only fields reported by the profile endpoint.
type Profile struct {
	PK            any    `json:"pk,omitempty"`
	Username      string `json:"username"`
	FullName      string `json:"full_name"`
	Biography     string `json:"biography"`
	ExternalURL   string `json:"external_url"`
	Followers     *int64 `json:"follower_count"`
	Following     *int64 `json:"following_count"`
	MediaCount    *int64 `json:"media_count"`
	IsPrivate     *bool  `json:"is_private"`
	IsVerified    *bool  `json:"is_verified"`
	Category      string `json:"category"`
	AvatarPath    string `json:"avatar_path,omitempty"`
	ProfilePicURL string `json:"profile_pic_url,omitempty"`
	AvatarURL     string `json:"avatar_url,omitempty"`
}

// Reel preserves absent metrics as nil; views and plays remain distinct.
type Reel struct {
	PK               any      `json:"pk,omitempty"`
	Code             string   `json:"code"`
	TakenAt          *int64   `json:"taken_at"`
	Plays            *int64   `json:"play_count"`
	InstagramPlays   *int64   `json:"ig_play_count"`
	FacebookPlays    *int64   `json:"fb_play_count"`
	Views            *int64   `json:"view_count"`
	Likes            *int64   `json:"like_count"`
	Comments         *int64   `json:"comment_count"`
	Reshares         *int64   `json:"reshare_count"`
	Reposts          *int64   `json:"media_repost_count"`
	PremiumReactions *int64   `json:"premium_reaction_count"`
	Duration         *float64 `json:"video_duration"`
	Caption          string   `json:"caption"`
	URL              string   `json:"url"`
	PublishedAt      string   `json:"published_at_utc"`
	ThumbnailPath    string   `json:"thumbnail_path,omitempty"`
	ThumbnailURL     string   `json:"thumbnail_url,omitempty"`
}

// Snapshot follows lookup.py's saved JSON schema, with no predictions or scores.
type Snapshot struct {
	MeasuredAt    string  `json:"measured_at_utc"`
	Profile       Profile `json:"profile"`
	FeedComplete  *bool   `json:"feed_complete"`
	PostsScanned  *int64  `json:"posts_scanned"`
	ReelsChecked  *int64  `json:"reels_checked"`
	RankingMetric string  `json:"ranking_metric"`
	MissingPlays  *int64  `json:"reels_missing_play_count"`
	Reels         []Reel  `json:"ranked_reels"`
	Source        string  `json:"-"`
	baseDir       string
	demo          bool
}

// Measurement returns a validated UTC instant, or zero when it was not recorded.
func (s *Snapshot) Measurement() time.Time {
	value, _ := time.Parse(time.RFC3339Nano, s.MeasuredAt)
	return value.UTC()
}
