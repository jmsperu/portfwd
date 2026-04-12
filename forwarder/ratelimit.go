package forwarder

import (
	"io"
	"time"
)

// RateLimitedReader wraps an io.Reader with a bytes-per-second rate limit.
type RateLimitedReader struct {
	r         io.Reader
	bytesPerS int
}

func NewRateLimitedReader(r io.Reader, bytesPerSec int) io.Reader {
	if bytesPerSec <= 0 {
		return r
	}
	return &RateLimitedReader{r: r, bytesPerS: bytesPerSec}
}

func (rl *RateLimitedReader) Read(p []byte) (int, error) {
	// Limit read chunk to the rate limit amount
	maxChunk := rl.bytesPerS
	if maxChunk > len(p) {
		maxChunk = len(p)
	}

	n, err := rl.r.Read(p[:maxChunk])
	if n > 0 {
		// Sleep proportionally to how many bytes we just read
		sleepDuration := time.Duration(float64(n) / float64(rl.bytesPerS) * float64(time.Second))
		if sleepDuration > 0 {
			time.Sleep(sleepDuration)
		}
	}
	return n, err
}
