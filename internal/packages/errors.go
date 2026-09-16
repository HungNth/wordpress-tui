package packages

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
)

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("HTTP %d", e.StatusCode)
}

// MetadataContractError marks errors caused by malformed, empty, or invalid metadata responses.
type MetadataContractError struct {
	Err error
}

func (e *MetadataContractError) Error() string {
	return fmt.Sprintf("metadata contract violation: %v", e.Err)
}

func (e *MetadataContractError) Unwrap() error {
	return e.Err
}

// IsTransientError identifies temporary network interruptions, timeouts, and server errors.
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}

	// Metadata contract errors must NEVER trigger stale fallback
	var contractErr *MetadataContractError
	if errors.As(err, &contractErr) {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return true
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.IsTemporary || dnsErr.Timeout() || dnsErr.IsNotFound {
			return true
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}

	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.StatusCode {
		case 408, 429, 500, 502, 503, 504:
			return true
		default:
			return false
		}
	}

	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "temporary failure") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "i/o timeout") {
		return true
	}

	return false
}
