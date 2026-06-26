package errors

import (
	"fmt"
	"testing"
)

func TestIsTransient(t *testing.T) {
	err := Wrap(KindTransient, "runtime unavailable", fmt.Errorf("timeout"))
	if !IsTransient(err) {
		t.Fatalf("expected transient error")
	}
}

func TestIsKindReturnsFalseForPlainError(t *testing.T) {
	if IsKind(fmt.Errorf("plain"), KindTransient) {
		t.Fatal("plain error should not match typed kind")
	}
}
