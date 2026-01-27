package common

import "bytes"

// ReadSeekCloser wraps bytes.Reader to satisfy io.ReadSeekCloser.
// Useful for in-memory buffers that need seeking without managing a closer.
type ReadSeekCloser struct {
	*bytes.Reader
}

// Close is a no-op for in-memory readers.
func (r *ReadSeekCloser) Close() error { return nil }
