package snapshot

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	fileBytesMax  = 8 * 1024 * 1024
	totalBytesMax = 32 * 1024 * 1024
	filesMax      = 128
	reelsMax      = 10000
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_.]{1,30}$`)

//go:embed demo
var demoFiles embed.FS

// Load reads one file or a non-recursive directory of JSON results. Symlinks are refused.
func Load(path string) ([]Snapshot, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, errors.New("cannot read snapshot path")
	}
	paths := []string{path}
	if info.IsDir() {
		paths, err = directoryFiles(path)
		if err != nil {
			return nil, err
		}
	}
	result := make([]Snapshot, 0, len(paths))
	total := 0
	for _, name := range paths {
		data, err := readFile(name)
		if err != nil {
			return nil, fmt.Errorf("snapshot %q: %w", filepath.Base(name), err)
		}
		total += len(data)
		if total > totalBytesMax {
			return nil, errors.New("snapshot set exceeds 32 MiB")
		}
		item, err := Parse(data)
		if err != nil {
			return nil, fmt.Errorf("snapshot %q: %w", filepath.Base(name), err)
		}
		item.Source = filepath.Base(name)
		item.baseDir, err = filepath.Abs(filepath.Dir(name))
		if err != nil {
			return nil, errors.New("cannot resolve snapshot directory")
		}
		result = append(result, item)
	}
	if len(result) == 0 {
		return nil, errors.New("no JSON snapshots found")
	}
	return result, nil
}

func directoryFiles(path string) ([]string, error) {
	directory, err := os.Open(path)
	if err != nil {
		return nil, errors.New("cannot open snapshot directory")
	}
	defer func() { _ = directory.Close() }()
	// Bound all entries, not only matches, so unrelated contents cannot cause unbounded work.
	names, err := directory.Readdirnames(4097)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, errors.New("cannot list snapshots")
	}
	if len(names) > 4096 {
		return nil, errors.New("snapshot directory exceeds 4096 entries")
	}
	paths := make([]string, 0)
	for _, name := range names {
		if strings.HasSuffix(strings.ToLower(name), ".json") && !strings.HasPrefix(name, ".") {
			paths = append(paths, filepath.Join(path, name))
		}
	}
	if len(paths) > filesMax {
		return nil, errors.New("at most 128 JSON snapshots may be opened")
	}
	sort.Strings(paths)
	return paths, nil
}

func readFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("expected a regular file, not a symlink")
	}
	if info.Size() > fileBytesMax {
		return nil, errors.New("snapshot exceeds 8 MiB")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New("cannot open file")
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.New("file changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(file, fileBytesMax+1))
	if err != nil || len(data) > fileBytesMax {
		return nil, errors.New("snapshot unreadable or exceeds 8 MiB")
	}
	return data, nil
}

// Parse validates shape and raw counters without inferring unavailable values.
func Parse(data []byte) (Snapshot, error) {
	var result Snapshot
	if len(data) > fileBytesMax {
		return result, errors.New("snapshot exceeds 8 MiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if decoder.Decode(&result) != nil {
		return result, errors.New("invalid snapshot JSON")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return result, errors.New("expected exactly one JSON snapshot")
	}
	if !usernamePattern.MatchString(result.Profile.Username) {
		return result, errors.New("missing or invalid profile username")
	}
	if len(result.Reels) > reelsMax {
		return result, errors.New("snapshot exceeds 10000 reels")
	}
	if result.MeasuredAt != "" {
		if _, err := time.Parse(time.RFC3339Nano, result.MeasuredAt); err != nil {
			return result, errors.New("invalid measured_at_utc timestamp")
		}
	}
	if err := validateCounts(&result); err != nil {
		return result, err
	}
	return result, nil
}

func validateCounts(item *Snapshot) error {
	counts := []*int64{item.Profile.Followers, item.Profile.Following, item.Profile.MediaCount,
		item.PostsScanned, item.ReelsChecked, item.MissingPlays}
	for i := range item.Reels {
		reel := &item.Reels[i]
		counts = append(counts, reel.Plays, reel.InstagramPlays, reel.FacebookPlays,
			reel.Views, reel.Likes, reel.Comments, reel.Reshares, reel.Reposts, reel.PremiumReactions)
		if reel.Duration != nil && *reel.Duration < 0 {
			return errors.New("negative reel duration")
		}
	}
	for _, count := range counts {
		if count != nil && *count < 0 {
			return errors.New("negative raw counter")
		}
	}
	return nil
}

// Demo returns fictional, embedded snapshots and performs no filesystem or network lookup.
func Demo() ([]Snapshot, error) {
	entries, err := demoFiles.ReadDir("demo")
	if err != nil {
		return nil, errors.New("demo unavailable")
	}
	result := make([]Snapshot, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := demoFiles.ReadFile("demo/" + entry.Name())
		if err != nil {
			return nil, errors.New("demo unavailable")
		}
		item, err := Parse(data)
		if err != nil {
			return nil, err
		}
		item.Source = "synthetic demo / " + entry.Name()
		item.demo = true
		result = append(result, item)
	}
	return result, nil
}
