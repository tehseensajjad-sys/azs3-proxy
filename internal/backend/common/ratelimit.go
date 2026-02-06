package common

import (
	"io"
	"math"
	"sync"
	"time"
)

// BandwidthLimiter caps throughput in bytes/sec using token buckets.
// Caps can be applied separately for downloads (Azure -> proxy), uploads
// (proxy -> Azure), and a combined cap shared across both directions.
type BandwidthLimiter struct {
	download *tokenBucket
	upload   *tokenBucket
	combined *tokenBucket
}

// NewBandwidthLimiter builds a limiter from Mbps caps. Returns nil when all caps are zero/negative.
func NewBandwidthLimiter(capReadMbps, capWriteMbps, capCombinedMbps float64) *BandwidthLimiter {
	if capReadMbps <= 0 && capWriteMbps <= 0 && capCombinedMbps <= 0 {
		return nil
	}

	toBps := func(mbps float64) float64 {
		return mbps * 1024 * 1024 / 8
	}

	return &BandwidthLimiter{
		download: newTokenBucket(toBps(capReadMbps)),
		upload:   newTokenBucket(toBps(capWriteMbps)),
		combined: newTokenBucket(toBps(capCombinedMbps)),
	}
}

// WrapDownload throttles a download stream (Azure -> proxy/client).
func (l *BandwidthLimiter) WrapDownload(rc io.ReadCloser) io.ReadCloser {
	if l == nil || rc == nil {
		return rc
	}
	buckets := l.buckets(true)
	if len(buckets) == 0 {
		return rc
	}
	return &throttledReadCloser{src: rc, buckets: buckets}
}

// WrapUpload throttles an upload reader (proxy -> Azure).
func (l *BandwidthLimiter) WrapUpload(r io.Reader) io.Reader {
	if l == nil || r == nil {
		return r
	}
	buckets := l.buckets(false)
	if len(buckets) == 0 {
		return r
	}
	if rc, ok := r.(io.ReadCloser); ok {
		return &throttledReadCloser{src: rc, buckets: buckets}
	}
	return &throttledReader{src: r, buckets: buckets}
}

// WrapUploadSeek throttles an upload reader that must also satisfy io.ReadSeekCloser.
func (l *BandwidthLimiter) WrapUploadSeek(r io.ReadSeekCloser) io.ReadSeekCloser {
	if l == nil || r == nil {
		return r
	}
	buckets := l.buckets(false)
	if len(buckets) == 0 {
		return r
	}
	return &throttledReadSeekCloser{src: r, buckets: buckets}
}

func (l *BandwidthLimiter) buckets(isDownload bool) []*tokenBucket {
	if l == nil {
		return nil
	}
	buckets := make([]*tokenBucket, 0, 2)
	if isDownload {
		if l.download != nil {
			buckets = append(buckets, l.download)
		}
	} else {
		if l.upload != nil {
			buckets = append(buckets, l.upload)
		}
	}
	if l.combined != nil {
		buckets = append(buckets, l.combined)
	}
	return buckets
}

const throttleChunk = 128 * 1024 // 128KiB

type tokenBucket struct {
	capacity float64
	tokens   float64
	fillRate float64
	last     time.Time
	mu       sync.Mutex
}

func newTokenBucket(bps float64) *tokenBucket {
	if bps <= 0 {
		return nil
	}
	now := time.Now()
	return &tokenBucket{
		capacity: bps,
		tokens:   bps,
		fillRate: bps,
		last:     now,
	}
}

// take blocks until n tokens are available, then consumes them.
func (tb *tokenBucket) take(n int) {
	if tb == nil || n <= 0 {
		return
	}

	tb.mu.Lock()
	defer tb.mu.Unlock()

	for {
		tb.refillLocked()
		if tb.tokens >= float64(n) {
			tb.tokens -= float64(n)
			return
		}

		deficit := float64(n) - tb.tokens
		wait := time.Duration(deficit / tb.fillRate * float64(time.Second))
		if wait < time.Millisecond {
			wait = time.Millisecond
		}
		tb.mu.Unlock()
		time.Sleep(wait)
		tb.mu.Lock()
	}
}

func (tb *tokenBucket) refillLocked() {
	now := time.Now()
	if !now.After(tb.last) {
		return
	}
	elapsed := now.Sub(tb.last).Seconds()
	tb.tokens = math.Min(tb.capacity, tb.tokens+tb.fillRate*elapsed)
	tb.last = now
}

type throttledReader struct {
	src     io.Reader
	buckets []*tokenBucket
}

func (tr *throttledReader) Read(p []byte) (int, error) {
	if len(p) > throttleChunk {
		p = p[:throttleChunk]
	}
	n, err := tr.src.Read(p)
	if n > 0 {
		for _, b := range tr.buckets {
			b.take(n)
		}
	}
	return n, err
}

type throttledReadCloser struct {
	src     io.ReadCloser
	buckets []*tokenBucket
}

func (trc *throttledReadCloser) Read(p []byte) (int, error) {
	if len(p) > throttleChunk {
		p = p[:throttleChunk]
	}
	n, err := trc.src.Read(p)
	if n > 0 {
		for _, b := range trc.buckets {
			b.take(n)
		}
	}
	return n, err
}

func (trc *throttledReadCloser) Close() error {
	return trc.src.Close()
}

type throttledReadSeekCloser struct {
	src     io.ReadSeekCloser
	buckets []*tokenBucket
}

func (trc *throttledReadSeekCloser) Read(p []byte) (int, error) {
	if len(p) > throttleChunk {
		p = p[:throttleChunk]
	}
	n, err := trc.src.Read(p)
	if n > 0 {
		for _, b := range trc.buckets {
			b.take(n)
		}
	}
	return n, err
}

func (trc *throttledReadSeekCloser) Seek(offset int64, whence int) (int64, error) {
	return trc.src.Seek(offset, whence)
}

func (trc *throttledReadSeekCloser) Close() error {
	return trc.src.Close()
}
