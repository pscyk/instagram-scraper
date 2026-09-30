package tui

import (
	"math"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/pscyk/instagram-scraper/internal/snapshot"
)

func demoModel(t *testing.T) *Model {
	t.Helper()
	return New(unitSnapshots())
}

// UI behavior fixtures stay in tests, independent of the user-visible saved demo.
func unitSnapshots() []snapshot.Snapshot {
	low, high := int64(284600), int64(421900)
	public := false
	reels := make([]snapshot.Reel, 7)
	for index := range reels {
		reels[index] = snapshot.Reel{Code: "UNIT", Caption: "Unit-test caption", PublishedAt: "2026-09-21T12:30:00Z"}
	}
	return []snapshot.Snapshot{
		{Profile: snapshot.Profile{Username: "fixture_lumen", FullName: "Lumen Fixture", Followers: &low, IsPrivate: &public},
			MeasuredAt: "2026-09-21T12:30:00Z", Reels: reels},
		{Profile: snapshot.Profile{Username: "fixture_motionlab", FullName: "Motion Fixture", Followers: &high, IsPrivate: &public},
			MeasuredAt: "2026-09-21T12:31:00Z", Reels: reels},
		{Profile: snapshot.Profile{Username: "fixture_unknown", FullName: "Unknown Fixture"}},
	}
}

func key(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

func TestEmptyFilterFromDetailDoesNotPanic(t *testing.T) {
	items := unitSnapshots()
	m := New(items[:1])
	m.detail = true
	m.filter = 2
	m.applyFilters()
	if m.detail {
		t.Fatal("empty selection remained in detail mode")
	}
	if !strings.Contains(ansi.Strip(m.render()), "No matching profiles") {
		t.Fatal("missing empty state")
	}
}

func TestSearchAndFiltersPreserveUnknown(t *testing.T) {
	m := demoModel(t)
	m.query = "lumen"
	m.applyFilters()
	if len(m.visible) != 1 || m.current().Profile.Username != "fixture_lumen" {
		t.Fatal("search mismatch")
	}
	m.query = ""
	m.filter = 2
	m.applyFilters()
	if len(m.visible) != 1 || m.current().Profile.Followers != nil {
		t.Fatal("unmeasured filter mismatch")
	}
	m.sort = 1
	m.filter = 0
	m.applyFilters()
	if m.current().Profile.Username != "fixture_motionlab" {
		t.Fatal("followers sort mismatch")
	}
}

func TestRenderBoundsAndProfileIdentity(t *testing.T) {
	for _, size := range [][2]int{{120, 38}, {90, 26}, {54, 16}, {40, 10}, {1, 1}} {
		m := demoModel(t)
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, detail := range []bool{false, true} {
			m.detail = detail
			m.refreshDetail()
			rendered := m.render()
			lines := strings.Split(rendered, "\n")
			if len(lines) > size[1] {
				t.Fatalf("%v: too many lines %d", size, len(lines))
			}
			for _, line := range lines {
				if ansi.StringWidth(line) > size[0] {
					t.Fatalf("%v: line exceeds width", size)
				}
			}
			if size[0] >= 54 && !strings.Contains(ansi.Strip(rendered), "INSTAGRAM SCRAPER") {
				t.Fatal("app name absent")
			}
			for _, obsolete := range []string{"REAL DATA", "SAVED SNAPSHOT", "OFFLINE", "RAW COUNTERS", "Source:", "no inferred metrics"} {
				if strings.Contains(ansi.Strip(rendered), obsolete) {
					t.Fatalf("obsolete product copy remains: %s", obsolete)
				}
			}
		}
	}
}

func TestSafeTextNeutralizesTerminalAndBidiControls(t *testing.T) {
	malicious := "safe\x1b]52;c;secret\a\x1b[2J\u202Ereversed\x00\x9b31m"
	value := SafeText(malicious, 100)
	if strings.ContainsAny(value, "\x1b\x00\x9b\u202E\a") || strings.Contains(value, "secret") {
		t.Fatalf("unsafe text %q", value)
	}
}

func TestRawCountersAndUTCDate(t *testing.T) {
	zero := int64(0)
	large := int64(1234567)
	if number(nil) != "—" || number(&zero) != "0" || number(&large) != "1,234,567" {
		t.Fatal("raw count formatting changed")
	}
	if date("2026-09-21T01:15:00+05:30") != "20 Sep 2026" {
		t.Fatal("publication day is not UTC")
	}
}

func TestLargeCountersNeverBecomeNumericPrefixes(t *testing.T) {
	maximum := int64(math.MaxInt64)
	for _, width := range []int{46, 80, 112} {
		row := ansi.Strip(metricRow([]string{"PLAYS", "LIKES", "COMMENTS"}, []*int64{&maximum, &maximum, &maximum}, width))
		if strings.Count(row, "9,223,372,036,854,775,807") != 3 {
			t.Fatalf("width %d silently truncated count: %s", width, row)
		}
		for _, line := range strings.Split(row, "\n") {
			if ansi.StringWidth(line) > width {
				t.Fatalf("width %d overflow", width)
			}
		}
	}
}

func TestPastedSearchCanBeApplied(t *testing.T) {
	m := demoModel(t)
	m.input.Focus()
	m.Update(tea.PasteMsg{Content: "lumen"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(m.visible) != 1 || m.current().Profile.Username != "fixture_lumen" {
		t.Fatal("pasted search was lost")
	}
}
