//go:build darwin

package osnotify

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	gosxnotifier "github.com/deckarep/gosx-notifier"
)

func Send(_ context.Context, title, message string) error {
	if _, err := ensureTerminalNotifierPath(); err != nil {
		return err
	}

	iconPath, err := ensurePNGPath()
	if err != nil {
		return err
	}

	notification := gosxnotifier.NewNotification(message)
	notification.Title = title
	notification.AppIcon = iconPath
	notification.Group = "notify-mcp"

	return notification.Push()
}

func ensureTerminalNotifierPath() (string, error) {
	// gosx-notifier 在 init 时把终端通知二进制解压到临时目录。
	// 临时目录可能被系统清理，导致路径失效，这里做存在性检查和自愈。
	if path := gosxnotifier.FinalPath; path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// 回退到系统已安装的 terminal-notifier（如 brew 安装的）。
	if sysPath, err := exec.LookPath("terminal-notifier"); err == nil {
		gosxnotifier.FinalPath = sysPath
		return sysPath, nil
	}

	return "", fmt.Errorf("terminal-notifier 不存在或已被清理（当前路径：%q）", gosxnotifier.FinalPath)
}
