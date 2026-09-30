package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/pscyk/instagram-scraper/internal/snapshot"
)

var (
	ink         = lipgloss.Color("#E7E9F1")
	muted       = lipgloss.Color("#9A9FB2")
	violet      = lipgloss.Color("#BEA1FF")
	mint        = lipgloss.Color("#8CE6C0")
	line        = lipgloss.Color("#343749")
	textStyle   = lipgloss.NewStyle().Foreground(ink)
	dimStyle    = lipgloss.NewStyle().Foreground(muted)
	accentStyle = lipgloss.NewStyle().Foreground(violet)
	goodStyle   = lipgloss.NewStyle().Foreground(mint)
	strongStyle = textStyle.Bold(true)
)

func (m *Model) View() tea.View {
	view := tea.NewView(m.render())
	view.AltScreen = true
	return view
}

func (m *Model) render() string {
	if m.width < 54 || m.height < 16 {
		message := "Instagram Scraper\nResize to 54 × 16 or larger.\nq quit"
		if m.demo {
			message += "\nSYNTHETIC DEMO · OFFLINE"
		}
		return fitFrame(message, m.width, m.height)
	}
	width := m.width - 4
	content := m.catalog(width)
	if m.detail {
		content = m.viewport.View()
	}
	if m.help {
		content = helpView(width)
	}
	status := m.status(width)
	frame := m.header(width) + "\n" + status + "\n" + dimStyle.Render(strings.Repeat("─", width)) + "\n" +
		fitFrame(content, width, m.height-9) + "\n" + dimStyle.Render(strings.Repeat("─", width)) + "\n" + m.footer(width)
	return fitFrame(lipgloss.NewStyle().Padding(1, 2).Render(frame), m.width, m.height)
}

func (m *Model) header(width int) string {
	title := strongStyle.Render("◈  INSTAGRAM ") + accentStyle.Bold(true).Render("SCRAPER")
	mode := goodStyle.Render("LOCAL SNAPSHOTS")
	if m.demo {
		mode = accentStyle.Render("SYNTHETIC DEMO · OFFLINE")
	}
	gap := max(1, width-ansi.StringWidth(title)-ansi.StringWidth(mode))
	return title + strings.Repeat(" ", gap) + mode
}

func (m *Model) status(width int) string {
	if m.input.Focused() {
		return m.input.View()
	}
	if m.previewNote != "" {
		return accentStyle.Render(clipped(m.previewNote, width))
	}
	filter := []string{"all profiles", "measured", "unmeasured", "public"}[m.filter]
	order := []string{"handle", "followers", "latest snapshot"}[m.sort]
	label := fmt.Sprintf("%02d snapshots  /  %s  /  sort: %s", len(m.visible), filter, order)
	if m.query != "" {
		label += "  /  \"" + m.query + "\""
	}
	if m.detail {
		label = "PROFILE / @" + m.current().Profile.Username + "  ·  saved raw metrics"
		if len(m.current().Reels) > 0 {
			label += fmt.Sprintf("  ·  reel %d/%d", m.reel+1, len(m.current().Reels))
		}
	}
	if m.help {
		label = "KEYBOARD / a local, read-only workspace"
	}
	return dimStyle.Render(clipped(label, width))
}

func (m *Model) footer(width int) string {
	keys := "/ search   ↑↓ select   enter profile   i picture   f filter   s sort   ? help   q quit"
	if m.detail {
		keys = "↑↓ scroll   n/p reel   t thumbnail   i picture   [ ] profile   esc back   ? help   q quit"
	}
	if m.input.Focused() {
		keys = "enter apply search   esc cancel   ctrl+c quit"
	}
	return dimStyle.Render(clipped(keys, width))
}

func (m *Model) catalog(width int) string {
	if len(m.visible) == 0 {
		return strongStyle.Render("No matching snapshots") + "\n\n" + dimStyle.Render("Press c to clear search and filters.")
	}
	if width < 86 {
		return m.profileList(width, m.height-9)
	}
	leftWidth := max(30, min(40, width/3))
	rightWidth := width - leftWidth - 5
	left := m.profileList(leftWidth, m.height-9)
	right := m.summary(m.current(), rightWidth, m.height-9)
	divider := dimStyle.Render(strings.Repeat("│\n", max(0, m.height-10)) + "│")
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", divider, "  ", right)
}

func (m *Model) profileList(width, height int) string {
	rows := []string{dimStyle.Render("SAVED PROFILES"), ""}
	pageSize := max(1, (height-3)/3)
	start := (m.selected / pageSize) * pageSize
	end := min(len(m.visible), start+pageSize)
	for position := start; position < end; position++ {
		item := &m.items[m.visible[position]]
		prefix := "  "
		style := textStyle
		if position == m.selected {
			prefix = "▸ "
			style = accentStyle.Bold(true)
		}
		rows = append(rows, style.Render(clipped(prefix+"@"+item.Profile.Username, width)))
		label := "  " + number(item.Profile.Followers) + " followers"
		if item.Profile.Followers == nil {
			label = "  followers unavailable"
		}
		rows = append(rows, dimStyle.Render(clipped(label, width)), "")
	}
	if end < len(m.visible) || start > 0 {
		rows = append(rows, dimStyle.Render(fmt.Sprintf("%d–%d of %d  ·  ↑↓ to explore", start+1, end, len(m.visible))))
	}
	return fitFrame(strings.Join(rows, "\n"), width, height)
}

