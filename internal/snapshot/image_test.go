package snapshot

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func testPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	pixels := image.NewRGBA(image.Rect(0, 0, width, height))
	pixels.Set(0, 0, color.RGBA{R: 180, G: 100, B: 230, A: 255})
	var data bytes.Buffer
	if err := png.Encode(&data, pixels); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func TestCachedImageIsBoundedAndReencoded(t *testing.T) {
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "profile.media"), 0o700); err != nil {
		t.Fatal(err)
	}
	relative := "profile.media/avatar.png"
	if err := os.WriteFile(filepath.Join(directory, relative), testPNG(t, 12, 8), 0o600); err != nil {
		t.Fatal(err)
	}
	saved := Snapshot{baseDir: directory}
	data, err := saved.ImagePNG(relative)
	if err != nil {
		t.Fatal(err)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != "png" || config.Width != 12 || config.Height != 8 {
		t.Fatalf("invalid preview: %s %+v %v", format, config, err)
	}
}

func TestImagePathsCannotEscapeSnapshotOrFollowSymlinks(t *testing.T) {
	directory, external := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "secret.png"), testPNG(t, 2, 2), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(directory, "media")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(external, "secret.png"), filepath.Join(directory, "avatar.png")); err != nil {
		t.Fatal(err)
	}
	saved := Snapshot{baseDir: directory}
	for _, name := range []string{"", ".", "../secret.png", "/secret.png", "media/../../secret.png", "media/secret.png", "avatar.png", `C:\secret.png`, "https://example.com/a.png"} {
		if _, err := saved.ImagePNG(name); err == nil {
			t.Errorf("accepted unsafe image path %q", name)
		}
	}
}

func TestImageRejectsUnsupportedOrOversizedPixels(t *testing.T) {
	for _, data := range [][]byte{[]byte("not an image"), []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), testPNG(t, 2049, 1), make([]byte, imageBytesMax+1)} {
		if _, err := decodeImage(data); err == nil {
			t.Fatal("accepted invalid or oversized image")
		}
	}
}

func TestSnapshotLoadKeepsLocalAssetBase(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "profile.json"), []byte(minimal), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "avatar.png"), testPNG(t, 4, 4), 0o600); err != nil {
		t.Fatal(err)
	}
	items, err := Load(filepath.Join(directory, "profile.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := items[0].ImagePNG("avatar.png"); err != nil {
		t.Fatal(err)
	}
}

func TestEverySavedDemoImageLoadsFromEmbeddedAssets(t *testing.T) {
	items, err := Demo()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.Profile.AvatarPath == "" {
			t.Fatalf("demo %s missing avatar", item.Profile.Username)
		}
		if _, err := item.ImagePNG(item.Profile.AvatarPath); err != nil {
			t.Fatalf("demo avatar: %v", err)
		}
	}
	checked := make(map[string]bool)
	for _, item := range items {
		for _, reel := range item.Reels {
			if reel.ThumbnailPath == "" {
				t.Fatalf("demo %s missing thumbnail", reel.Code)
			}
			if checked[reel.ThumbnailPath] {
				continue
			}
			if _, err := item.ImagePNG(reel.ThumbnailPath); err != nil {
				t.Fatalf("demo thumbnail: %v", err)
			}
			checked[reel.ThumbnailPath] = true
		}
	}
}
