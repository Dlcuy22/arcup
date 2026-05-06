// Module: internal/retry
// Purpose: Provides a generic retry mechanism with delay for robust
// execution of archiving and uploading tasks.
//
// Key Components:
//   - Do(): Executes a function with retries and delay
//   - ParseError(): Parses common error strings into user-friendly text
//
// Dependencies:
//   - github.com/rs/zerolog/log
package retry

import (
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

/*
ParseError analyzes an error message for common underlying causes
and returns a short, human-friendly description.

	params:
		  err: the error to analyze
	returns:
		  string: friendly error cause, or empty if not recognized
*/
func ParseError(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	lower := strings.ToLower(s)

	if strings.Contains(lower, "no space left on device") {
		return "storage full"
	}
	if strings.Contains(lower, "connection refused") || strings.Contains(lower, "dial tcp") || strings.Contains(lower, "network is unreachable") || strings.Contains(lower, "no such host") {
		return "network disconnected/unreachable"
	}
	if strings.Contains(lower, "accessdenied") || strings.Contains(lower, "403 forbidden") || strings.Contains(lower, "401 unauthorized") || strings.Contains(lower, "auth") {
		return "access denied / unauthorized"
	}
	if strings.Contains(lower, "executable file not found in $path") {
		return "missing dependency in $PATH"
	}
	if strings.Contains(lower, "not found") || strings.Contains(lower, "404") || strings.Contains(lower, "directory not found") {
		return "not found"
	}

	return ""
}

/*
Do executes the given function up to maxAttempts times, waiting delay
between retries. If the function succeeds, it returns immediately.

	params:
		  ctxName: string context for logging (e.g. "archive" or "upload")
		  maxAttempts: total number of times to try
		  delay: duration string (e.g. "1m") to wait between attempts
		  fn: the function to execute
	returns:
		  error: the last error encountered if all attempts fail, or nil
*/
func Do(ctxName string, maxAttempts int, delay string, fn func() error) error {
	d, err := time.ParseDuration(delay)
	if err != nil {
		d = time.Minute // Default fallback
	}

	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		friendly := ParseError(lastErr)
		logMsg := lastErr.Error()
		if friendly != "" {
			logMsg = friendly
		}

		// Fast-fail for predictable, non-transient errors
		if friendly == "missing dependency in $PATH" || friendly == "access denied / unauthorized" {
			log.Error().
				Str("task", ctxName).
				Str("cause", logMsg).
				Msg("task failed permanently (fatal error)")
			return lastErr
		}

		if attempt < maxAttempts {
			log.Warn().
				Str("task", ctxName).
				Int("attempt", attempt).
				Int("max", maxAttempts).
				Str("cause", logMsg).
				Msgf("task failed, retrying in %s...", d)
			time.Sleep(d)
		} else {
			log.Error().
				Str("task", ctxName).
				Int("attempt", attempt).
				Int("max", maxAttempts).
				Str("cause", logMsg).
				Msg("task failed permanently")
		}
	}
	return lastErr
}
