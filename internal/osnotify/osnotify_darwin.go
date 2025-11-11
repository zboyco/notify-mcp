//go:build darwin

package osnotify

import (
	"context"

	gosxnotifier "github.com/deckarep/gosx-notifier"
)

func Send(_ context.Context, title, message string) error {
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
