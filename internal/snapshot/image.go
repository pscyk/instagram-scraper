package snapshot

import (
	"bytes"
	"errors"
	"image"
	_ "image/jpeg" // Cached media may use JPEG; previews are re-encoded as PNG.
	"image/png"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
)

const imageBytesMax = 4 * 1024 * 1024

// ImagePNG reads only a named local asset and returns bounded, re-encoded pixels.
// Saved CDN URLs are metadata only: this path has no network implementation.
func (s *Snapshot) ImagePNG(relative string) ([]byte, error) {
	if err := validImagePath(relative); err != nil {
		return nil, err
	}
	var data []byte
	var err error
	if s.demo {
		data, err = demoFiles.ReadFile("demo/" + relative)
		if err != nil {
			return nil, errors.New("embedded demo image is unavailable")
		}
	} else {
		data, err = localImage(s.baseDir, relative)
		if err != nil {
			return nil, err
		}
	}
	return decodeImage(data)
}

func validImagePath(relative string) error {
	if relative == "" {
		return errors.New("no local image is saved; collect with --download-images first")
	}
	if len(relative) > 1024 || !fs.ValidPath(relative) || relative == "." ||
		strings.ContainsAny(relative, "\\:\x00") {
		return errors.New("image path must stay relative to its snapshot directory")
	}
	return nil
}

func localImage(directory, relative string) ([]byte, error) {
	if directory == "" {
		return nil, errors.New("image has no local snapshot directory")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, errors.New("cannot open the snapshot image directory")
	}
	defer func() { _ = root.Close() }()
	if err = rejectImageSymlinks(root, relative); err != nil {
		return nil, err
	}
	info, err := root.Lstat(relative)
	if err != nil || !info.Mode().IsRegular() || info.Size() > imageBytesMax {
		return nil, errors.New("cached image must be a regular file of at most 4 MiB")
	}
	file, err := root.Open(relative)
	if err != nil {
		return nil, errors.New("cannot read cached image")
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.New("cached image changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(file, imageBytesMax+1))
	if err != nil || len(data) > imageBytesMax {
		return nil, errors.New("cached image is unreadable or exceeds 4 MiB")
	}
	return data, nil
}

func rejectImageSymlinks(root *os.Root, relative string) error {
	current := ""
	parts := strings.Split(relative, "/")
	for index, component := range parts {
		current = path.Join(current, component)
		info, err := root.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 ||
			(index < len(parts)-1 && !info.IsDir()) {
			return errors.New("cached image path is missing or contains a symlink")
		}
	}
	return nil
}

func decodeImage(data []byte) ([]byte, error) {
	if len(data) > imageBytesMax {
		return nil, errors.New("cached image exceeds 4 MiB")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") || config.Width < 1 ||
		config.Height < 1 || config.Width > 2048 || config.Height > 2048 {
		return nil, errors.New("cached image must be JPEG/PNG at most 2048 by 2048 pixels")
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("cached image cannot be decoded")
	}
	var output bytes.Buffer
	if err := png.Encode(&output, decoded); err != nil {
		return nil, errors.New("cached image cannot be prepared")
	}
	return output.Bytes(), nil
}
