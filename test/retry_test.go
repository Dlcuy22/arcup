// Module: test/retry_test.go
// Purpose: Unit tests for the retry mechanism, specifically testing intentional
// failure logic, delays, attempt counting, and error string parsing.
package test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dlcuy22/arcup/internal/retry"
)

func TestRetry_Do_IntentionalFailure(t *testing.T) {
	attempts := 0
	maxAttempts := 3
	expectedErr := errors.New("no space left on device")

	start := time.Now()
	
	err := retry.Do(context.Background(), "test_task", maxAttempts, "100ms", func() error {
		attempts++
		return expectedErr
	})

	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if attempts != maxAttempts {
		t.Fatalf("expected %d attempts, got %d", maxAttempts, attempts)
	}

	if elapsed < 200*time.Millisecond {
		t.Fatalf("expected execution to take at least 200ms due to delays, took %v", elapsed)
	}

	friendly := retry.ParseError(err)
	if friendly != "storage full" {
		t.Fatalf("expected parsed error 'storage full', got '%s'", friendly)
	}
}

func TestRetry_ParseError(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{errors.New("dial tcp 127.0.0.1: connection refused"), "network disconnected/unreachable"},
		{errors.New("HTTP 403 Forbidden AccessDenied"), "access denied / unauthorized"},
		{errors.New("file not found"), "not found"},
		{errors.New("exit status 130"), "interrupted"},
		{errors.New("unknown error 123"), ""},
	}

	for _, tt := range tests {
		actual := retry.ParseError(tt.err)
		if actual != tt.expected {
			t.Errorf("for error %q: expected %q, got %q", tt.err.Error(), tt.expected, actual)
		}
	}
}

func TestRetry_Do_SucceedsEventually(t *testing.T) {
	attempts := 0
	
	err := retry.Do(context.Background(), "test_task_recover", 3, "10ms", func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}
