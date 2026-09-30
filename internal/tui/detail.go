package tui

import (
	"fmt"
	"strings"

	"github.com/pscyk/instagram-scraper/internal/snapshot"
)

func (m *Model) detailContent(item *snapshot.Snapshot, width int) string {
	profile := item.Profile
	name := profile.FullName
	if name == "" {
		name = profile.Username
	}
	heading := strongStyle.Render(clipped(name, width)) + "\n" + accentStyle.Render("@"+profile.Username)
	verified := ""
	if profile.IsVerified != nil && *profile.IsVerified {
		verified = "  ·  verified"
	}
	lines := []string{heading, dimStyle.Render(privacy(profile.IsPrivate) + category(profile.Category) + verified), "",
		metricRow([]string{"FOLLOWERS", "FOLLOWING", "POSTS"}, []*int64{profile.Followers, profile.Following, profile.MediaCount}, width),
		"", wrap(profile.Biography, width, 5), "", dimStyle.Render(measured(item)), dimStyle.Render(coverage(item))}
	if item.Source != "" {
		lines = append(lines, dimStyle.Render("Source: "+clipped(item.Source, max(1, width-8))))
	}
	if profile.AvatarPath != "" {
		lines = append(lines, accentStyle.Render("i  profile picture · cached locally"))
	}
	lines = append(lines, "", accentStyle.Bold(true).Render("REELS / RAW COUNTERS"), dimStyle.Render("— unavailable  ·  counts are preserved exactly  ·  no inferred metrics"), "")
	if len(item.Reels) == 0 {
		lines = append(lines, dimStyle.Render("No reel observations saved in this snapshot."))
	}
	m.reelOffsets = make([]int, len(item.Reels))
	row := strings.Count(strings.Join(lines, "\n"), "\n") + 1
	for index := range item.Reels {
		m.reelOffsets[index] = row
		card := reelCard(&item.Reels[index], index, width, index == m.reel)
		row += strings.Count(card, "\n") + 1
		lines = append(lines, card)
	}
	return strings.Join(lines, "\n")
}

func reelCard(reel *snapshot.Reel, index, width int, selected bool) string {
	title := fmt.Sprintf("%02d  %s", index+1, date(reel.PublishedAt))
	style := strongStyle
	if selected {
		title = "▸ " + title
		style = accentStyle.Bold(true)
	}
	if reel.Duration != nil {
		title += fmt.Sprintf("  ·  %.1fs", *reel.Duration)
	}
	if reel.Code != "" {
		title += "  ·  " + SafeText(reel.Code, 50)
	}
	lines := []string{style.Render(clipped(title, width)), wrap(reel.Caption, width, 2), ""}
	if selected && reel.ThumbnailPath != "" {
		lines = append(lines, accentStyle.Render("t  reel thumbnail · n/p select another reel"), "")
	}
	if width >= 80 {
		lines = append(lines, metricRow([]string{"PLAYS", "LIKES", "COMMENTS", "RESHARES"},
			[]*int64{reel.Plays, reel.Likes, reel.Comments, reel.Reshares}, width))
		lines = append(lines, metricRow([]string{"IG PLAYS", "FB PLAYS", "VIEWS", "REPOSTS"},
			[]*int64{reel.InstagramPlays, reel.FacebookPlays, reel.Views, reel.Reposts}, width))
	} else {
		lines = append(lines, metricRow([]string{"PLAYS", "LIKES", "COMMENTS"}, []*int64{reel.Plays, reel.Likes, reel.Comments}, width),
			metricRow([]string{"RESHARES", "VIEWS", "REPOSTS"}, []*int64{reel.Reshares, reel.Views, reel.Reposts}, width),
			metricRow([]string{"IG PLAYS", "FB PLAYS", "PREMIUM"}, []*int64{reel.InstagramPlays, reel.FacebookPlays, reel.PremiumReactions}, width))
	}
	if width >= 80 && reel.PremiumReactions != nil {
		lines = append(lines, dimStyle.Render("Premium reactions: ")+textStyle.Render(number(reel.PremiumReactions)))
	}
	lines = append(lines, "", dimStyle.Render(strings.Repeat("─", width)), "")
	return strings.Join(lines, "\n")
}

func metricRow(labels []string, values []*int64, width int) string {
	cellWidth := max(1, width/len(labels))
	for _, value := range values {
		if len(number(value)) >= cellWidth {
			rows := make([]string, 0, len(labels))
			for index, label := range labels {
				rows = append(rows, dimStyle.Render(label+": ")+textStyle.Render(number(values[index])))
			}
			return strings.Join(rows, "\n")
		}
	}
	head, counts := "", ""
	for index, label := range labels {
		head += dimStyle.Render(pad(label, cellWidth))
		style := textStyle
		if label == "PLAYS" {
			style = goodStyle.Bold(true)
		}
		counts += style.Render(pad(number(values[index]), cellWidth))
	}
	return head + "\n" + counts
}

func pad(value string, width int) string {
	return fitFrame(value, width, 1)
}
