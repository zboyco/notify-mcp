//go:build windows

package osnotify

import (
	"context"

	toast "git.sr.ht/~jackmordaunt/go-toast"
)

const toastAppID = "notify-mcp"

func Send(_ context.Context, title, message string) error {
	iconPath, err := ensurePNGPath()
	if err != nil {
		return err
	}

	notification := toast.Notification{
		AppID: toastAppID,
		Title: title,
		Body:  message,
		Icon:  iconPath,
	}

	return notification.Push()
}
