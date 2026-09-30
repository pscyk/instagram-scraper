package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"os/exec"
	"time"

	tea "charm.land/bubbletea/v2"
)

type imageReadyMsg struct {
	generation    uint64
	data          []byte
	binary, title string
	err           error
}

type previewDoneMsg struct{ err error }

func (m *Model) previewImage(thumbnail bool) tea.Cmd {
	item := m.current()
	if item == nil {
		return nil
	}
	relative, title := item.Profile.AvatarPath, "@"+item.Profile.Username+" / PROFILE PICTURE"
	if thumbnail {
		if !m.detail || len(item.Reels) == 0 {
			m.previewNote = "Open a profile to preview its reels."
			return nil
		}
		relative = item.Reels[m.reel].ThumbnailPath
		title = fmt.Sprintf("@%s / REEL %02d THUMBNAIL", item.Profile.Username, m.reel+1)
	}
	if relative == "" {
		m.previewNote = "Image unavailable. Collect with --download-images."
		return nil
	}
	binary, err := kittyBinary()
	if err != nil {
		m.previewNote = err.Error()
		return nil
	}
	m.previewGeneration++
	generation, saved := m.previewGeneration, *item
	m.previewNote = "Opening image…"
	return func() tea.Msg {
		data, err := saved.ImagePNG(relative)
		return imageReadyMsg{generation: generation, data: data, binary: binary, title: title, err: err}
	}
}

func (m *Model) imageReady(message imageReadyMsg) tea.Cmd {
	if message.generation != m.previewGeneration {
		return nil
	}
	if message.err != nil {
		m.previewNote = message.err.Error()
		return nil
	}
	m.previewNote = ""
	return tea.Exec(&imageCommand{ctx: m.ctx, binary: message.binary, data: message.data,
		title: message.title, width: m.width, height: m.height},
		func(err error) tea.Msg { return previewDoneMsg{err: err} })
}

func kittyBinary() (string, error) {
	if os.Getenv("TERM") != "xterm-kitty" {
		return "", errors.New("Image previews require Kitty.")
	}
	if binary, err := exec.LookPath("kitten"); err == nil {
		return binary, nil
	}
	const macBinary = "/Applications/kitty.app/Contents/MacOS/kitten"
	if info, err := os.Stat(macBinary); err == nil && info.Mode().IsRegular() {
		return macBinary, nil
	}
	return "", errors.New("Kitty's kitten command was not found.")
}

// imageCommand owns terminal I/O only after Bubble Tea releases the terminal.
// The helper receives one re-encoded local PNG, never a URL or captured header.
type imageCommand struct {
	ctx           context.Context
	binary, title string
	data          []byte
	width, height int
	input         io.Reader
	output        io.Writer
}

func (c *imageCommand) SetStdin(reader io.Reader)  { c.input = reader }
func (c *imageCommand) SetStdout(writer io.Writer) { c.output = writer }
func (c *imageCommand) SetStderr(io.Writer)        {}

func (c *imageCommand) Run() error {
	file, err := os.CreateTemp("", "instagram-preview-*.png")
	if err != nil {
		return errors.New("cannot prepare a private image preview file")
	}
	defer func() { _ = os.Remove(file.Name()) }()
	_, writeErr := file.Write(c.data)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		return errors.New("cannot prepare image preview")
	}
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Minute)
	defer cancel()
	id := rand.Uint32()
	if id == 0 {
		id = 1
	}
	if _, err = io.WriteString(c.output, c.frame()); err != nil {
		return errors.New("cannot open image preview screen")
	}
	defer func() { _, _ = fmt.Fprintf(c.output, "\x1b_Ga=d,d=I,i=%d,q=2\x1b\\\x1b[?1049l", id) }()
	command := exec.CommandContext(ctx, c.binary, c.arguments(file.Name(), id)...)
	command.Stdin, command.Stdout, command.Stderr = c.input, c.output, io.Discard
	if command.Run() != nil {
		return errors.New("Could not open the image in Kitty.")
	}
	return nil
}

func (c *imageCommand) arguments(filename string, id uint32) []string {
	return []string{"icat", "--hold", "--stdin=no", "--transfer-mode=stream", "--engine=builtin",
		"--loop=0", "--detection-timeout=2", "--scale-up",
		fmt.Sprintf("--place=%dx%d@2x4", max(1, c.width-4), max(1, c.height-8)),
		fmt.Sprintf("--image-id=%d", id), filename}
}

func (c *imageCommand) frame() string {
	return "\x1b[?1049h\x1b[2J\x1b[H\n  " + strongStyle.Render(clipped(c.title, max(1, c.width-4)))
}
