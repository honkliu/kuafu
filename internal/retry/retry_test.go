package retry

import (
	"context"
	"fmt"
	"testing"
)

func TestDoRetriesWhenAllowed(t *testing.T) {
	attempts := 0
	err := Do(context.Background(), Config{Attempts: 3}, func(error) bool { return true }, func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("try again")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestDoDoesNotRetryPermanentError(t *testing.T) {
	attempts := 0
	err := Do(context.Background(), Config{Attempts: 3}, func(error) bool { return false }, func() error {
		attempts++
		return fmt.Errorf("permanent")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}
