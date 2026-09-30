package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/pscyk/instagram-scraper/internal/snapshot"
)

func TestJSONDemoWithoutTerminalOrCredentials(t *testing.T) {
	var output, diagnostic bytes.Buffer
	code := run(context.Background(), []string{"--demo", "--json"}, &output, &diagnostic)
	if code != 0 {
		t.Fatalf("exit=%d: %s", code, diagnostic.String())
	}
	var items []snapshot.Snapshot
	if err := json.Unmarshal(output.Bytes(), &items); err != nil || len(items) == 0 {
		t.Fatalf("invalid JSON demo: %v", err)
	}
}

func TestInputOptions(t *testing.T) {
	for _, args := range [][]string{{"--input", "results"}, {"results"}} {
		options, err := parseOptions(args, io.Discard)
		if err != nil || options.path != "results" {
			t.Fatalf("input: %+v %v", options, err)
		}
	}
	for _, args := range [][]string{{}, {"--demo", "results"}, {"--demo", "--input", "results"}, {"--input", "a", "b"}, {"a", "b"}} {
		if _, err := parseOptions(args, io.Discard); err == nil {
			t.Fatalf("invalid options accepted: %v", args)
		}
	}
}
