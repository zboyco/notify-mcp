package osnotify

import (
	"context"
	"testing"
)

func TestSendDispatchesNotification(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	if err := Send(ctx, "Notify MCP", "Triggered from go test"); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
}
