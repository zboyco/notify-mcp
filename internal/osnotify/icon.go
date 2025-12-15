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
	pngMu   sync.Mutex
)

func ensurePNGPath() (string, error) {
	pngOnce.Do(func() {
		pngPath, pngErr = writeTempPNG()
	})
	if pngErr != nil {
		return "", pngErr
	}

	if _, err := os.Stat(pngPath); err == nil {
		return pngPath, nil
	}

	// 临时文件被系统清理后需要重新写入。
	pngMu.Lock()
	defer pngMu.Unlock()

	// 双重检查，避免并发重复写文件。
	if _, err := os.Stat(pngPath); err == nil {
		return pngPath, nil
	}

	pngPath, pngErr = writeTempPNG()
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
