package osnotify

import (
	"os"
	"path/filepath"
	"sync"

	_ "embed"
)

//go:embed icon.png
var iconPNG []byte

var (
	pngOnce sync.Once
	pngPath string
	pngErr  error
)

func ensurePNGPath() (string, error) {
	pngOnce.Do(func() {
		pngPath, pngErr = writeTempPNG()
	})
	return pngPath, pngErr
}

func writeTempPNG() (string, error) {
	f, err := os.CreateTemp("", "notify-mcp-icon-*.png")
	if err != nil {
		return "", err
	}

	if _, err := f.Write(iconPNG); err != nil {
		f.Close()
		return "", err
	}

	if err := f.Close(); err != nil {
		return "", err
	}

	path, err := filepath.Abs(f.Name())
	if err != nil {
		path = f.Name()
	}

	return path, nil
}
