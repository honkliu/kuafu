package retry

import (
	"context"
	"time"
)

type Config struct {
	Attempts int
	Backoff  time.Duration
}

func DefaultConfig() Config {
	return Config{Attempts: 3, Backoff: 25 * time.Millisecond}
}

func Do(ctx context.Context, cfg Config, shouldRetry func(error) bool, operation func() error) error {
	if cfg.Attempts <= 0 {
		cfg.Attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < cfg.Attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := operation(); err != nil {
			lastErr = err
			if !shouldRetry(err) || attempt == cfg.Attempts-1 {
				return err
			}
			if cfg.Backoff > 0 {
				timer := time.NewTimer(cfg.Backoff)
				select {
				case <-ctx.Done():
					timer.Stop()
					return ctx.Err()
				case <-timer.C:
				}
			}
			continue
		}
		return nil
	}
	return lastErr
}
