package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const minimal = `{"profile":{"username":"synthetic"},"ranked_reels":[{"play_count":null,"like_count":0}]}`

func TestParseUnknownAndZeroStayDistinct(t *testing.T) {
	item, err := Parse([]byte(minimal))
	if err != nil {
		t.Fatal(err)
	}
	if item.Profile.Followers != nil || item.Reels[0].Plays != nil {
		t.Fatal("unknown counter became measured")
	}
	if item.Reels[0].Likes == nil || *item.Reels[0].Likes != 0 {
		t.Fatal("measured zero lost")
	}
	encoded, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"play_count":null`) || !strings.Contains(string(encoded), `"like_count":0`) {
		t.Fatal("JSON counter semantics changed")
	}
}

func TestParseRejectsInvalidInputs(t *testing.T) {
	for _, data := range []string{`{}`, `null`, minimal + ` {}`, `{"profile":{"username":"wrong/name"}}`,
		`{"profile":{"username":"valid","follower_count":-1}}`,
		`{"profile":{"username":"valid"},"measured_at_utc":"yesterday"}`,
		`{"profile":{"username":"valid"},"ranked_reels":[{"play_count":1.5}]}`} {
		if _, err := Parse([]byte(data)); err == nil {
			t.Errorf("accepted invalid snapshot: %s", data)
		}
	}
}

func TestLoadBoundedDirectoryAndSymlinks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.json")
	if err := os.WriteFile(path, []byte(minimal), 0o600); err != nil {
		t.Fatal(err)
	}
	items, err := Load(dir)
	if err != nil || len(items) != 1 {
		t.Fatalf("load directory: count=%d err=%v", len(items), err)
	}
	link := filepath.Join(dir, "linked.json")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(link); err == nil {
		t.Fatal("symlink accepted")
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("directory silently accepted symlink")
	}
}

func TestLoadRefusesOversizedFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.json")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = file.Truncate(fileBytesMax + 1); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("oversized file accepted")
	}
}

func TestDemoIsDeterministicSyntheticAndOffline(t *testing.T) {
	first, err := Demo()
	if err != nil {
		t.Fatal(err)
	}
	second, err := Demo()
	if err != nil {
		t.Fatal(err)
	}
	left, _ := json.Marshal(first)
	right, _ := json.Marshal(second)
	if string(left) != string(right) || len(first) < 4 {
		t.Fatal("demo is not deterministic")
	}
	for _, item := range first {
		if !strings.HasPrefix(item.Profile.Username, "demo_") || !strings.HasPrefix(item.Source, "synthetic demo") {
			t.Fatal("demo identity is not labeled synthetic")
		}
		if item.Profile.ExternalURL != "" {
			t.Fatal("demo contains real external URL")
		}
		for _, reel := range item.Reels {
			if reel.URL != "" {
				t.Fatal("demo contains real reel URL")
			}
		}
	}
}

func TestMeasurementUsesUTC(t *testing.T) {
	item := Snapshot{MeasuredAt: "2026-09-21T01:15:00+05:30"}
	if got := item.Measurement().Format("2006-01-02 15:04 MST"); got != "2026-09-20 19:45 UTC" {
		t.Fatal(got)
	}
}
