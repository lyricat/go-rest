package rest

import (
	"compress/flate"
	"compress/gzip"
	"io"
)

type Compresser interface {
	Name() string
	Reader(r io.Reader) (io.ReadCloser, error)
	Writer(w io.Writer) (io.WriteCloser, error)
}

// Register a compresser with corresponding name.
func RegisterCompresser(compresser Compresser) {
	compressers[compresser.Name()] = compresser
}

var compressers map[string]Compresser

func init() {
	compressers = make(map[string]Compresser)
	for _, c := range []Compresser{new(GzipCompress), new(DeflateCompress)} {
		RegisterCompresser(c)
	}
}

func getCompresser(name string) (Compresser, bool) {
	ret, ok := compressers[name]
	return ret, ok
}

type GzipCompress struct{}

func (g GzipCompress) Name() string {
	return "gzip"
}

func (g GzipCompress) Reader(r io.Reader) (io.ReadCloser, error) {
	return gzip.NewReader(r)
}

func (g GzipCompress) Writer(w io.Writer) (io.WriteCloser, error) {
	return gzip.NewWriter(w), nil
}

type DeflateCompress struct{}

func (g DeflateCompress) Name() string {
	return "deflate"
}

func (d DeflateCompress) Reader(r io.Reader) (io.ReadCloser, error) {
	return flate.NewReader(r), nil
}

func (d DeflateCompress) Writer(w io.Writer) (io.WriteCloser, error) {
	return flate.NewWriter(w, flate.DefaultCompression)
}
