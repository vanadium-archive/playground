// LimitedWriter is an io.Writer wrapper that limits the total number of bytes
// written to the underlying writer.
//
// All attempted writes count against the limit, regardless of whether they
// succeed.
// Not thread-safe.

package lib

import (
	"errors"
	"io"
	"net/http"
	"sync"
)

var ErrWriteLimitExceeded = errors.New("LimitedWriter: write limit exceeded")

// Initialize using NewLimitedWriter.
type LimitedWriter struct {
	io.Writer
	maxLen           int
	maxLenExceededCb func()
	lenWritten       int
}

func NewLimitedWriter(writer io.Writer, maxLen int, maxLenExceededCb func()) *LimitedWriter {
	return &LimitedWriter{
		Writer:           writer,
		maxLen:           maxLen,
		maxLenExceededCb: maxLenExceededCb,
	}
}

func (t *LimitedWriter) Write(p []byte) (n int, err error) {
	if t.lenWritten+len(p) > t.maxLen {
		t.lenWritten = t.maxLen
		if t.maxLenExceededCb != nil {
			t.maxLenExceededCb()
		}
		return 0, ErrWriteLimitExceeded
	}
	if len(p) == 0 {
		return 0, nil
	}
	t.lenWritten += len(p)
	return t.Writer.Write(p)
}

var _ http.Flusher = (*LimitedWriter)(nil)

func (t *LimitedWriter) Flush() {
	if f, ok := t.Writer.(http.Flusher); ok {
		f.Flush()
	}
}

// Wraps a function to prevent it from executing more than once.
func DoOnce(f func()) func() {
	var once sync.Once
	return func() {
		once.Do(f)
	}
}
