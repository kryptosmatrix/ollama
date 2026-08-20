package mlxrunner

import (
	"context"
	"errors"
	"testing"
)

func TestAcquireTokenizerCancellation(t *testing.T) {
	runner := Runner{tokenizeSemaphore: make(chan struct{}, 1)}
	if err := runner.acquireTokenizer(context.Background()); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runner.acquireTokenizer(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}

	runner.releaseTokenizer()
	if err := runner.acquireTokenizer(context.Background()); err != nil {
		t.Fatal(err)
	}
	runner.releaseTokenizer()
}
