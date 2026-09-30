package tui

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestNonKittyKeepsSavedTextUsable(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	m := demoModel(t)
	m.current().Profile.AvatarPath = "avatar.png"
	if command := m.previewImage(false); command != nil {
		t.Fatal("started image helper outside Kitty")
	}
	if !strings.Contains(m.previewNote, "require Kitty") {
		t.Fatal(m.previewNote)
	}
	if !strings.Contains(ansi.Strip(m.render()), "SAVED PROFILES") {
		t.Fatal("saved list disappeared")
	}
}

func TestReelSelectionAndStaleImageResult(t *testing.T) {
	m := demoModel(t)
	m.detail = true
	m.refreshDetail()
	m.selectReel(1)
	if m.reel != 1 || m.viewport.YOffset() != m.reelOffsets[1] {
		t.Fatal("reel selection did not scroll to its card")
	}
	m.move(1)
	if m.reel != 0 {
		t.Fatal("profile navigation retained stale reel")
	}
	generation := m.previewGeneration
	m.Update(tea.KeyPressMsg{Code: '/'})
	if m.imageReady(imageReadyMsg{generation: generation}) != nil {
		t.Fatal("stale preview opened during search")
	}
}

func TestImageArgumentsNeverContainURLsOrShell(t *testing.T) {
	command := imageCommand{width: 120, height: 38}
	arguments := command.arguments("/tmp/private-image.png", 42)
	joined := strings.Join(arguments, " ")
	for _, expected := range []string{"icat", "--hold", "--transfer-mode=stream", "--engine=builtin", "--place=116x30@2x4", "--image-id=42"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing %s", expected)
		}
	}
	for _, forbidden := range []string{"https://", "--clear", "sh -c", "Authorization"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("unsafe helper arguments: %s", joined)
		}
	}
}

func TestPreviewFailureRestoresOnlyItsScreenAndImage(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("TMPDIR", directory)
	var output bytes.Buffer
	command := imageCommand{ctx: context.Background(), binary: filepath.Join(directory, "missing-kitten"),
		data: []byte("synthetic test bytes"), width: 120, height: 38, title: "DEMO", demo: true, output: &output}
	if err := command.Run(); err == nil {
		t.Fatal("missing helper reported success")
	}
	text := output.String()
	if !strings.Contains(text, "\x1b[?1049h") || !strings.HasSuffix(text, "\x1b[?1049l") || !strings.Contains(text, "a=d,d=I,i=") {
		t.Fatal("preview did not restore screen and delete its own graphics")
	}
	if !strings.Contains(ansi.Strip(text), "SYNTHETIC DEMO") {
		t.Fatal("synthetic image preview not labeled")
	}
	if !strings.Contains(ansi.Strip(text), "Press Enter or Esc to return to saved metrics.") {
		t.Fatal("preview close instructions do not match kitten's hold behavior")
	}
	files, err := os.ReadDir(directory)
	if err != nil || len(files) != 0 {
		t.Fatalf("private preview file was not cleaned up: %v", err)
	}
}
