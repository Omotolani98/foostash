package foostash

import (
	"context"
	"log/slog"
	"time"
)

// MinWatchInterval is the floor enforced by Watch to avoid hammering the server.
const MinWatchInterval = 5 * time.Second

// WatchOption customizes Watch behavior.
type WatchOption func(*watchOptions)

type watchOptions struct {
	logger *slog.Logger
}

// WithLogger attaches a slog.Logger that Watch uses to report transient
// polling errors. If unset, errors during polling are silently dropped —
// Watch never closes its channel on a transient failure.
func WithLogger(l *slog.Logger) WatchOption {
	return func(o *watchOptions) { o.logger = l }
}

// Watch polls the server on the given interval and emits a Snapshot whenever
// any key's version changes (including key add/remove). The returned channel
// is closed when ctx is canceled.
//
// The initial snapshot is always emitted once, synchronously, before the
// first tick — so callers can block on the first receive to load their
// initial config. If the initial Pull fails, Watch returns the error and
// no channel.
//
// Intervals below MinWatchInterval are clamped up.
func (c *Client) Watch(ctx context.Context, interval time.Duration, opts ...WatchOption) (<-chan Snapshot, error) {
	if interval < MinWatchInterval {
		interval = MinWatchInterval
	}
	o := &watchOptions{}
	for _, fn := range opts {
		fn(o)
	}

	initial, err := c.pullSnapshot(ctx)
	if err != nil {
		return nil, err
	}

	ch := make(chan Snapshot, 1)
	ch <- initial

	go c.watchLoop(ctx, interval, initial.Versions, ch, o)
	return ch, nil
}

func (c *Client) watchLoop(ctx context.Context, interval time.Duration, last map[string]int, ch chan<- Snapshot, o *watchOptions) {
	defer close(ch)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snap, err := c.pullSnapshot(ctx)
			if err != nil {
				if o.logger != nil {
					o.logger.Warn("foostash watch: poll failed", "error", err)
				}
				continue
			}
			if versionsEqual(last, snap.Versions) {
				continue
			}
			last = snap.Versions
			select {
			case ch <- snap:
			case <-ctx.Done():
				return
			}
		}
	}
}

func versionsEqual(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		if vb, ok := b[k]; !ok || vb != va {
			return false
		}
	}
	return true
}