func (m *Model) summary(item *snapshot.Snapshot, width, height int) string {
	profile := item.Profile
	name := profile.FullName
	if name == "" {
		name = profile.Username
	}
	lines := []string{strongStyle.Render(clipped(name, width)), accentStyle.Render("@" + profile.Username), ""}
	lines = append(lines, dimStyle.Render(clipped(privacy(profile.IsPrivate)+category(profile.Category), width)), "")
	lines = append(lines, metric("FOLLOWERS", profile.Followers, width), metric("FOLLOWING", profile.Following, width), metric("POSTS", profile.MediaCount, width), "")
	lines = append(lines, wrap(profile.Biography, width, 3), "", dimStyle.Render("SAVED EVIDENCE"))
	lines = append(lines, textStyle.Render(measured(item)), dimStyle.Render(coverage(item)))
	if len(item.Reels) > 0 && height >= 22 {
		lines = append(lines, "", dimStyle.Render("TOP SAVED REEL"), goodStyle.Bold(true).Render(number(item.Reels[0].Plays)+" plays"),
			dimStyle.Render(clipped(item.Reels[0].Caption, width)))
	}
	lines = append(lines, "", accentStyle.Render("enter  →  profile & reel metrics"))
	if profile.AvatarPath != "" {
		lines = append(lines, accentStyle.Render("i  →  profile picture in Kitty"))
	}
	return fitFrame(strings.Join(lines, "\n"), width, height)
}

func metric(label string, value *int64, width int) string {
	count := number(value)
	return dimStyle.Render(label) + strings.Repeat(" ", max(1, width-len(label)-len(count))) + strongStyle.Render(count)
}

func privacy(value *bool) string {
	if value == nil {
		return "Privacy unknown"
	}
	if *value {
		return "Private profile"
	}
	return "Public profile"
}

func category(value string) string {
	if value == "" {
		return ""
	}
	return "  ·  " + SafeText(value, 100)
}

func measured(item *snapshot.Snapshot) string {
	if item.Measurement().IsZero() {
		return "Measurement time unavailable"
	}
	return "Measured " + item.Measurement().Format("02 Jan 2006 15:04 UTC")
}

func coverage(item *snapshot.Snapshot) string {
	status := "feed coverage unknown"
	if item.FeedComplete != nil {
		status = "partial feed"
		if *item.FeedComplete {
			status = "complete accessible feed"
		}
	}
	return fmt.Sprintf("%d saved reels · %s", len(item.Reels), status)
}

func wrap(value string, width, lines int) string {
	wrapped := ansi.Wrap(SafeText(value, 4096), max(1, width), "")
	parts := strings.Split(wrapped, "\n")
	if len(parts) > lines {
		parts = parts[:lines]
		parts[lines-1] = ansi.Truncate(parts[lines-1], max(1, width-1), "") + "…"
	}
	return textStyle.Render(strings.Join(parts, "\n"))
}

func fitFrame(value string, width, height int) string {
	if width < 1 || height < 1 {
		return ""
	}
	lines := strings.Split(value, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for index := range lines {
		lines[index] = ansi.Truncate(lines[index], width, "")
		lines[index] += strings.Repeat(" ", max(0, width-ansi.StringWidth(lines[index])))
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines, "\n")
}

func helpView(width int) string {
	text := `A small, offline workspace for saved profile and reel data.

NAVIGATE                  EXPLORE
↑ ↓ / j k  Select         /       Search handles, names, category
enter      Open profile   f       All / measured / unmeasured / public
esc        Back           s       Handle / followers / latest
[ ]        Next profile   c       Clear search and filters
g / G      First / last   ?       Toggle this keyboard guide
q          Quit           ctrl+c  Quit immediately

KITTY IMAGES
i          Saved avatar   n / p   Select next / previous reel
t          Reel thumbnail          (in profile details)
Enter or Esc closes the native image preview and restores this screen.
Only local cached images are read; missing images stay unavailable.

Profile details scroll with ↑ ↓, j k, Page Up, Page Down.

DATA NOTES
— means unavailable, never zero. Plays, views, likes, comments,
reshares, and reposts are separate API-reported counters.
Timestamps are displayed in UTC. Feed coverage is explicit.
No scores, demographic guesses, hidden fetches, or network calls.`
	return textStyle.Render(ansi.Hardwrap(text, width, true))
}

func date(value string) string {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return "date unknown"
	}
	return parsed.UTC().Format("02 Jan 2006")
}
