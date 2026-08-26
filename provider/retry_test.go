// Copyright tecRacer Group 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"testing"
	"time"
)

// withShortRetryDelays temporarily swaps eventualConsistencyRetryDelays for a
// handful of millisecond-scale delays, so retry/backoff tests don't actually
// wait through the real multi-second production delays. It mutates a
// package-level var, so subtests using it must not run in parallel with each
// other or with subtests that assume the production delays.
func withShortRetryDelays(t *testing.T) {
	t.Helper()
	original := eventualConsistencyRetryDelays
	eventualConsistencyRetryDelays = []time.Duration{1 * time.Millisecond, 1 * time.Millisecond, 1 * time.Millisecond}
	t.Cleanup(func() { eventualConsistencyRetryDelays = original })
}

func TestRetryOnEventualConsistency(t *testing.T) {
	t.Run("succeeds first try", func(t *testing.T) {
		t.Parallel()
		calls := 0
		err := retryOnEventualConsistency(context.Background(),
			func(error) bool { return true },
			func() error { calls++; return nil },
		)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if calls != 1 {
			t.Fatalf("calls = %d, want 1", calls)
		}
	})

	t.Run("non-retryable error returns immediately", func(t *testing.T) {
		t.Parallel()
		calls := 0
		wantErr := errors.New("permanent failure")
		err := retryOnEventualConsistency(context.Background(),
			func(error) bool { return false },
			func() error { calls++; return wantErr },
		)
		if !errors.Is(err, wantErr) {
			t.Fatalf("err = %v, want %v", err, wantErr)
		}
		if calls != 1 {
			t.Fatalf("calls = %d, want 1 (should not retry)", calls)
		}
	})

	// The following two subtests mutate eventualConsistencyRetryDelays via
	// withShortRetryDelays, so they run sequentially, not in parallel.

	t.Run("retries until success within attempt budget", func(t *testing.T) {
		withShortRetryDelays(t)
		calls := 0
		err := retryOnEventualConsistency(context.Background(),
			func(error) bool { return true },
			func() error {
				calls++
				if calls < 3 {
					return errors.New("not found yet")
				}
				return nil
			},
		)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if calls != 3 {
			t.Fatalf("calls = %d, want 3", calls)
		}
	})

	t.Run("gives up after exhausting retry delays", func(t *testing.T) {
		withShortRetryDelays(t)
		calls := 0
		wantErr := errors.New("still not found")
		err := retryOnEventualConsistency(context.Background(),
			func(error) bool { return true },
			func() error { calls++; return wantErr },
		)
		if !errors.Is(err, wantErr) {
			t.Fatalf("err = %v, want %v", err, wantErr)
		}
		wantCalls := len(eventualConsistencyRetryDelays) + 1
		if calls != wantCalls {
			t.Fatalf("calls = %d, want %d", calls, wantCalls)
		}
	})

	t.Run("respects context cancellation while waiting", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		calls := 0
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()
		err := retryOnEventualConsistency(ctx,
			func(error) bool { return true },
			func() error { calls++; return errors.New("not found yet") },
		)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	})
}
