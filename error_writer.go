package rest

import (
	"io"
)

// ErrorWriter wrap a writer and check error when writing.
// Example:
//     w := NewErrorWriter(writer)
//     w.Write(...)
//     w.Write(...)
//     w.Write(...)
//     if err := w.Error(); err != nil {
//         return err
//     }
//     ... // success process
type ErrorWriter struct {
	w   io.Writer
	err error
}

// NewErrorWriter wrap writer w and return a ErrorWriter.
func NewErrorWriter(w io.Writer) *ErrorWriter {
	return &ErrorWriter{
		w: w,
	}
}

// Write to inner writer.
func (w *ErrorWriter) Write(p []byte) {
	if w.err != nil {
		return
	}
	for len(p) > 0 {
		n, err := w.w.Write(p)
		if err != nil {
			w.err = err
			return
		}
		p = p[n:]
	}
}

// Error return the error if writing failed before.
func (w *ErrorWriter) Error() error {
	return w.err
}
